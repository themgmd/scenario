package scenario

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	tele "gopkg.in/telebot.v3"

	"github.com/themgmd/scenario/mocks"
)

func TestSessionToBase(t *testing.T) {
	type UserData struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	sess := &Session[UserData]{
		ChatID:  123,
		UserID:  456,
		Scene:   "test_scene",
		Step:    1,
		Data:    UserData{Name: "John", Age: 30},
		Updated: time.Now(),
	}

	base, err := sess.toBase()
	require.NoError(t, err)
	assert.Equal(t, int64(123), base.ChatID)
	assert.Equal(t, int64(456), base.UserID)
	assert.Equal(t, SceneName("test_scene"), base.Scene)
	assert.Equal(t, 1, base.Step)
	assert.NotZero(t, base.Updated)

	// Verify data is JSON marshaled correctly
	var data UserData
	err = json.Unmarshal(base.Data, &data)
	require.NoError(t, err)
	assert.Equal(t, "John", data.Name)
	assert.Equal(t, 30, data.Age)
}

func TestFromBase(t *testing.T) {
	type UserData struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	t.Run("normal data", func(t *testing.T) {
		dataJSON := []byte(`{"name":"Alice","age":25}`)
		base := &SessionBase{
			ChatID:  789,
			UserID:  101,
			Scene:   "test",
			Step:    2,
			Data:    dataJSON,
			Updated: time.Now(),
		}

		sess, err := fromBase[UserData](base)
		require.NoError(t, err)
		assert.Equal(t, int64(789), sess.ChatID)
		assert.Equal(t, int64(101), sess.UserID)
		assert.Equal(t, SceneName("test"), sess.Scene)
		assert.Equal(t, 2, sess.Step)
		assert.Equal(t, "Alice", sess.Data.Name)
		assert.Equal(t, 25, sess.Data.Age)
	})

	t.Run("null data", func(t *testing.T) {
		base := &SessionBase{
			ChatID: 1,
			UserID: 2,
			Data:   []byte("null"),
		}

		sess, err := fromBase[UserData](base)
		require.NoError(t, err)
		assert.Equal(t, UserData{}, sess.Data)
	})

	t.Run("empty data", func(t *testing.T) {
		base := &SessionBase{
			ChatID: 1,
			UserID: 2,
			Data:   []byte("{}"),
		}

		sess, err := fromBase[UserData](base)
		require.NoError(t, err)
		assert.Equal(t, UserData{}, sess.Data)
	})

	t.Run("nil base", func(t *testing.T) {
		sess, err := fromBase[UserData](nil)
		require.NoError(t, err)
		assert.NotNil(t, sess)
		assert.Equal(t, UserData{}, sess.Data)
	})
}

func TestGetChatUserIDs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("with message and chat", func(t *testing.T) {
		mockCtx := mocks.NewMockContext(ctrl)
		mockCtx.EXPECT().Sender().Return(&tele.User{ID: 100}).AnyTimes()
		mockCtx.EXPECT().Message().Return(&tele.Message{
			Chat: &tele.Chat{ID: 200},
		}).AnyTimes()

		chatID, userID := getChatUserIDs(mockCtx)
		assert.Equal(t, int64(200), chatID)
		assert.Equal(t, int64(100), userID)
	})

	t.Run("without message", func(t *testing.T) {
		mockCtx := mocks.NewMockContext(ctrl)
		mockCtx.EXPECT().Sender().Return(&tele.User{ID: 100}).AnyTimes()
		mockCtx.EXPECT().Message().Return(nil).AnyTimes()

		chatID, userID := getChatUserIDs(mockCtx)
		assert.Equal(t, int64(0), chatID)
		assert.Equal(t, int64(100), userID)
	})
}

func TestContextDirtyFlag(t *testing.T) {
	type TestData struct {
		Value string
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCtx := mocks.NewMockContext(ctrl)
	mockCtx.EXPECT().Sender().Return(&tele.User{ID: 1}).AnyTimes()
	mockCtx.EXPECT().Message().Return(&tele.Message{
		Chat: &tele.Chat{ID: 2},
	}).AnyTimes()

	scenario := New(nil)
	sess := &Session[TestData]{Data: TestData{Value: "initial"}}
	context := newCtx(scenario, mockCtx, sess)

	assert.False(t, context.isDirty())

	context.markDirty()
	assert.True(t, context.isDirty())
	assert.Nil(t, context.cachedBase) // cache should be invalidated

	context.clearDirty()
	assert.False(t, context.isDirty())
}

func TestContextSetData(t *testing.T) {
	type TestData struct {
		Value string
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCtx := mocks.NewMockContext(ctrl)
	mockCtx.EXPECT().Sender().Return(&tele.User{ID: 1}).AnyTimes()
	mockCtx.EXPECT().Message().Return(&tele.Message{
		Chat: &tele.Chat{ID: 2},
	}).AnyTimes()

	scenario := New(nil)
	sess := &Session[TestData]{Data: TestData{Value: "initial"}}
	context := newCtx(scenario, mockCtx, sess)

	assert.False(t, context.isDirty())

	context.SetData(TestData{Value: "updated"})
	assert.True(t, context.isDirty())
	assert.Equal(t, "updated", context.GetData().Value)
}

func TestContextGetSessionBase(t *testing.T) {
	type TestData struct {
		Value string
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCtx := mocks.NewMockContext(ctrl)
	mockCtx.EXPECT().Sender().Return(&tele.User{ID: 1}).AnyTimes()
	mockCtx.EXPECT().Message().Return(&tele.Message{
		Chat: &tele.Chat{ID: 2},
	}).AnyTimes()

	scenario := New(nil)
	sess := &Session[TestData]{Data: TestData{Value: "test"}}
	context := newCtx(scenario, mockCtx, sess)

	// First call should create and cache base
	base1, err := context.getSessionBase()
	require.NoError(t, err)
	assert.NotNil(t, base1)
	assert.NotNil(t, context.cachedBase)

	// Second call should return cached base (if not dirty)
	base2, err := context.getSessionBase()
	require.NoError(t, err)
	assert.Equal(t, base1, base2)

	// After marking dirty, cache should be invalidated
	context.markDirty()
	base3, err := context.getSessionBase()
	require.NoError(t, err)
	assert.NotEqual(t, base1, base3) // new base created
}

func TestContextSetSessionBase(t *testing.T) {
	type TestData struct {
		Value string
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCtx := mocks.NewMockContext(ctrl)
	mockCtx.EXPECT().Sender().Return(&tele.User{ID: 1}).AnyTimes()
	mockCtx.EXPECT().Message().Return(&tele.Message{
		Chat: &tele.Chat{ID: 2},
	}).AnyTimes()

	scenario := New(nil)
	sess := &Session[TestData]{Data: TestData{Value: "old"}}
	context := newCtx(scenario, mockCtx, sess)

	base := &SessionBase{
		ChatID: 100,
		UserID: 200,
		Data:   []byte(`{"Value":"new"}`),
	}

	err := context.setSessionBase(base)
	require.NoError(t, err)
	assert.Equal(t, "new", context.GetData().Value)
	assert.False(t, context.isDirty())        // should be reset after loading
	assert.Equal(t, base, context.cachedBase) // should cache the base
}

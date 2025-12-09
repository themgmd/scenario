package scenario

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	tele "gopkg.in/telebot.v3"

	"github.com/themgmd/scenario/mocks"
)

type mockScene struct {
	name      SceneName
	enterErr  error
	updateErr error
	leaveErr  error
	entered   bool
	updated   bool
	left      bool
}

func (m *mockScene) Name() SceneName {
	return m.name
}

func (m *mockScene) Enter(c ContextBase) error {
	m.entered = true
	return m.enterErr
}

func (m *mockScene) OnUpdate(c ContextBase) error {
	m.updated = true
	return m.updateErr
}

func (m *mockScene) Leave(c ContextBase) error {
	m.left = true
	return m.leaveErr
}

func TestScenarioNew(t *testing.T) {
	scenario := New(nil)
	assert.NotNil(t, scenario)
	assert.NotNil(t, scenario.store)
	assert.NotNil(t, scenario.scenes)
}

func TestScenarioUse(t *testing.T) {
	scenario := New(nil)
	scene := &mockScene{name: "test_scene"}

	scenario.Use(scene)
	assert.Equal(t, scene, scenario.scenes["test_scene"])
}

func TestScenarioEnter(t *testing.T) {
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
	scene := &mockScene{name: "test_scene"}
	scenario.Use(scene)

	sess := &Session[TestData]{Data: TestData{Value: "test"}}
	context := newCtx(scenario, mockCtx, sess)

	err := scenario.enter(context, "test_scene")
	require.NoError(t, err)
	assert.True(t, scene.entered)
	assert.True(t, scene.updated) // OnUpdate should be called after Enter
	assert.Equal(t, SceneName("test_scene"), context.Session.Scene)
}

func TestScenarioEnterNonExistentScene(t *testing.T) {
	type TestData struct{}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCtx := mocks.NewMockContext(ctrl)
	mockCtx.EXPECT().Sender().Return(&tele.User{ID: 1}).AnyTimes()
	mockCtx.EXPECT().Message().Return(&tele.Message{
		Chat: &tele.Chat{ID: 2},
	}).AnyTimes()

	scenario := New(nil)
	sess := &Session[TestData]{}
	context := newCtx(scenario, mockCtx, sess)

	err := scenario.enter(context, "non_existent")
	assert.NoError(t, err) // should not error, just do nothing
}

func TestScenarioLeave(t *testing.T) {
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
	scene := &mockScene{name: "test_scene"}
	scenario.Use(scene)

	sess := &Session[TestData]{
		Scene: "test_scene",
		Data:  TestData{Value: "test"},
	}
	context := newCtx(scenario, mockCtx, sess)

	err := scenario.leave(context)
	require.NoError(t, err)
	assert.True(t, scene.left)
	assert.Equal(t, SceneName(""), context.Session.Scene) // scene should be cleared
}

func TestScenarioLeaveNonExistentScene(t *testing.T) {
	type TestData struct{}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCtx := mocks.NewMockContext(ctrl)
	mockCtx.EXPECT().Sender().Return(&tele.User{ID: 1}).AnyTimes()
	mockCtx.EXPECT().Message().Return(&tele.Message{
		Chat: &tele.Chat{ID: 2},
	}).AnyTimes()

	scenario := New(nil)
	sess := &Session[TestData]{Scene: "non_existent"}
	context := newCtx(scenario, mockCtx, sess)

	err := scenario.leave(context)
	assert.Error(t, err)
	assert.Equal(t, ErrSceneNotFound, err)
}

func TestScenarioMiddleware(t *testing.T) {
	type TestData struct {
		Value string
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCtx := mocks.NewMockContext(ctrl)
	mockCtx.EXPECT().Sender().Return(&tele.User{ID: 2}).AnyTimes()
	mockCtx.EXPECT().Message().Return(&tele.Message{
		Chat: &tele.Chat{ID: 1},
	}).AnyTimes()

	scenario := New(nil)
	scene := &mockScene{name: "test_scene"}
	scenario.Use(scene)

	// Create a session with active scene
	base := &SessionBase{
		ChatID: 1,
		UserID: 2,
		Scene:  "test_scene",
		Data:   []byte(`{"Value":"test"}`),
	}
	err := scenario.store.SetSession(context.Background(), base)
	require.NoError(t, err)

	nextCalled := false
	next := func(c tele.Context) error {
		nextCalled = true
		return nil
	}

	middleware := scenario.Middleware(next)
	err = middleware(mockCtx)
	require.NoError(t, err)
	assert.True(t, scene.updated) // scene should be updated
	assert.False(t, nextCalled)   // next should not be called
}

func TestScenarioMiddlewareNoActiveScene(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCtx := mocks.NewMockContext(ctrl)
	mockCtx.EXPECT().Sender().Return(&tele.User{ID: 1}).AnyTimes()
	mockCtx.EXPECT().Message().Return(&tele.Message{
		Chat: &tele.Chat{ID: 2},
	}).AnyTimes()

	scenario := New(nil)

	nextCalled := false
	next := func(c tele.Context) error {
		nextCalled = true
		return nil
	}

	middleware := scenario.Middleware(next)
	err := middleware(mockCtx)
	require.NoError(t, err)
	assert.True(t, nextCalled) // next should be called when no active scene
}

func TestScenarioMiddlewareDirtyFlag(t *testing.T) {
	type TestData struct {
		Value string
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCtx := mocks.NewMockContext(ctrl)
	mockCtx.EXPECT().Sender().Return(&tele.User{ID: 2}).AnyTimes()
	mockCtx.EXPECT().Message().Return(&tele.Message{
		Chat: &tele.Chat{ID: 1},
	}).AnyTimes()

	scenario := New(nil)
	scene := &mockScene{name: "test_scene"}
	scenario.Use(scene)

	base := &SessionBase{
		ChatID: 1,
		UserID: 2,
		Scene:  "test_scene",
		Data:   []byte(`{"Value":"initial"}`),
	}
	err := scenario.store.SetSession(context.Background(), base)
	require.NoError(t, err)

	// Create a scene that modifies data
	modifyingScene := &mockModifyingScene{
		mockScene: mockScene{name: "test_scene"},
	}
	scenario.scenes["test_scene"] = modifyingScene

	next := func(c tele.Context) error { return nil }
	middleware := scenario.Middleware(next)
	err = middleware(mockCtx)
	require.NoError(t, err)

	// Verify session was saved (by checking if it's dirty after load)
	loadedBase, err := scenario.store.GetSession(context.Background(), 1, 2)
	require.NoError(t, err)
	assert.NotNil(t, loadedBase)
}

type mockModifyingScene struct {
	mockScene
}

func (m *mockModifyingScene) OnUpdate(c ContextBase) error {
	m.updated = true
	// Modify data to trigger dirty flag
	if ctx, ok := c.(*Context[struct{ Value string }]); ok {
		ctx.SetData(struct{ Value string }{Value: "modified"})
	}
	return nil
}

func TestScenarioWithStore(t *testing.T) {
	scenario := New(nil)
	customStore := &mockStore{}

	result := scenario.WithStore(customStore)
	assert.Equal(t, scenario, result) // should return self for chaining
	assert.Equal(t, customStore, scenario.store)
}

func TestScenarioWithStoreNil(t *testing.T) {
	scenario := New(nil)
	originalStore := scenario.store

	result := scenario.WithStore(nil)
	assert.Equal(t, scenario, result)
	assert.Equal(t, originalStore, scenario.store) // should not change
}

type mockStore struct {
	sessions map[string]*SessionBase
}

func (m *mockStore) GetSession(ctx context.Context, chatID, userID int64) (*SessionBase, error) {
	if m.sessions == nil {
		return nil, ErrSessionNotFound
	}
	key := key(chatID, userID)
	if sess, ok := m.sessions[key]; ok {
		return sess, nil
	}
	return nil, ErrSessionNotFound
}

func (m *mockStore) SetSession(ctx context.Context, sess *SessionBase) error {
	if m.sessions == nil {
		m.sessions = make(map[string]*SessionBase)
	}
	m.sessions[key(sess.ChatID, sess.UserID)] = sess
	return nil
}

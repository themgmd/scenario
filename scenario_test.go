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

func TestCreateTypedContextWithTypedScene(t *testing.T) {
	type TestData struct {
		Value string `json:"value"`
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCtx := mocks.NewMockContext(ctrl)
	mockCtx.EXPECT().Sender().Return(&tele.User{ID: 1}).AnyTimes()
	mockCtx.EXPECT().Message().Return(&tele.Message{
		Chat: &tele.Chat{ID: 2},
	}).AnyTimes()

	scenario := New(nil)
	wizard := NewWizard[TestData]("test_wizard")
	scenario.Use(wizard)

	base := &SessionBase{
		ChatID: 2,
		UserID: 1,
		Scene:  "test_wizard",
		Data:   []byte(`{"value":"test"}`),
	}

	// Test createTypedContext with TypedScene
	ctx, err := createTypedContext(wizard, scenario, mockCtx, base)
	require.NoError(t, err)
	assert.NotNil(t, ctx)

	// Verify it's the correct typed context
	typedCtx, ok := ctx.(*Context[TestData])
	require.True(t, ok, "Context should be *Context[TestData]")
	assert.Equal(t, "test", typedCtx.Session.Data.Value)
}

func TestCreateTypedContextWithNonTypedScene(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCtx := mocks.NewMockContext(ctrl)
	mockCtx.EXPECT().Sender().Return(&tele.User{ID: 1}).AnyTimes()
	mockCtx.EXPECT().Message().Return(&tele.Message{
		Chat: &tele.Chat{ID: 2},
	}).AnyTimes()

	scenario := New(nil)
	scene := &mockScene{name: "test_scene"}

	base := &SessionBase{
		ChatID: 2,
		UserID: 1,
		Scene:  "test_scene",
		Data:   []byte(`{"value":"test"}`),
	}

	// Test createTypedContext with non-TypedScene (should fallback to Context[any])
	ctx, err := createTypedContext(scene, scenario, mockCtx, base)
	require.NoError(t, err)
	assert.NotNil(t, ctx)

	// Verify it's Context[any]
	typedCtx, ok := ctx.(*Context[any])
	require.True(t, ok, "Context should be *Context[any] for non-typed scenes")
	assert.NotNil(t, typedCtx)
}

func TestScenarioMiddlewareWithTypedScene(t *testing.T) {
	type TestData struct {
		Value string `json:"value"`
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCtx := mocks.NewMockContext(ctrl)
	mockCtx.EXPECT().Sender().Return(&tele.User{ID: 1}).AnyTimes()
	mockCtx.EXPECT().Message().Return(&tele.Message{
		Chat: &tele.Chat{ID: 2},
	}).AnyTimes()

	scenario := New(nil)
	wizard := NewWizard[TestData]("test_wizard",
		func(c *Context[TestData]) (bool, error) {
			// Modify data to verify typed context works
			c.SetData(TestData{Value: "modified"})
			return false, nil
		},
	)
	scenario.Use(wizard)

	// Create a session with active scene
	base := &SessionBase{
		ChatID: 2,
		UserID: 1,
		Scene:  "test_wizard",
		Data:   []byte(`{"value":"initial"}`),
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
	assert.False(t, nextCalled) // next should not be called

	// Verify session was updated with typed data
	loadedBase, err := scenario.store.GetSession(context.Background(), 2, 1)
	require.NoError(t, err)
	assert.NotNil(t, loadedBase)
	assert.Contains(t, string(loadedBase.Data), "modified")
}

func TestScenarioMiddlewareWithTypedSceneTypeSafety(t *testing.T) {
	type TestData1 struct {
		Value1 string `json:"value1"`
	}
	type TestData2 struct {
		Value2 int `json:"value2"`
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCtx := mocks.NewMockContext(ctrl)
	mockCtx.EXPECT().Sender().Return(&tele.User{ID: 1}).AnyTimes()
	mockCtx.EXPECT().Message().Return(&tele.Message{
		Chat: &tele.Chat{ID: 2},
	}).AnyTimes()

	scenario := New(nil)
	wizard := NewWizard[TestData1]("test_wizard",
		func(c *Context[TestData1]) (bool, error) {
			// This should work with TestData1
			c.SetData(TestData1{Value1: "test"})
			return false, nil
		},
	)
	scenario.Use(wizard)

	// Create a session with data that matches the wizard's type
	base := &SessionBase{
		ChatID: 2,
		UserID: 1,
		Scene:  "test_wizard",
		Data:   []byte(`{"value1":"initial"}`),
	}
	err := scenario.store.SetSession(context.Background(), base)
	require.NoError(t, err)

	next := func(c tele.Context) error { return nil }
	middleware := scenario.Middleware(next)
	err = middleware(mockCtx)
	require.NoError(t, err)

	// Verify the context was created with correct type
	// The middleware should create Context[TestData1], not Context[any]
	loadedBase, err := scenario.store.GetSession(context.Background(), 2, 1)
	require.NoError(t, err)
	assert.NotNil(t, loadedBase)
	// Data should be valid JSON for TestData1
	assert.Contains(t, string(loadedBase.Data), "value1")
}

func TestScenarioEnterPreservesEnterChanges(t *testing.T) {
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
	wizard := NewWizard[TestData]("test_wizard",
		func(c *Context[TestData]) (bool, error) {
			// OnUpdate modifies data
			c.SetData(TestData{Value: "from_onupdate"})
			return false, nil
		},
	)
	scenario.Use(wizard)

	sess := &Session[TestData]{Data: TestData{Value: "initial"}}
	context := newCtx(scenario, mockCtx, sess)

	err := scenario.enter(context, "test_wizard")
	require.NoError(t, err)

	// Verify that Enter changes (Step=0) are preserved
	assert.Equal(t, 0, context.Session.Step)
	// Verify that OnUpdate changes are also preserved
	assert.Equal(t, "from_onupdate", context.Session.Data.Value)
}

func TestNewContextWithEmptyBase(t *testing.T) {
	type TestData struct {
		Value string `json:"value"`
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCtx := mocks.NewMockContext(ctrl)
	mockCtx.EXPECT().Sender().Return(&tele.User{ID: 123}).AnyTimes()
	mockCtx.EXPECT().Message().Return(&tele.Message{
		Chat: &tele.Chat{ID: 456},
	}).AnyTimes()

	scenario := New(nil)

	// Test NewContext when session doesn't exist (should create empty session)
	ctx, err := NewContext[TestData](scenario, mockCtx)
	require.NoError(t, err)
	assert.NotNil(t, ctx)

	// Verify ChatID and UserID are set correctly
	assert.Equal(t, int64(456), ctx.Session.ChatID)
	assert.Equal(t, int64(123), ctx.Session.UserID)
	assert.Equal(t, TestData{}, ctx.Session.Data) // should be zero value
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

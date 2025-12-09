package scenario

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	tele "gopkg.in/telebot.v3"

	"github.com/themgmd/scenario/mocks"
)

func TestWizardSceneName(t *testing.T) {
	type TestData struct{}
	wizard := NewWizard[TestData]("test_wizard")
	assert.Equal(t, SceneName("test_wizard"), wizard.Name())
}

func TestWizardSceneEnter(t *testing.T) {
	type TestData struct {
		Step int
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCtx := mocks.NewMockContext(ctrl)
	mockCtx.EXPECT().Sender().Return(&tele.User{ID: 1}).AnyTimes()
	mockCtx.EXPECT().Message().Return(&tele.Message{
		Chat: &tele.Chat{ID: 2},
	}).AnyTimes()

	scenario := New(nil)
	sess := &Session[TestData]{Step: -1}
	context := newCtx(scenario, mockCtx, sess)

	wizard := NewWizard[TestData]("test_wizard")
	err := wizard.Enter(context)
	require.NoError(t, err)
	assert.Equal(t, 0, context.Session.Step)
	assert.True(t, context.isDirty())
}

func TestWizardSceneOnUpdate(t *testing.T) {
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
			return true, nil // advance to next step
		},
		func(c *Context[TestData]) (bool, error) {
			return false, nil // second step
		},
	)
	scenario.Use(wizard)

	sess := &Session[TestData]{Step: 0}
	context := newCtx(scenario, mockCtx, sess)

	err := wizard.OnUpdate(context)
	require.NoError(t, err)
	assert.Equal(t, 1, context.Session.Step) // should advance
	assert.True(t, context.isDirty())
}

func TestWizardSceneOnUpdateNoAdvance(t *testing.T) {
	type TestData struct{}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCtx := mocks.NewMockContext(ctrl)
	mockCtx.EXPECT().Sender().Return(&tele.User{ID: 1}).AnyTimes()
	mockCtx.EXPECT().Message().Return(&tele.Message{
		Chat: &tele.Chat{ID: 2},
	}).AnyTimes()

	scenario := New(nil)
	sess := &Session[TestData]{Step: 0}
	context := newCtx(scenario, mockCtx, sess)

	wizard := NewWizard[TestData]("test_wizard",
		func(c *Context[TestData]) (bool, error) {
			return false, nil // don't advance
		},
	)

	err := wizard.OnUpdate(context)
	require.NoError(t, err)
	assert.Equal(t, 0, context.Session.Step) // should stay at 0
}

func TestWizardSceneOnUpdateLastStep(t *testing.T) {
	type TestData struct{}
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
			return true, nil // advance
		},
	)
	scenario.Use(wizard)

	sess := &Session[TestData]{
		Step:  0,
		Scene: "test_wizard", // Set scene so Leave can find it
	}
	context := newCtx(scenario, mockCtx, sess)

	err := wizard.OnUpdate(context)
	require.NoError(t, err)
	// Since there's only one step and we advance, wizard should leave
	// After leave, Scene should be cleared and Step should be -1
	// But Leave() reloads session from base, so we need to check after reload
	assert.Equal(t, SceneName(""), context.Session.Scene) // Scene cleared after leave
	assert.Equal(t, -1, context.Session.Step)             // Step set to -1
}

func TestWizardSceneOnUpdateCancelCommand(t *testing.T) {
	type TestData struct{}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCtx := mocks.NewMockContext(ctrl)
	mockCtx.EXPECT().Sender().Return(&tele.User{ID: 1}).AnyTimes()
	mockCtx.EXPECT().Message().Return(&tele.Message{
		Text: "/cancel",
		Chat: &tele.Chat{ID: 2},
	}).AnyTimes()
	mockCtx.EXPECT().Reply("Отменено").Return(nil).AnyTimes()

	scenario := New(nil)
	wizard := NewWizard[TestData]("test_wizard",
		func(c *Context[TestData]) (bool, error) {
			return false, nil
		},
	)
	scenario.Use(wizard)

	sess := &Session[TestData]{
		Step:  0,
		Scene: "test_wizard", // Set scene so Leave can find it
	}
	context := newCtx(scenario, mockCtx, sess)

	err := wizard.OnUpdate(context)
	require.NoError(t, err)
	// Should trigger leave, Scene should be cleared and Step should be -1
	assert.Equal(t, SceneName(""), context.Session.Scene) // Scene cleared after leave
	assert.Equal(t, -1, context.Session.Step)             // Step set to -1
}

func TestWizardSceneLeave(t *testing.T) {
	type TestData struct{}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCtx := mocks.NewMockContext(ctrl)
	mockCtx.EXPECT().Sender().Return(&tele.User{ID: 1}).AnyTimes()
	mockCtx.EXPECT().Message().Return(&tele.Message{
		Chat: &tele.Chat{ID: 2},
	}).AnyTimes()

	scenario := New(nil)
	sess := &Session[TestData]{Step: 5}
	context := newCtx(scenario, mockCtx, sess)

	wizard := NewWizard[TestData]("test_wizard")
	err := wizard.Leave(context)
	require.NoError(t, err)
	assert.Equal(t, -1, context.Session.Step)
	assert.True(t, context.isDirty())
}

func TestWizardSceneTypeMismatch(t *testing.T) {
	type Data1 struct{ Value string }
	type Data2 struct{ Value int }

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCtx := mocks.NewMockContext(ctrl)
	mockCtx.EXPECT().Sender().Return(&tele.User{ID: 1}).AnyTimes()
	mockCtx.EXPECT().Message().Return(&tele.Message{
		Chat: &tele.Chat{ID: 2},
	}).AnyTimes()

	scenario := New(nil)
	sess := &Session[Data1]{}
	context := newCtx(scenario, mockCtx, sess)

	wizard := NewWizard[Data2]("test_wizard")
	err := wizard.Enter(context)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Data2")
}

func TestWizardSceneStepError(t *testing.T) {
	type TestData struct{}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCtx := mocks.NewMockContext(ctrl)
	mockCtx.EXPECT().Sender().Return(&tele.User{ID: 1}).AnyTimes()
	mockCtx.EXPECT().Message().Return(&tele.Message{
		Chat: &tele.Chat{ID: 2},
	}).AnyTimes()

	scenario := New(nil)
	sess := &Session[TestData]{Step: 0}
	context := newCtx(scenario, mockCtx, sess)

	expectedErr := errors.New("step error")
	wizard := NewWizard[TestData]("test_wizard",
		func(c *Context[TestData]) (bool, error) {
			return false, expectedErr
		},
	)

	err := wizard.OnUpdate(context)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

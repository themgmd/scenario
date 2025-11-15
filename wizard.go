package scenario

import (
	"strings"

	_ "gopkg.in/telebot.v3"
)

// WizardStep is a step handler returning whether to advance to next step.
type WizardStep func(*Context) (advance bool, err error)

// WizardScene .
type WizardScene struct {
	name  SceneName
	steps []WizardStep
}

// NewWizard .
func NewWizard(name SceneName, steps ...WizardStep) *WizardScene {
	return &WizardScene{name: name, steps: steps}
}

// Name .
func (w *WizardScene) Name() SceneName { return w.name }

// Enter .
func (w *WizardScene) Enter(c *Context) error {
	c.Session.Step = 0
	return nil
}

// OnUpdate .
func (w *WizardScene) OnUpdate(c *Context) error {
	idx := c.Session.Step
	if idx < 0 || idx >= len(w.steps) {
		return c.Leave()
	}

	// special command handling inside scenes
	if m := c.Message(); m != nil {
		if strings.EqualFold(m.Text, "/cancel") {
			_ = c.Reply("Отменено")
			return c.Leave()
		}
	}

	advance, err := w.steps[idx](c)
	if err != nil {
		return err
	}
	if advance {
		idx++
		if idx >= len(w.steps) {
			return c.Leave()
		}
		c.Session.Step = idx
	}
	return nil
}

// Leave .
func (w *WizardScene) Leave(c *Context) error {
	c.Session.Step = -1
	return nil
}

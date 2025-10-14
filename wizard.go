package scenario

import (
	"context"
	"strings"
	"time"

	_ "gopkg.in/telebot.v3"
)

// WizardStep is a step handler returning whether to advance to next step.
type WizardStep func(*Ctx) (advance bool, err error)

type WizardScene struct {
	name  string
	steps []WizardStep
}

func NewWizard(name string, steps ...WizardStep) *WizardScene {
	return &WizardScene{name: name, steps: steps}
}

func (w *WizardScene) Name() string { return w.name }

func (w *WizardScene) Enter(c *Ctx, _ ...any) error {
	c.Session.Step = 0
	return nil
}

func (w *WizardScene) OnUpdate(c *Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	idx := c.Session.Step
	if idx < 0 || idx >= len(w.steps) {
		return c.Leave(ctx)
	}

	// special command handling inside scenes
	if m := c.Message(); m != nil {
		if strings.EqualFold(m.Text, "/cancel") {
			_ = c.Reply("Отменено")
			return c.Leave(ctx)
		}
	}

	advance, err := w.steps[idx](c)
	if err != nil {
		return err
	}
	if advance {
		idx++
		if idx >= len(w.steps) {
			return c.Leave(ctx)
		}
		c.Session.Step = idx
	}
	return nil
}

func (w *WizardScene) Leave(c *Ctx) error {
	c.Session.Step = -1
	return nil
}

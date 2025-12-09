package scenario

import (
	"fmt"
	"strings"

	_ "gopkg.in/telebot.v3"
)

// WizardStep is a step handler returning whether to advance to next step.
// T is the type of data stored in the session.
type WizardStep[T any] func(*Context[T]) (advance bool, err error)

// WizardScene is a scene that manages a sequence of steps (wizard pattern).
// T is the type of data stored in the session.
type WizardScene[T any] struct {
	name  SceneName
	steps []WizardStep[T]
}

// NewWizard creates a new wizard scene with typed steps.
// T is the type of data stored in the session.
func NewWizard[T any](name SceneName, steps ...WizardStep[T]) *WizardScene[T] {
	return &WizardScene[T]{name: name, steps: steps}
}

// Name returns the scene name.
func (w *WizardScene[T]) Name() SceneName { return w.name }

// Enter initializes the wizard by setting step to 0.
func (w *WizardScene[T]) Enter(c ContextBase) error {
	ctx, ok := c.(*Context[T])
	if !ok {
		return fmt.Errorf("WizardScene[%T]: expected Context[%T], got %T", *new(T), *new(T), c)
	}
	ctx.Session.Step = 0
	ctx.markDirty()
	return nil
}

// OnUpdate processes the current step and advances if needed.
func (w *WizardScene[T]) OnUpdate(c ContextBase) error {
	ctx, ok := c.(*Context[T])
	if !ok {
		return fmt.Errorf("WizardScene[%T]: expected Context[%T], got %T", *new(T), *new(T), c)
	}

	idx := ctx.Session.Step
	if idx < 0 || idx >= len(w.steps) {
		return ctx.Leave()
	}

	// special command handling inside scenes
	if m := ctx.Message(); m != nil {
		if strings.EqualFold(m.Text, "/cancel") {
			_ = ctx.Reply("Отменено")
			return ctx.Leave()
		}
	}

	advance, err := w.steps[idx](ctx)
	if err != nil {
		return err
	}
	if advance {
		idx++
		if idx >= len(w.steps) {
			return ctx.Leave()
		}
		// Update step directly without conversion
		ctx.Session.Step = idx
		ctx.markDirty()
	}
	return nil
}

// Leave cleans up the wizard by setting step to -1.
func (w *WizardScene[T]) Leave(c ContextBase) error {
	ctx, ok := c.(*Context[T])
	if !ok {
		return fmt.Errorf("WizardScene[%T]: expected Context[%T], got %T", *new(T), *new(T), c)
	}
	// Update step directly without conversion
	ctx.Session.Step = -1
	ctx.markDirty()
	return nil
}

package scenario

import (
	"context"
	"errors"
	"fmt"
	"time"

	tele "gopkg.in/telebot.v3"
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSceneNotFound   = errors.New("scene not found")
)

// SceneName .
type SceneName string

// Handler is a function to process updates inside a scene.
type Handler func(ContextBase) error

// Scene defines a simple lifecycle similar to grammy scenes.
type Scene interface {
	Name() SceneName
	Enter(ContextBase) error
	OnUpdate(ContextBase) error
	Leave(ContextBase) error
}

// TypedScene is a scene that can create a typed context from SessionBase.
// This allows middleware to create the correct Context[T] type.
type TypedScene interface {
	Scene
	// CreateContext creates a typed Context[T] from SessionBase.
	// The returned ContextBase should be of type *Context[T] where T matches the scene's data type.
	CreateContext(scenario *Scenario, c tele.Context, base *SessionBase) (ContextBase, error)
}

// Scenario routes updates to scenes, stores session and current scene.
type Scenario struct {
	bot    *tele.Bot
	store  Store
	scenes map[SceneName]Scene
}

// New .
func New(bot *tele.Bot) *Scenario {
	return &Scenario{
		bot:    bot,
		store:  newMemoryStore(),
		scenes: make(map[SceneName]Scene),
	}
}

// WithStore replaces the default in-memory store with a custom Store (e.g., DB-backed).
func (s *Scenario) WithStore(store Store) *Scenario {
	if store != nil {
		s.store = store
	}
	return s
}

func (s *Scenario) Use(sc Scene) *Scenario {
	s.scenes[sc.Name()] = sc
	return s
}

// createTypedContext creates a typed Context[T] based on the scene type.
// If the scene implements TypedScene, it uses CreateContext method.
// Otherwise, falls back to Context[any].
func createTypedContext(scene Scene, scenario *Scenario, c tele.Context, base *SessionBase) (ContextBase, error) {
	// Check if scene implements TypedScene interface
	if typedScene, ok := scene.(TypedScene); ok {
		return typedScene.CreateContext(scenario, c, base)
	}

	// Fallback to Context[any] for non-typed scenes
	sess, err := fromBase[any](base)
	if err != nil {
		return nil, fmt.Errorf("fromBase[any]: %w", err)
	}
	return newCtx(scenario, c, sess), nil
}

// Middleware returns a telebot middleware that injects scene context and dispatches to active scene.
// This middleware creates a typed Context[T] based on the scene's type parameter.
func (s *Scenario) Middleware(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cid, uid := getChatUserIDs(c)
		base, err := s.store.GetSession(ctx, cid, uid)
		if err != nil && !errors.Is(err, ErrSessionNotFound) {
			return err
		}
		if base == nil {
			base = &SessionBase{}
		}

		sc, ok := s.scenes[base.Scene]
		if !ok || sc == nil || base.Scene == "" {
			// Fallback to next handlers if no active scene
			return next(c)
		}

		// Create typed context based on scene type
		sceneCtx, err := createTypedContext(sc, s, c, base)
		if err != nil {
			return fmt.Errorf("createTypedContext: %w", err)
		}

		// Dispatch to current scene
		if err = sc.OnUpdate(sceneCtx); err != nil {
			return err
		}

		// persist session changes only if dirty
		if sceneCtx.isDirty() {
			base, err = sceneCtx.getSessionBase()
			if err != nil {
				return fmt.Errorf("getSessionBase: %w", err)
			}
			err = s.store.SetSession(ctx, base)
			if err != nil {
				return fmt.Errorf("store.SetSession: %w", err)
			}
			sceneCtx.clearDirty()
		}
		return nil
	}
}

// enter sets current scene and calls Enter.
func (s *Scenario) enter(c ContextBase, scene SceneName) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sc, ok := s.scenes[scene]
	if !ok {
		return nil
	}

	if err := sc.Enter(c); err != nil {
		return err
	}

	// Update scene directly in Session[T] to avoid double conversion
	// Get base once, update scene, save, then set back
	base, err := c.getSessionBase()
	if err != nil {
		return fmt.Errorf("getSessionBase: %w", err)
	}
	base.Scene = scene
	c.markDirty()

	// Save scene change
	err = s.store.SetSession(ctx, base)
	if err != nil {
		return fmt.Errorf("store.SetSession: %w", err)
	}
	if err := c.setSessionBase(base); err != nil {
		return fmt.Errorf("setSessionBase: %w", err)
	}

	// Immediately trigger the first step to send initial message
	if err = sc.OnUpdate(c); err != nil {
		return err
	}

	// Save if dirty after OnUpdate (only one conversion needed)
	if c.isDirty() {
		base, err = c.getSessionBase()
		if err != nil {
			return fmt.Errorf("getSessionBase: %w", err)
		}
		err = s.store.SetSession(ctx, base)
		if err != nil {
			return fmt.Errorf("store.SetSession: %w", err)
		}
		c.clearDirty()
	}

	return nil
}

// leave clears current scene and calls Leave if any.
func (s *Scenario) leave(c ContextBase) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	base, err := c.getSessionBase()
	if err != nil {
		return fmt.Errorf("getSessionBase: %w", err)
	}

	sc, ok := s.scenes[base.Scene]
	if !ok {
		return ErrSceneNotFound
	}

	err = sc.Leave(c)
	if err != nil {
		return fmt.Errorf("sc.Leave: %w", err)
	}

	// Get updated base after Leave() (which may have modified session data)
	base, err = c.getSessionBase()
	if err != nil {
		return fmt.Errorf("getSessionBase after Leave: %w", err)
	}

	// Clear scene and save (reuse base to avoid double conversion)
	base.Scene = ""
	c.markDirty()
	err = s.store.SetSession(ctx, base)
	if err != nil {
		return fmt.Errorf("store.RemoveScene: %w", err)
	}
	if err := c.setSessionBase(base); err != nil {
		return fmt.Errorf("setSessionBase: %w", err)
	}
	c.clearDirty()

	return nil
}

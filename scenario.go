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
type Handler func(*Context) error

// Scene defines a simple lifecycle similar to grammy scenes.
type Scene interface {
	Name() SceneName
	Enter(*Context) error
	OnUpdate(*Context) error
	Leave(*Context) error
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

// Middleware returns a telebot middleware that injects scene context and dispatches to active scene.
func (s *Scenario) Middleware(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		uid := c.Sender().ID
		var cid int64
		if m := c.Message(); m != nil && m.Chat != nil {
			cid = m.Chat.ID
		}

		sess, err := s.store.GetSession(ctx, cid, uid)
		if err != nil && !errors.Is(err, ErrSessionNotFound) {
			return err
		}

		sc, ok := s.scenes[sess.Scene]
		sceneCtx := newCtx(s, c, sess)

		if ok && sc != nil {
			// Dispatch to current scene
			if err = sc.OnUpdate(sceneCtx); err != nil {
				return err
			}
			// persist session changes after each handled update
			err = s.store.SetSession(ctx, sceneCtx.Session)
			if err != nil {
				return fmt.Errorf("store.SetSession: %w", err)
			}
			return nil
		}

		// Fallback to next handlers if no active scene
		return next(c)
	}
}

// enter sets current scene and calls Enter.
func (s *Scenario) enter(c *Context, scene SceneName) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sc, ok := s.scenes[scene]
	if !ok {
		return nil
	}

	if err := sc.Enter(c); err != nil {
		return err
	}

	c.Session.Scene = scene
	err := s.store.SetSession(ctx, c.Session)
	if err != nil {
		return fmt.Errorf("store.SetSession: %w", err)
	}

	// Immediately trigger the first step to send initial message
	if err = sc.OnUpdate(c); err != nil {
		return err
	}

	return nil
}

// leave clears current scene and calls Leave if any.
func (s *Scenario) leave(c *Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sc, ok := s.scenes[c.Session.Scene]
	if !ok {
		return ErrSceneNotFound
	}

	err := sc.Leave(c)
	if err != nil {
		return fmt.Errorf("sc.Leave: %w", err)
	}

	c.Session.Scene = ""
	err = s.store.SetSession(ctx, c.Session)
	if err != nil {
		return fmt.Errorf("store.RemoveScene: %w", err)
	}

	return nil
}

package scenario

import (
	"context"
	tele "gopkg.in/telebot.v3"
	"time"
)

// Handler is a function to process updates inside a scene.
type Handler func(*Ctx) error

// Scene defines a simple lifecycle similar to grammy scenes.
type Scene interface {
	Name() string
	Enter(*Ctx, ...any) error
	OnUpdate(*Ctx) error
	Leave(*Ctx) error
}

// Scenario routes updates to scenes, stores session and current scene.
type Scenario struct {
	bot    *tele.Bot
	store  Store
	scenes map[string]Scene
}

// NewScenario .
func NewScenario(bot *tele.Bot) *Scenario {
	return &Scenario{
		bot:    bot,
		store:  newMemoryStore(),
		scenes: make(map[string]Scene),
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
		sess := s.store.GetSession(ctx, cid, uid)
		scName := s.store.GetScene(ctx, cid, uid)
		sc, ok := s.scenes[scName]
		sceneCtx := newCtx(s, c, sess)

		if ok && sc != nil {
			// Dispatch to current scene
			if err := sc.OnUpdate(sceneCtx); err != nil {
				return err
			}
			// persist session changes after each handled update
			s.store.SetSession(ctx, cid, uid, sceneCtx.Session)
			return nil
		}

		// Fallback to next handlers if no active scene
		return next(c)
	}
}

// enter sets current scene and calls Enter.
func (s *Scenario) enter(c *Ctx, scene string, args ...any) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s.store.SetScene(ctx, c.chatID, c.userID, scene)
	if sc, ok := s.scenes[scene]; ok {
		if err := sc.Enter(c, args...); err != nil {
			return err
		}
		// persist session right after enter (e.g., __step = 0)
		s.store.SetSession(ctx, c.chatID, c.userID, c.Session)
		return nil
	}
	return nil
}

// leave clears current scene and calls Leave if any.
func (s *Scenario) leave(userID, chatID int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	scName := s.store.GetScene(ctx, chatID, userID)
	if sc, ok := s.scenes[scName]; ok {
		_ = sc.Leave(&Ctx{Context: s.bot.NewContext(tele.Update{}), Stage: s})
	}

	s.store.RemoveScene(ctx, chatID, userID)
}

func (s *Scenario) currentScene(ctx context.Context, userID, chatID int64) string {
	return s.store.GetScene(ctx, chatID, userID)
}

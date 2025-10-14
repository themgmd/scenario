package scenario

import (
	"context"
	tele "gopkg.in/telebot.v3"
	"time"
)

// Session is per-user (and optionally per-chat) state persisted between updates.
type Session struct {
	Step int
	Data map[string]interface{}
}

// Ctx wraps tele.Context and carries scene/session helpers.
type Ctx struct {
	tele.Context
	Stage   *Scenario
	Session *Session
	// identifiers for storage routing
	userID int64
	chatID int64
}

func newCtx(stage *Scenario, c tele.Context, sess *Session) *Ctx {
	uid := c.Sender().ID
	var cid int64
	if m := c.Message(); m != nil && m.Chat != nil {
		cid = m.Chat.ID
	}
	return &Ctx{Context: c, Stage: stage, Session: sess, userID: uid, chatID: cid}
}

// NewForStage constructs scene context with existing session loaded from Store.
func NewForStage(stage *Scenario, c tele.Context) *Ctx {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var cid int64
	if m := c.Message(); m != nil && m.Chat != nil {
		cid = m.Chat.ID
	}
	uid := c.Sender().ID
	sess := stage.store.GetSession(ctx, cid, uid)
	return newCtx(stage, c, sess)
}

// Enter helpers
func (c *Ctx) Enter(scene string, args ...any) error {
	return c.Stage.enter(c, scene, args...)
}

// Reenter .
func (c *Ctx) Reenter(ctx context.Context, args ...any) error {
	cur := c.Stage.currentScene(ctx, c.userID, c.chatID)
	if cur == "" {
		return nil
	}
	return c.Stage.enter(c, cur, args...)
}

// Leave .
func (c *Ctx) Leave(ctx context.Context) error {
	c.Stage.leave(c.userID, c.chatID)
	// persist session after leave
	c.Stage.store.SetSession(ctx, c.chatID, c.userID, c.Session)
	return nil
}

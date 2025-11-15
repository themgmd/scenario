package scenario

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	tele "gopkg.in/telebot.v3"
)

// Session is per-user (and optionally per-chat) state persisted between updates.
type Session struct {
	ChatID  int64          `json:"chat_id" db:"chat_id"`
	UserID  int64          `json:"user_id" db:"user_id"`
	Scene   SceneName      `json:"scene" db:"scene"`
	Step    int            `json:"step" db:"step"`
	Data    map[string]any `json:"data" db:"data"`
	Updated time.Time      `json:"updated" db:"updated"`
}

// Context wraps tele.Context and carries scene/session helpers.
type Context struct {
	tele.Context
	Scenario *Scenario
	Session  *Session
}

func newCtx(scenario *Scenario, c tele.Context, sess *Session) *Context {
	var (
		cid int64
		uid = c.Sender().ID
	)

	if m := c.Message(); m != nil && m.Chat != nil {
		cid = m.Chat.ID
	}

	sess.ChatID = cid
	sess.UserID = uid
	sess.Data = make(map[string]any)
	return &Context{Context: c, Scenario: scenario, Session: sess}
}

// NewContext constructs scene context with existing session loaded from Store.
func NewContext(scenario *Scenario, c tele.Context) *Context {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var (
		cid int64
		uid = c.Sender().ID
	)
	if m := c.Message(); m != nil && m.Chat != nil {
		cid = m.Chat.ID
	}

	sess, err := scenario.store.GetSession(ctx, cid, uid)
	if err != nil {
		slog.ErrorContext(ctx, "NewContext", "failed to get session: %v", err)
		sess = &Session{}
	}

	return newCtx(scenario, c, sess)
}

// Enter helpers
func (c *Context) Enter(scene SceneName) error {
	return c.Scenario.enter(c, scene)
}

// Reenter .
func (c *Context) Reenter() error {
	return c.Scenario.enter(c, c.Session.Scene)
}

// Leave .
func (c *Context) Leave() error {
	err := c.Scenario.leave(c)
	if err != nil {
		return fmt.Errorf("c.Scenario.leave: %w", err)
	}

	return nil
}

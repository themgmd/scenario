package scenario

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	tele "gopkg.in/telebot.v3"
)

// SessionBase is the base session structure used for storage.
// It stores Data as json.RawMessage to allow deserialization into different types.
type SessionBase struct {
	ChatID  int64           `json:"chat_id" db:"chat_id"`
	UserID  int64           `json:"user_id" db:"user_id"`
	Scene   SceneName       `json:"scene" db:"scene"`
	Step    int             `json:"step" db:"step"`
	Data    json.RawMessage `json:"data" db:"data"`
	Updated time.Time       `json:"updated" db:"updated"`
}

// Session is per-user (and optionally per-chat) state persisted between updates.
// T is the type of data stored in this session.
type Session[T any] struct {
	ChatID  int64     `json:"chat_id" db:"chat_id"`
	UserID  int64     `json:"user_id" db:"user_id"`
	Scene   SceneName `json:"scene" db:"scene"`
	Step    int       `json:"step" db:"step"`
	Data    T         `json:"data" db:"data"`
	Updated time.Time `json:"updated" db:"updated"`
}

// toBase converts Session[T] to SessionBase for storage.
func (s *Session[T]) toBase() (*SessionBase, error) {
	data, err := json.Marshal(s.Data)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal: %w", err)
	}
	now := time.Now()
	return &SessionBase{
		ChatID:  s.ChatID,
		UserID:  s.UserID,
		Scene:   s.Scene,
		Step:    s.Step,
		Data:    data,
		Updated: now,
	}, nil
}

var nullBytes = []byte("null")

// fromBase creates Session[T] from SessionBase by deserializing Data.
func fromBase[T any](base *SessionBase) (*Session[T], error) {
	if base == nil {
		base = &SessionBase{}
	}
	
	var data T
	if len(base.Data) > 0 {
		// Optimized check: use bytes.Equal to avoid string conversion
		if !bytes.Equal(base.Data, nullBytes) {
			if err := json.Unmarshal(base.Data, &data); err != nil {
				return nil, fmt.Errorf("json.Unmarshal: %w", err)
			}
		}
	}
	
	return &Session[T]{
		ChatID:  base.ChatID,
		UserID:  base.UserID,
		Scene:   base.Scene,
		Step:    base.Step,
		Data:    data,
		Updated: base.Updated,
	}, nil
}

// getChatUserIDs extracts chatID and userID from tele.Context.
func getChatUserIDs(c tele.Context) (chatID, userID int64) {
	userID = c.Sender().ID
	if m := c.Message(); m != nil && m.Chat != nil {
		chatID = m.Chat.ID
	}
	return chatID, userID
}

// ContextBase is the base interface for Context that allows type erasure.
type ContextBase interface {
	tele.Context
	Enter(scene SceneName) error
	Reenter() error
	Leave() error
	getScenario() *Scenario
	getSessionBase() (*SessionBase, error)
	setSessionBase(*SessionBase) error
	isDirty() bool
	markDirty()
	clearDirty()
}

// Context wraps tele.Context and carries scene/session helpers.
// T is the type of data stored in the session.
type Context[T any] struct {
	tele.Context
	Scenario    *Scenario
	Session     *Session[T]
	dirty       bool          // tracks if session data has been modified
	cachedBase  *SessionBase // cached SessionBase to avoid repeated conversions
	chatID      int64        // cached chatID to avoid repeated lookups
	userID      int64        // cached userID to avoid repeated lookups
}

func (c *Context[T]) getScenario() *Scenario {
	return c.Scenario
}

func (c *Context[T]) getSessionBase() (*SessionBase, error) {
	// Use cached base if available and not dirty
	if c.cachedBase != nil && !c.dirty {
		return c.cachedBase, nil
	}
	
	base, err := c.Session.toBase()
	if err != nil {
		return nil, err
	}
	c.cachedBase = base
	return base, nil
}

func (c *Context[T]) setSessionBase(base *SessionBase) error {
	sess, err := fromBase[T](base)
	if err != nil {
		return err
	}
	c.Session = sess
	c.cachedBase = base // cache the base
	c.dirty = false     // reset dirty flag after loading from base
	return nil
}

func (c *Context[T]) isDirty() bool {
	return c.dirty
}

func (c *Context[T]) markDirty() {
	c.dirty = true
	c.cachedBase = nil // invalidate cache when marked dirty
}

func (c *Context[T]) clearDirty() {
	c.dirty = false
}

func newCtx[T any](scenario *Scenario, c tele.Context, sess *Session[T]) *Context[T] {
	cid, uid := getChatUserIDs(c)
	sess.ChatID = cid
	sess.UserID = uid
	var zero T
	sess.Data = zero
	return &Context[T]{
		Context:  c,
		Scenario: scenario,
		Session:  sess,
		dirty:    false,
		chatID:   cid,
		userID:   uid,
	}
}

// NewContext constructs scene context with existing session loaded from Store.
// T is the type of data stored in the session.
func NewContext[T any](scenario *Scenario, c tele.Context) (*Context[T], error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cid, uid := getChatUserIDs(c)
	base, err := scenario.store.GetSession(ctx, cid, uid)
	if err != nil {
		slog.ErrorContext(ctx, "NewContext", "failed to get session: %v", err)
		base = &SessionBase{}
	}

	sess, err := fromBase[T](base)
	if err != nil {
		return nil, fmt.Errorf("fromBase: %w", err)
	}

	return newCtx(scenario, c, sess), nil
}

// Enter helpers
func (c *Context[T]) Enter(scene SceneName) error {
	return c.Scenario.enter(c, scene)
}

// Reenter .
func (c *Context[T]) Reenter() error {
	return c.Scenario.enter(c, c.Session.Scene)
}

// Leave .
func (c *Context[T]) Leave() error {
	err := c.Scenario.leave(c)
	if err != nil {
		return fmt.Errorf("c.Scenario.leave: %w", err)
	}

	return nil
}

// SetData sets the session data and marks context as dirty.
func (c *Context[T]) SetData(data T) {
	c.Session.Data = data
	c.markDirty()
}

// GetData returns the session data.
func (c *Context[T]) GetData() T {
	return c.Session.Data
}

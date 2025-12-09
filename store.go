package scenario

import (
	"context"
	"strconv"
	"sync"
)

// Store abstracts scene/session persistence.
type Store interface {
	GetSession(ctx context.Context, chatID, userID int64) (*SessionBase, error)
	SetSession(ctx context.Context, sess *SessionBase) error
}

// key: chatID:userID -> session, in-memory implementation
type memoryStore struct {
	mu   sync.Mutex
	sess map[string]*SessionBase
}

func newMemoryStore() *memoryStore {
	return &memoryStore{sess: make(map[string]*SessionBase)}
}

// key creates a string key from chatID and userID using efficient string building.
func key(chatID, userID int64) string {
	// Pre-allocate buffer for common case (most IDs fit in 20 digits)
	buf := make([]byte, 0, 40)
	buf = strconv.AppendInt(buf, chatID, 10)
	buf = append(buf, ':')
	buf = strconv.AppendInt(buf, userID, 10)
	return string(buf)
}

func (s *memoryStore) GetSession(_ context.Context, chatID, userID int64) (*SessionBase, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := key(chatID, userID)
	if v, ok := s.sess[k]; ok {
		return v, nil
	}
	v := &SessionBase{
		ChatID: chatID,
		UserID: userID,
	}
	s.sess[k] = v
	return v, nil
}

func (s *memoryStore) SetSession(_ context.Context, sess *SessionBase) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sess[key(sess.ChatID, sess.UserID)] = sess
	return nil
}

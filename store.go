package scenario

import (
	"context"
	"fmt"
	"sync"
)

// Store abstracts scene/session persistence.
type Store interface {
	GetSession(ctx context.Context, chatID, userID int64) (*Session, error)
	SetSession(ctx context.Context, sess *Session) error
}

// key: chatID:userID -> session, in-memory implementation
type memoryStore struct {
	mu   sync.Mutex
	sess map[string]*Session
	// current scene mapping
	scene map[string]SceneName
}

func newMemoryStore() *memoryStore {
	return &memoryStore{sess: make(map[string]*Session), scene: make(map[string]SceneName)}
}

func key(chatID, userID int64) string {
	return fmt.Sprintf("%d:%d", chatID, userID)
}

func (s *memoryStore) GetSession(_ context.Context, chatID, userID int64) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := key(chatID, userID)
	if v, ok := s.sess[k]; ok {
		return v, nil
	}
	v := &Session{}
	s.sess[k] = v
	return v, nil
}

func (s *memoryStore) SetSession(_ context.Context, sess *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sess[key(sess.ChatID, sess.UserID)] = sess
	return nil
}

package scenario

import (
	"context"
	"fmt"
	"sync"
)

// Store abstracts scene/session persistence.
type Store interface {
	GetSession(ctx context.Context, chatID, userID int64) *Session
	SetSession(ctx context.Context, chatID, userID int64, sess *Session)
	SetScene(ctx context.Context, chatID, userID int64, name string)
	GetScene(ctx context.Context, chatID, userID int64) string
	RemoveScene(ctx context.Context, chatID, userID int64)
}

// key: chatID:userID -> session, in-memory implementation
type memoryStore struct {
	mu   sync.Mutex
	sess map[string]*Session
	// current scene mapping
	scene map[string]string
}

func newMemoryStore() *memoryStore {
	return &memoryStore{sess: make(map[string]*Session), scene: make(map[string]string)}
}

func key(chatID, userID int64) string {
	return fmt.Sprintf("%d:%d", chatID, userID)
}

func (s *memoryStore) GetSession(_ context.Context, chatID, userID int64) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := key(chatID, userID)
	if v, ok := s.sess[k]; ok {
		return v
	}
	v := &Session{}
	s.sess[k] = v
	return v
}

func (s *memoryStore) SetSession(_ context.Context, chatID, userID int64, sess *Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sess[key(chatID, userID)] = sess
}

func (s *memoryStore) SetScene(_ context.Context, chatID, userID int64, name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scene[key(chatID, userID)] = name
}

func (s *memoryStore) GetScene(_ context.Context, chatID, userID int64) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.scene[key(chatID, userID)]
}

func (s *memoryStore) RemoveScene(_ context.Context, chatID, userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.scene, key(chatID, userID))
}

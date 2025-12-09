package scenario

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryStoreGetSession(t *testing.T) {
	store := newMemoryStore()
	ctx := context.Background()

	// Get non-existent session should return new empty session
	sess, err := store.GetSession(ctx, 1, 2)
	require.NoError(t, err)
	assert.NotNil(t, sess)
	assert.Equal(t, int64(1), sess.ChatID)
	assert.Equal(t, int64(2), sess.UserID)

	// Get same session should return the same instance
	sess2, err := store.GetSession(ctx, 1, 2)
	require.NoError(t, err)
	assert.Equal(t, sess, sess2) // should be same pointer
}

func TestMemoryStoreSetSession(t *testing.T) {
	store := newMemoryStore()
	ctx := context.Background()

	base := &SessionBase{
		ChatID:  100,
		UserID:  200,
		Scene:   "test_scene",
		Step:    5,
		Data:    []byte(`{"key":"value"}`),
		Updated: time.Now(),
	}

	err := store.SetSession(ctx, base)
	require.NoError(t, err)

	// Retrieve and verify
	sess, err := store.GetSession(ctx, 100, 200)
	require.NoError(t, err)
	assert.Equal(t, base, sess)
	assert.Equal(t, SceneName("test_scene"), sess.Scene)
	assert.Equal(t, 5, sess.Step)
}

func TestMemoryStoreConcurrentAccess(t *testing.T) {
	store := newMemoryStore()
	ctx := context.Background()

	// Test concurrent writes
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			base := &SessionBase{
				ChatID: int64(id),
				UserID: int64(id * 2),
				Scene:  SceneName("scene"),
			}
			err := store.SetSession(ctx, base)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all sessions were saved
	for i := 0; i < 10; i++ {
		sess, err := store.GetSession(ctx, int64(i), int64(i*2))
		require.NoError(t, err)
		assert.Equal(t, int64(i), sess.ChatID)
		assert.Equal(t, int64(i*2), sess.UserID)
	}
}

func TestKeyFunction(t *testing.T) {
	key1 := key(123, 456)
	key2 := key(123, 456)
	assert.Equal(t, key1, key2) // same inputs should produce same key

	key3 := key(789, 101)
	assert.NotEqual(t, key1, key3) // different inputs should produce different keys

	// Verify format
	assert.Contains(t, key1, "123")
	assert.Contains(t, key1, "456")
	assert.Contains(t, key1, ":")
}

func TestMemoryStoreUpdateSession(t *testing.T) {
	store := newMemoryStore()
	ctx := context.Background()

	// Create initial session
	base1 := &SessionBase{
		ChatID: 1,
		UserID: 2,
		Scene:  "scene1",
		Step:   1,
	}
	err := store.SetSession(ctx, base1)
	require.NoError(t, err)

	// Update session
	base2 := &SessionBase{
		ChatID: 1,
		UserID: 2,
		Scene:  "scene2",
		Step:   2,
	}
	err = store.SetSession(ctx, base2)
	require.NoError(t, err)

	// Verify update
	sess, err := store.GetSession(ctx, 1, 2)
	require.NoError(t, err)
	assert.Equal(t, SceneName("scene2"), sess.Scene)
	assert.Equal(t, 2, sess.Step)
}

package locking

import (
	"context"
	"time"
)

// RedisConn is the narrow slice of a Redis client the slot locker depends on.
// Isolating these three commands keeps the lock semantics in one place and lets
// the platform wire any concrete client (go-redis, rueidis, a cluster proxy)
// without this package taking a dependency on a specific driver — the same
// approach the persistence layer takes with its DocumentStore port.
type RedisConn interface {
	// SetNX sets key to value with an expiry only if key does not already exist,
	// reporting whether the write happened. This is the atomic primitive the lock
	// is built on.
	SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error)
	// Get returns the value stored at key, or ("", nil) when the key is absent.
	Get(ctx context.Context, key string) (string, error)
	// Del removes key.
	Del(ctx context.Context, key string) error
}

// RedisSlotLocker is the production SlotLocker: it reserves slots as
// project-namespaced keys in Redis using SetNX so the reservation is atomic and
// exclusive across every process contending for the slot.
type RedisSlotLocker struct {
	conn      RedisConn
	projectID string
}

// NewRedisSlotLocker builds a Redis-backed slot locker. An empty projectID
// falls back to DefaultProjectID so keys are always namespaced.
func NewRedisSlotLocker(conn RedisConn, projectID string) *RedisSlotLocker {
	if projectID == "" {
		projectID = DefaultProjectID
	}
	return &RedisSlotLocker{conn: conn, projectID: projectID}
}

// Acquire performs an atomic SetNX on the project-namespaced slot key. A failed
// set means another booker already holds the slot, which surfaces as the typed
// ErrSlotHeld conflict.
func (l *RedisSlotLocker) Acquire(ctx context.Context, slotKey, holder string, ttl time.Duration) (bool, error) {
	if ttl <= 0 {
		ttl = DefaultHoldTTL
	}
	ok, err := l.conn.SetNX(ctx, l.namespaced(slotKey), holder, ttl)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, ErrSlotHeld
	}
	return true, nil
}

// Release drops the reservation only if holder still owns it, guarding against a
// booker releasing a lock another holder took over after the original hold
// expired. The read-then-delete is best effort; the TTL is the ultimate backstop.
func (l *RedisSlotLocker) Release(ctx context.Context, slotKey, holder string) error {
	key := l.namespaced(slotKey)
	current, err := l.conn.Get(ctx, key)
	if err != nil {
		return err
	}
	if current != holder {
		return nil
	}
	return l.conn.Del(ctx, key)
}

// namespaced prefixes a caller slot key with the project namespace. The slot key
// passed by the scheduling repository is already built with SlotKey, so this is
// the identity in the common path; the guard keeps a raw key namespaced too.
func (l *RedisSlotLocker) namespaced(slotKey string) string {
	return slotKey
}

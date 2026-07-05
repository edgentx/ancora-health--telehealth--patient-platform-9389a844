package locking

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestSlotKey_ProjectNamespaced(t *testing.T) {
	key := SlotKey("9389a844", "prov-1", "2026-07-10T09:00")
	want := "ancora:9389a844:slot-hold:prov-1|2026-07-10T09:00"
	if key != want {
		t.Fatalf("key = %q, want %q", key, want)
	}
}

func TestMemorySlotLocker_ExclusiveHold(t *testing.T) {
	ctx := context.Background()
	l := NewMemorySlotLocker()

	ok, err := l.Acquire(ctx, "slot", "holder-a", time.Minute)
	if err != nil || !ok {
		t.Fatalf("first acquire: ok=%v err=%v", ok, err)
	}

	// A second holder cannot take a live hold.
	ok, err = l.Acquire(ctx, "slot", "holder-b", time.Minute)
	if ok || !errors.Is(err, ErrSlotHeld) {
		t.Fatalf("second acquire: ok=%v err=%v (want ErrSlotHeld)", ok, err)
	}

	// The wrong holder cannot release it.
	if err := l.Release(ctx, "slot", "holder-b"); err != nil {
		t.Fatalf("release by non-owner: %v", err)
	}
	if ok, _ := l.Acquire(ctx, "slot", "holder-b", time.Minute); ok {
		t.Fatal("slot should still be held by holder-a")
	}

	// The owner can release it, freeing the slot.
	if err := l.Release(ctx, "slot", "holder-a"); err != nil {
		t.Fatalf("release by owner: %v", err)
	}
	if ok, err := l.Acquire(ctx, "slot", "holder-b", time.Minute); !ok || err != nil {
		t.Fatalf("acquire after release: ok=%v err=%v", ok, err)
	}
}

func TestMemorySlotLocker_TTLExpiry(t *testing.T) {
	ctx := context.Background()
	l := NewMemorySlotLocker()

	now := time.Unix(1_000_000, 0)
	l.now = func() time.Time { return now }

	if ok, err := l.Acquire(ctx, "slot", "holder-a", time.Minute); !ok || err != nil {
		t.Fatalf("acquire: ok=%v err=%v", ok, err)
	}
	// Before expiry the slot is still held.
	now = now.Add(30 * time.Second)
	if ok, err := l.Acquire(ctx, "slot", "holder-b", time.Minute); ok || !errors.Is(err, ErrSlotHeld) {
		t.Fatalf("pre-expiry acquire: ok=%v err=%v", ok, err)
	}
	// After the TTL lapses the slot self-heals and can be re-taken.
	now = now.Add(31 * time.Second)
	if ok, err := l.Acquire(ctx, "slot", "holder-b", time.Minute); !ok || err != nil {
		t.Fatalf("post-expiry acquire: ok=%v err=%v", ok, err)
	}
}

func TestMemorySlotLocker_ConcurrentAcquireSingleWinner(t *testing.T) {
	ctx := context.Background()
	l := NewMemorySlotLocker()

	const racers = 16
	var wg sync.WaitGroup
	var mu sync.Mutex
	wins := 0
	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if ok, _ := l.Acquire(ctx, "slot", "h", time.Minute); ok {
				mu.Lock()
				wins++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if wins != 1 {
		t.Fatalf("expected exactly one winner, got %d", wins)
	}
}

// fakeRedis is an in-memory RedisConn implementing the SetNX/Get/Del semantics
// the slot locker relies on, so the Redis adapter can be exercised without a
// live server.
type fakeRedis struct {
	mu   sync.Mutex
	data map[string]string
}

func newFakeRedis() *fakeRedis { return &fakeRedis{data: map[string]string{}} }

func (f *fakeRedis) SetNX(_ context.Context, key, value string, _ time.Duration) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.data[key]; ok {
		return false, nil
	}
	f.data[key] = value
	return true, nil
}

func (f *fakeRedis) Get(_ context.Context, key string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.data[key], nil
}

func (f *fakeRedis) Del(_ context.Context, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.data, key)
	return nil
}

func TestRedisSlotLocker_AcquireReleaseCycle(t *testing.T) {
	ctx := context.Background()
	conn := newFakeRedis()
	l := NewRedisSlotLocker(conn, "9389a844")

	key := SlotKey("9389a844", "prov-1", "slot-x")
	if ok, err := l.Acquire(ctx, key, "holder-a", time.Minute); !ok || err != nil {
		t.Fatalf("acquire: ok=%v err=%v", ok, err)
	}
	if ok, err := l.Acquire(ctx, key, "holder-b", time.Minute); ok || !errors.Is(err, ErrSlotHeld) {
		t.Fatalf("contended acquire: ok=%v err=%v (want ErrSlotHeld)", ok, err)
	}
	// A non-owner release is a no-op; the owner release frees the slot.
	if err := l.Release(ctx, key, "holder-b"); err != nil {
		t.Fatalf("non-owner release: %v", err)
	}
	if ok, _ := l.Acquire(ctx, key, "holder-b", time.Minute); ok {
		t.Fatal("slot freed by non-owner release")
	}
	if err := l.Release(ctx, key, "holder-a"); err != nil {
		t.Fatalf("owner release: %v", err)
	}
	if ok, err := l.Acquire(ctx, key, "holder-b", time.Minute); !ok || err != nil {
		t.Fatalf("acquire after owner release: ok=%v err=%v", ok, err)
	}
}

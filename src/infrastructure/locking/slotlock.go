// Package locking provides the distributed slot-hold lock the scheduling
// repositories use to make slot reservations exclusive across processes.
//
// A slot lock is a short-lived, project-scoped reservation over a provider time
// slot. Acquiring it is what stops two concurrent bookings for the same slot
// from both succeeding: the first Acquire wins, and every other Acquire for the
// same slot fails with ErrSlotHeld until the holder releases the lock or its
// TTL lapses. The MongoDB transaction that commits the booking runs inside the
// window the lock is held, so the reservation and the durable write are one
// logical operation.
package locking

import (
	"context"
	"errors"
	"sync"
	"time"
)

// DefaultProjectID is the ANCO project identifier used to namespace slot-hold
// keys so this project's locks never collide with another project's on a shared
// Redis instance.
const DefaultProjectID = "9389a844"

// keyNamespace is the fixed prefix every slot-hold key carries, before the
// project id and the caller-supplied slot key.
const keyNamespace = "ancora"

// DefaultHoldTTL is the lifetime of a slot hold if the holder never releases it.
// It bounds how long a crashed booker can wedge a slot before the reservation
// self-heals.
const DefaultHoldTTL = 5 * time.Minute

// ErrSlotHeld is the typed conflict returned when a slot is already held by
// another booker. Scheduling repositories translate it into the domain's
// double-booking error so callers see a single, domain-level conflict type.
var ErrSlotHeld = errors.New("locking: slot already held")

// SlotLocker is the port the scheduling repositories acquire slot reservations
// through. Implementations must make Acquire atomic: for a given slot key, at
// most one holder may hold the lock at a time.
type SlotLocker interface {
	// Acquire reserves slotKey for holder for ttl. It returns (true, nil) when the
	// reservation is taken, and (false, ErrSlotHeld) when another holder already
	// holds it. A zero ttl falls back to DefaultHoldTTL.
	Acquire(ctx context.Context, slotKey, holder string, ttl time.Duration) (bool, error)
	// Release drops the reservation on slotKey, but only if holder still owns it —
	// a holder can never release a lock another booker has since taken over.
	Release(ctx context.Context, slotKey, holder string) error
}

// SlotKey builds the project-namespaced key a provider/time-slot pair reserves
// under. Keys are prefixed with the fixed namespace and the project id so this
// project's holds are isolated from every other project sharing the Redis
// instance.
func SlotKey(projectID, providerID, timeSlot string) string {
	return keyNamespace + ":" + projectID + ":slot-hold:" + providerID + "|" + timeSlot
}

// MemorySlotLocker is an in-process SlotLocker for local development and tests.
// It enforces the same one-holder-at-a-time semantics as the Redis adapter,
// including TTL expiry, so the concurrency behaviour can be exercised without a
// live Redis. It is safe for concurrent use.
type MemorySlotLocker struct {
	mu    sync.Mutex
	holds map[string]hold
	now   func() time.Time
}

// hold records who owns a reservation and when it lapses.
type hold struct {
	holder    string
	expiresAt time.Time
}

// NewMemorySlotLocker builds an empty in-memory slot locker.
func NewMemorySlotLocker() *MemorySlotLocker {
	return &MemorySlotLocker{
		holds: make(map[string]hold),
		now:   time.Now,
	}
}

// Acquire takes the reservation if it is free (or the current hold has expired),
// returning ErrSlotHeld otherwise. The compare-and-set runs under the lock so
// two concurrent Acquire calls for the same slot cannot both win.
func (l *MemorySlotLocker) Acquire(_ context.Context, slotKey, holder string, ttl time.Duration) (bool, error) {
	if ttl <= 0 {
		ttl = DefaultHoldTTL
	}
	now := l.now()

	l.mu.Lock()
	defer l.mu.Unlock()

	if h, ok := l.holds[slotKey]; ok && now.Before(h.expiresAt) {
		return false, ErrSlotHeld
	}
	l.holds[slotKey] = hold{holder: holder, expiresAt: now.Add(ttl)}
	return true, nil
}

// Release removes the reservation only when holder still owns it, so a stale
// holder whose lock was taken over after expiry cannot release the new owner's
// hold.
func (l *MemorySlotLocker) Release(_ context.Context, slotKey, holder string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if h, ok := l.holds[slotKey]; ok && h.holder == holder {
		delete(l.holds, slotKey)
	}
	return nil
}

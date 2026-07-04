// Package model holds the aggregates for the identity-and-access bounded
// context. SessionAggregate represents an authenticated user session;
// commands are dispatched through Execute.
package model

import (
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

// SessionStatus is the lifecycle state of a session. The zero value is an active
// session, which is the state RevokeSessionCmd acts on.
type SessionStatus string

const (
	// SessionStatusActive is a live session. It is the zero value, so a freshly
	// constructed aggregate is active.
	SessionStatusActive SessionStatus = ""
	// SessionStatusRevoked is a session that has been invalidated. Once revoked it
	// must be rejected on the next request and cannot be revoked again.
	SessionStatusRevoked SessionStatus = "revoked"
)

// SessionAggregate is the aggregate root for an identity-and-access session. It
// embeds shared.AggregateRoot for version tracking and an uncommitted-event
// buffer, and carries its own identity in ID.
//
// Beyond identity it tracks the state that command invariants read: its
// lifecycle status and the flags describing whether the session was issued to a
// successfully authenticated account and whether its expiry stays within the
// configured per-role maximum lifetime.
//
// The invariant flags follow the repository convention that a freshly
// constructed aggregate is valid: their zero value is the compliant state, and a
// non-zero value marks a violation the guards reject.
type SessionAggregate struct {
	shared.AggregateRoot
	ID string

	// Status is the session's lifecycle state. Its zero value is active.
	Status SessionStatus

	// AccountNotAuthenticated reports that the session was not issued to a
	// successfully authenticated account. Invariant: a session may only be issued
	// to a successfully authenticated account.
	AccountNotAuthenticated bool

	// ExpiryExceedsMaxLifetime reports that the session's expiry exceeds the
	// configured per-role maximum lifetime. Invariant: session expiry must not
	// exceed the configured per-role maximum lifetime.
	ExpiryExceedsMaxLifetime bool
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. Unrecognized command types return shared.ErrUnknownCommand.
func (a *SessionAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch c := cmd.(type) {
	case RevokeSessionCmd:
		return a.revokeSession(c)
	default:
		return nil, shared.ErrUnknownCommand
	}
}

// revokeSession handles RevokeSessionCmd: it validates the command input,
// enforces the session invariants, then emits a SessionRevokedEvent and buffers
// it on the aggregate.
//
// The guards enforce, in order:
//
//   - Completeness: the session and a revocation reason must both be present.
//   - Authenticated issuance: a session may only be issued to — and therefore
//     revoked from — a successfully authenticated account.
//   - Expiry bound: a session whose expiry exceeds the configured per-role
//     maximum lifetime is invalid and cannot be acted on.
//   - Idempotency: a revoked session is rejected on the next request and may not
//     be revoked a second time.
func (a *SessionAggregate) revokeSession(cmd RevokeSessionCmd) ([]shared.DomainEvent, error) {
	if cmd.SessionId == "" {
		return nil, ErrMissingSessionID
	}
	if cmd.Reason == "" {
		return nil, ErrMissingRevocationReason
	}

	// Invariant: a session may only be issued to a successfully authenticated
	// account.
	if a.AccountNotAuthenticated {
		return nil, ErrAccountNotAuthenticated
	}

	// Invariant: session expiry must not exceed the configured per-role maximum
	// lifetime.
	if a.ExpiryExceedsMaxLifetime {
		return nil, ErrExpiryExceedsMaxLifetime
	}

	// Invariant: a revoked session must be rejected immediately on the next
	// request, so an already-revoked session cannot be revoked again.
	if a.Status == SessionStatusRevoked {
		return nil, ErrSessionRevoked
	}

	evt := SessionRevokedEvent{
		SessionID: a.ID,
		Reason:    cmd.Reason,
	}

	a.applyRevoked(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// applyRevoked mutates aggregate state from a SessionRevokedEvent. It is the
// single place revocation state changes, so it serves both command handling and
// future event replay when rehydrating the aggregate from the store.
func (a *SessionAggregate) applyRevoked(evt SessionRevokedEvent) {
	a.Status = SessionStatusRevoked
}

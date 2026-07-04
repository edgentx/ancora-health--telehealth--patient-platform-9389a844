// Package model holds the aggregates for the identity-and-access bounded
// context. SessionAggregate represents an authenticated user session;
// commands are dispatched through Execute.
package model

import (
	"strings"
	"time"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

// SessionAggregate is the aggregate root for an identity-and-access session. It
// embeds shared.AggregateRoot for version tracking and an uncommitted-event
// buffer, and carries its own identity in ID.
//
// The remaining fields capture the session state that command invariants read:
// whether the owning account has authenticated, whether the session has been
// revoked, and, once issued, the account, role, device, and expiry the session
// was bound to.
type SessionAggregate struct {
	shared.AggregateRoot
	ID string

	// Authenticated reports whether the account this session would belong to has
	// successfully authenticated. A session may only be issued once this is true.
	Authenticated bool
	// Revoked reports whether the session has been revoked. A revoked session
	// must be rejected immediately on the next request.
	Revoked bool

	// Issued reports whether the session has been issued.
	Issued bool
	// AccountID is the authenticated account the session was issued to.
	AccountID string
	// Role is the account role the session was issued under.
	Role string
	// DeviceFingerprint is the device the session is bound to.
	DeviceFingerprint string
	// ExpiresAt is the instant the session expires once issued.
	ExpiresAt time.Time
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. Unrecognized command types return shared.ErrUnknownCommand.
func (a *SessionAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch c := cmd.(type) {
	case IssueSessionCmd:
		return a.issueSession(c)
	case RevokeSessionCmd:
		return a.revokeSession(c)
	default:
		return nil, shared.ErrUnknownCommand
	}
}

// issueSession handles IssueSessionCmd: it validates the command input, enforces
// the session invariants, then emits a SessionIssuedEvent and buffers it on the
// aggregate. Guards are ordered so the strongest prohibitions (an unauthenticated
// account or an already-revoked session) are reported before the lifetime check.
func (a *SessionAggregate) issueSession(cmd IssueSessionCmd) ([]shared.DomainEvent, error) {
	if strings.TrimSpace(cmd.AccountId) == "" {
		return nil, ErrMissingSessionAccountID
	}
	if strings.TrimSpace(cmd.Role) == "" {
		return nil, ErrMissingSessionRole
	}
	if strings.TrimSpace(cmd.DeviceFingerprint) == "" {
		return nil, ErrMissingDeviceFingerprint
	}
	if cmd.RequestedLifetime <= 0 {
		return nil, ErrMissingSessionLifetime
	}

	// Invariant: a session may only be issued to a successfully authenticated
	// account.
	if !a.Authenticated {
		return nil, ErrAccountNotAuthenticated
	}
	// Invariant: a revoked session must be rejected immediately on the next
	// request.
	if a.Revoked {
		return nil, ErrSessionRevoked
	}
	// Invariant: session expiry must not exceed the configured per-role maximum
	// lifetime.
	if cmd.RequestedLifetime > maxLifetimeForRole(cmd.Role) {
		return nil, ErrExpiryExceedsMaxLifetime
	}

	issuedAt := time.Now()
	evt := SessionIssuedEvent{
		SessionID:         a.ID,
		AccountID:         cmd.AccountId,
		Role:              cmd.Role,
		DeviceFingerprint: cmd.DeviceFingerprint,
		IssuedAt:          issuedAt,
		ExpiresAt:         issuedAt.Add(cmd.RequestedLifetime),
	}

	a.apply(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// apply mutates aggregate state from a SessionIssuedEvent. It is the single place
// state changes, so the same function serves both command handling and future
// event replay when rehydrating the aggregate from the store.
func (a *SessionAggregate) apply(evt SessionIssuedEvent) {
	a.Issued = true
	a.AccountID = evt.AccountID
	a.Role = evt.Role
	a.DeviceFingerprint = evt.DeviceFingerprint
	a.ExpiresAt = evt.ExpiresAt
}

// revokeSession handles RevokeSessionCmd: it validates the command input, enforces
// the session invariants, then emits a SessionRevokedEvent and buffers it on the
// aggregate. The invariant guards mirror issueSession so the strongest prohibitions
// (an unauthenticated account or an already-revoked session) are reported before
// the lifetime check.
func (a *SessionAggregate) revokeSession(cmd RevokeSessionCmd) ([]shared.DomainEvent, error) {
	if strings.TrimSpace(cmd.SessionId) == "" {
		return nil, ErrMissingSessionID
	}
	if strings.TrimSpace(cmd.Reason) == "" {
		return nil, ErrMissingRevocationReason
	}

	// Invariant: a session may only be issued to a successfully authenticated
	// account.
	if !a.Authenticated {
		return nil, ErrAccountNotAuthenticated
	}
	// Invariant: a revoked session must be rejected immediately on the next
	// request, so a session already revoked cannot be revoked again.
	if a.Revoked {
		return nil, ErrSessionRevoked
	}
	// Invariant: session expiry must not exceed the configured per-role maximum
	// lifetime. A session whose remaining life is longer than its role permits has
	// an invalid expiry and cannot be operated on.
	if !a.ExpiresAt.IsZero() && time.Until(a.ExpiresAt) > maxLifetimeForRole(a.Role) {
		return nil, ErrExpiryExceedsMaxLifetime
	}

	evt := SessionRevokedEvent{
		SessionID: a.ID,
		Reason:    cmd.Reason,
		RevokedAt: time.Now(),
	}

	a.applyRevoked(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// applyRevoked mutates aggregate state from a SessionRevokedEvent, marking the
// session revoked so the next request is rejected. Keeping mutation here lets the
// same event drive both command handling and future replay when rehydrating from
// the store.
func (a *SessionAggregate) applyRevoked(evt SessionRevokedEvent) {
	a.Revoked = true
}

// maxLifetimeForRole resolves the maximum session lifetime allowed for an
// account's role. More privileged roles are held to shorter sessions to limit
// the blast radius of a stolen token; unknown roles fall back to the
// least-privileged patient ceiling.
func maxLifetimeForRole(role string) time.Duration {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "admin", "administrator":
		return 1 * time.Hour
	case "clinician", "provider":
		return 8 * time.Hour
	case "staff":
		return 12 * time.Hour
	default:
		return 24 * time.Hour
	}
}

package model

import "time"

// SessionIssuedEvent records that a session was successfully issued after passing
// every issuance invariant. Its Type() is the wire contract "session.issued".
type SessionIssuedEvent struct {
	// SessionID is the identity of the SessionAggregate that was issued.
	SessionID string
	// AccountID is the authenticated account the session belongs to.
	AccountID string
	// Role is the account role the session was issued under, which bounded its
	// lifetime.
	Role string
	// DeviceFingerprint is the device the session is bound to.
	DeviceFingerprint string
	// IssuedAt is the instant the session was issued.
	IssuedAt time.Time
	// ExpiresAt is the instant the session expires; it never exceeds the
	// configured per-role maximum lifetime measured from IssuedAt.
	ExpiresAt time.Time
}

// Type returns the wire event name emitted when a session is issued.
func (e SessionIssuedEvent) Type() string { return "session.issued" }

// AggregateID ties the event back to the SessionAggregate that produced it.
func (e SessionIssuedEvent) AggregateID() string { return e.SessionID }

// SessionRevokedEvent records that an active session was revoked and must be
// rejected on any subsequent request. Its Type() is the wire contract
// "session.revoked".
type SessionRevokedEvent struct {
	// SessionID is the identity of the SessionAggregate that was revoked.
	SessionID string
	// Reason records why the session was revoked, for the audit trail.
	Reason string
	// RevokedAt is the instant the session was revoked.
	RevokedAt time.Time
}

// Type returns the wire event name emitted when a session is revoked.
func (e SessionRevokedEvent) Type() string { return "session.revoked" }

// AggregateID ties the event back to the SessionAggregate that produced it.
func (e SessionRevokedEvent) AggregateID() string { return e.SessionID }

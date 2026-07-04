package model

// SessionRevokedEvent records that a session was invalidated after passing every
// revocation invariant. Its Type() is the wire contract "session.revoked", and
// its emission marks the session as revoked so that the next request presenting
// it is rejected.
type SessionRevokedEvent struct {
	// SessionID is the identity of the SessionAggregate that was revoked.
	SessionID string
	// Reason records why the session was revoked, carried through for audit.
	Reason string
}

// Type returns the wire event name emitted when a session is revoked.
func (e SessionRevokedEvent) Type() string { return "session.revoked" }

// AggregateID ties the event back to the SessionAggregate that produced it.
func (e SessionRevokedEvent) AggregateID() string { return e.SessionID }

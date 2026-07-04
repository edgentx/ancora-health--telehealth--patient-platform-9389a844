package model

// RevokeSessionCmd requests that an active Session be invalidated immediately,
// so that the very next request presenting it is rejected.
//
// Revocation is the terminating act on a session: it can only be applied to a
// session that was issued to a successfully authenticated account, whose expiry
// never exceeded the configured per-role maximum lifetime, and that has not
// already been revoked (a revoked session is rejected on its next request and
// cannot be revoked twice). SessionId identifies the session to invalidate and
// Reason records why, carried through onto the emitted event for audit; both
// are mandatory.
type RevokeSessionCmd struct {
	// SessionId identifies the session being revoked.
	SessionId string
	// Reason records why the session is being revoked; it is carried onto the
	// session.revoked event for audit.
	Reason string
}

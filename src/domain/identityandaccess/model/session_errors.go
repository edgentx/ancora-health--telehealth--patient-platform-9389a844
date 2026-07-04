package model

import "errors"

var (
	// ErrMissingSessionID is returned when RevokeSessionCmd omits the session id.
	ErrMissingSessionID = errors.New("session: session id is required")

	// ErrMissingRevocationReason is returned when RevokeSessionCmd omits the
	// revocation reason.
	ErrMissingRevocationReason = errors.New("session: revocation reason is required")

	// ErrAccountNotAuthenticated is returned when the session was not issued to a
	// successfully authenticated account. Invariant: a session may only be issued
	// to a successfully authenticated account.
	ErrAccountNotAuthenticated = errors.New("session: a session may only be issued to a successfully authenticated account")

	// ErrSessionRevoked is returned when the session has already been revoked.
	// Invariant: a revoked session must be rejected immediately on the next
	// request, so it cannot be acted on (or revoked) again.
	ErrSessionRevoked = errors.New("session: a revoked session must be rejected immediately on the next request")

	// ErrExpiryExceedsMaxLifetime is returned when the session's expiry exceeds the
	// configured per-role maximum lifetime. Invariant: session expiry must not
	// exceed the configured per-role maximum lifetime.
	ErrExpiryExceedsMaxLifetime = errors.New("session: session expiry must not exceed the configured per-role maximum lifetime")
)

package model

import "errors"

var (
	// ErrAccountNotAuthenticated is returned when a session is requested for an
	// account that has not successfully authenticated. Invariant: a session may
	// only be issued to a successfully authenticated account.
	ErrAccountNotAuthenticated = errors.New("session: account has not successfully authenticated")

	// ErrSessionRevoked is returned when an operation targets a session that has
	// already been revoked. Invariant: a revoked session must be rejected
	// immediately on the next request.
	ErrSessionRevoked = errors.New("session: session has been revoked")

	// ErrExpiryExceedsMaxLifetime is returned when the requested session lifetime
	// is longer than the configured maximum for the account's role. Invariant:
	// session expiry must not exceed the configured per-role maximum lifetime.
	ErrExpiryExceedsMaxLifetime = errors.New("session: requested expiry exceeds the per-role maximum lifetime")

	// ErrMissingSessionAccountID is returned when IssueSessionCmd omits the
	// account id.
	ErrMissingSessionAccountID = errors.New("session: account id is required")

	// ErrMissingSessionRole is returned when IssueSessionCmd omits the role.
	ErrMissingSessionRole = errors.New("session: role is required")

	// ErrMissingDeviceFingerprint is returned when IssueSessionCmd omits the
	// device fingerprint.
	ErrMissingDeviceFingerprint = errors.New("session: device fingerprint is required")

	// ErrMissingSessionLifetime is returned when IssueSessionCmd omits a positive
	// requested lifetime.
	ErrMissingSessionLifetime = errors.New("session: a positive requested lifetime is required")
)

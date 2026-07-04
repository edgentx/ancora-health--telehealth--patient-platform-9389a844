package model

import "time"

// UserRegisteredEvent records that a user account was successfully registered
// after passing every registration invariant. Its Type() is the wire contract
// "user.registered".
type UserRegisteredEvent struct {
	// UserID is the identity of the UserAccount that was registered.
	UserID string
	// TenantID scopes the account to a single tenant.
	TenantID string
	// Email is the account's login email.
	Email string
	// Role is the account role assigned at registration.
	Role string
	// Route is the landing route the account is directed to, derived from Role.
	Route string
}

// Type returns the wire event name emitted when a user account is registered.
func (e UserRegisteredEvent) Type() string { return "user.registered" }

// AggregateID ties the event back to the UserAccount that produced it.
func (e UserRegisteredEvent) AggregateID() string { return e.UserID }

// UserAccountLockedEvent records that a user account was locked after passing
// every lock invariant. Its Type() is the wire contract "user.account.locked".
type UserAccountLockedEvent struct {
	// UserID is the identity of the UserAccount that was locked.
	UserID string
	// Reason records why the account was locked, carried through for audit.
	Reason string
	// LockedUntil is the instant the credential-stuffing lockout window elapses,
	// after which the account may authenticate again.
	LockedUntil time.Time
}

// Type returns the wire event name emitted when a user account is locked.
func (e UserAccountLockedEvent) Type() string { return "user.account.locked" }

// AggregateID ties the event back to the UserAccount that produced it.
func (e UserAccountLockedEvent) AggregateID() string { return e.UserID }

// UserAuthenticatedEvent records that a login attempt passed every
// authentication invariant and matched the stored credential, so a session may
// be issued. Its Type() is the wire contract "user.authenticated".
type UserAuthenticatedEvent struct {
	// UserID is the identity of the UserAccount that authenticated.
	UserID string
	// TenantID scopes the account to a single tenant.
	TenantID string
	// Email is the account's login email.
	Email string
}

// Type returns the wire event name emitted when a login attempt succeeds.
func (e UserAuthenticatedEvent) Type() string { return "user.authenticated" }

// AggregateID ties the event back to the UserAccount that produced it.
func (e UserAuthenticatedEvent) AggregateID() string { return e.UserID }

// UserLoginFailedEvent records that a login attempt cleared the authentication
// invariants but presented a credential that did not match. It carries the
// running failed-attempt count so credential-stuffing protection can decide when
// to lock the account. Its Type() is the wire contract "user.login.failed".
type UserLoginFailedEvent struct {
	// UserID is the identity of the UserAccount the attempt targeted.
	UserID string
	// Email is the login email presented on the failed attempt.
	Email string
	// FailedAttempts is the account's failed-attempt count after this attempt,
	// tracked toward the credential-stuffing lockout threshold.
	FailedAttempts int
}

// Type returns the wire event name emitted when a login attempt fails on a bad
// credential.
func (e UserLoginFailedEvent) Type() string { return "user.login.failed" }

// AggregateID ties the event back to the UserAccount that produced it.
func (e UserLoginFailedEvent) AggregateID() string { return e.UserID }

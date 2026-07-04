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

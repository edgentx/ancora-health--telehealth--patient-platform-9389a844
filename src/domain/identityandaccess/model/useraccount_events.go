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

// PasswordResetRequestedEvent records that a single-use password-reset token was
// issued for a user account after passing every reset invariant. Its Type() is
// the wire contract "user.password.reset.requested".
type PasswordResetRequestedEvent struct {
	// UserID is the identity of the UserAccount the reset was requested for.
	UserID string
	// TenantID scopes the account to a single tenant.
	TenantID string
	// Email is the account email the reset token was issued to.
	Email string
	// ResetToken is the freshly issued, single-use reset token.
	ResetToken string
	// ExpiresAt is the instant after which the reset token can no longer be used
	// to change the credential.
	ExpiresAt time.Time
}

// Type returns the wire event name emitted when a password reset is requested.
func (e PasswordResetRequestedEvent) Type() string { return "user.password.reset.requested" }

// AggregateID ties the event back to the UserAccount that produced it.
func (e PasswordResetRequestedEvent) AggregateID() string { return e.UserID }

package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// UserAuthenticatedEventType is the stable type name emitted when a login
// attempt succeeds and a session may be issued.
const UserAuthenticatedEventType = "user.authenticated"

// UserLoginFailedEventType is the stable type name emitted when a login attempt
// is rejected on credential or second-factor grounds and the failure is tallied
// against the account's credential-stuffing protection.
const UserLoginFailedEventType = "user.login.failed"

// UserAuthenticatedEvent is emitted when an AuthenticateUserCmd succeeds: the
// credential and, when enrolled, the second factor were both valid.
type UserAuthenticatedEvent struct {
	// UserID is the identity of the UserAccountAggregate that authenticated.
	UserID string
	// TenantID scopes the account to a single tenant.
	TenantID string
	// Email is the address the account authenticated with.
	Email string
}

// Type identifies the event kind.
func (e UserAuthenticatedEvent) Type() string { return UserAuthenticatedEventType }

// AggregateID ties the event back to the user account that produced it.
func (e UserAuthenticatedEvent) AggregateID() string { return e.UserID }

// UserLoginFailedEvent is emitted when a login attempt fails verification. It
// carries the running failed-attempt count so downstream projections can react
// to credential-stuffing pressure, and a machine-readable reason.
type UserLoginFailedEvent struct {
	// UserID is the identity of the UserAccountAggregate the attempt targeted.
	UserID string
	// TenantID scopes the account to a single tenant.
	TenantID string
	// Email is the address the failed attempt was made against.
	Email string
	// FailedAttempts is the consecutive-failure count after this attempt.
	FailedAttempts int
	// Reason is a short, machine-readable cause for the failure.
	Reason string
}

// Type identifies the event kind.
func (e UserLoginFailedEvent) Type() string { return UserLoginFailedEventType }

// AggregateID ties the event back to the user account that produced it.
func (e UserLoginFailedEvent) AggregateID() string { return e.UserID }

// Compile-time assertions that both events satisfy the DomainEvent contract.
var (
	_ shared.DomainEvent = UserAuthenticatedEvent{}
	_ shared.DomainEvent = UserLoginFailedEvent{}
)

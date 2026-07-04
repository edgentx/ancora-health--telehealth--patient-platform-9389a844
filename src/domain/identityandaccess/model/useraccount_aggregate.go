// Package model holds the aggregates for the identity-and-access bounded
// context. UserAccountAggregate is a platform user account; commands are
// dispatched through Execute.
package model

import (
	"strings"
	"time"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

// UserAccountAggregate is the aggregate root for an identity-and-access user
// account. It embeds shared.AggregateRoot for version tracking and an
// uncommitted-event buffer, and carries its own identity in ID.
//
// The remaining fields capture the account state that command invariants read:
// whether the email is already claimed by an active account, an active
// credential-stuffing lockout window, the state of any pending password-reset
// token, and the account's MFA posture.
type UserAccountAggregate struct {
	shared.AggregateRoot
	ID string

	// TenantID scopes the account to a single tenant. Email uniqueness is
	// enforced per tenant, so this is set once the account is registered.
	TenantID string
	// Email is the account's login email. Empty until the account registers.
	Email string
	// Role is the account's assigned role, used for role-based routing.
	Role string

	// EmailRegistered reports whether an active account already claims this
	// email within the tenant. Registration must reject a duplicate to keep
	// email unique per tenant.
	EmailRegistered bool
	// LockedUntil is the instant a credential-stuffing lockout elapses. A zero
	// value means the account is not locked.
	LockedUntil time.Time
	// ResetTokenConsumed reports whether the pending password-reset token has
	// already been used. A reset token is single-use.
	ResetTokenConsumed bool
	// ResetTokenExpiresAt is the expiry of the pending password-reset token. A
	// zero value means no reset token is pending.
	ResetTokenExpiresAt time.Time
	// MFAEnrolled reports whether the account has enrolled a second factor.
	MFAEnrolled bool
	// SecondFactorVerified reports whether a valid second factor has been
	// presented for the current attempt.
	SecondFactorVerified bool
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. Unrecognized command types return shared.ErrUnknownCommand.
func (a *UserAccountAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch c := cmd.(type) {
	case RegisterUserCmd:
		return a.registerUser(c)
	default:
		return nil, shared.ErrUnknownCommand
	}
}

// registerUser handles RegisterUserCmd: it validates the command input, enforces
// the account invariants, then emits a UserRegisteredEvent and buffers it on the
// aggregate. Guards are ordered so the strongest prohibitions (a duplicate email
// or a locked account) are reported before the credential and MFA checks.
func (a *UserAccountAggregate) registerUser(cmd RegisterUserCmd) ([]shared.DomainEvent, error) {
	if strings.TrimSpace(cmd.Email) == "" {
		return nil, ErrMissingEmail
	}
	if strings.TrimSpace(cmd.Password) == "" {
		return nil, ErrMissingPassword
	}
	if strings.TrimSpace(cmd.Role) == "" {
		return nil, ErrMissingRole
	}
	if strings.TrimSpace(cmd.TenantId) == "" {
		return nil, ErrMissingTenant
	}

	// Invariant: email must be unique per tenant and cannot be reused by an
	// active account.
	if a.EmailRegistered {
		return nil, ErrEmailAlreadyRegistered
	}
	// Invariant: an account locked by credential-stuffing protection cannot
	// authenticate until the lockout window elapses.
	if !a.LockedUntil.IsZero() && a.LockedUntil.After(time.Now()) {
		return nil, ErrAccountLocked
	}
	// Invariant: a password reset token is single-use and must be unexpired to
	// change the credential.
	if a.ResetTokenConsumed ||
		(!a.ResetTokenExpiresAt.IsZero() && !a.ResetTokenExpiresAt.After(time.Now())) {
		return nil, ErrResetTokenInvalid
	}
	// Invariant: MFA-enrolled accounts must present a valid second factor before
	// a session is issued.
	if a.MFAEnrolled && !a.SecondFactorVerified {
		return nil, ErrSecondFactorRequired
	}

	evt := UserRegisteredEvent{
		UserID:   a.ID,
		TenantID: cmd.TenantId,
		Email:    cmd.Email,
		Role:     cmd.Role,
		Route:    routeForRole(cmd.Role),
	}

	a.apply(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// apply mutates aggregate state from a domain event. It is the single place
// state changes, so the same function serves both command handling and future
// event replay when rehydrating the aggregate from the store.
func (a *UserAccountAggregate) apply(evt UserRegisteredEvent) {
	a.TenantID = evt.TenantID
	a.Email = evt.Email
	a.Role = evt.Role
	a.EmailRegistered = true
}

// routeForRole resolves the landing route an account is directed to based on its
// role. Unknown roles fall back to the patient portal, the least-privileged
// surface.
func routeForRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "clinician", "provider":
		return "/clinician"
	case "admin", "administrator":
		return "/admin"
	case "staff":
		return "/staff"
	default:
		return "/patient"
	}
}

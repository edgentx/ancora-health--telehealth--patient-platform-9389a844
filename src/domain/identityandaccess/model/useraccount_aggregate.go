// Package model holds the aggregates for the identity-and-access bounded
// context. UserAccountAggregate is a platform user account; commands are
// dispatched through Execute.
package model

import (
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

// resetTokenTTL is the window a freshly issued password-reset token stays valid.
const resetTokenTTL = time.Hour

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
	case InitiatePasswordResetCmd:
		return a.initiatePasswordReset(c)
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

// initiatePasswordReset handles InitiatePasswordResetCmd: it validates the
// command input, enforces the account invariants, then issues a single-use
// reset token by emitting a PasswordResetRequestedEvent and buffering it on the
// aggregate. The invariant guards mirror registerUser so the same prohibitions
// (a reused email, a lockout window, a stale reset token, or an unmet second
// factor) are reported consistently across every command on this aggregate.
func (a *UserAccountAggregate) initiatePasswordReset(cmd InitiatePasswordResetCmd) ([]shared.DomainEvent, error) {
	if strings.TrimSpace(cmd.Email) == "" {
		return nil, ErrMissingEmail
	}
	// A reset may only be requested for the account's own email. The guard is
	// skipped for a not-yet-registered account whose email is still unset.
	if a.Email != "" && !strings.EqualFold(strings.TrimSpace(a.Email), strings.TrimSpace(cmd.Email)) {
		return nil, ErrEmailMismatch
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
	// change the credential. A new token cannot be issued while a prior one is
	// still consumed or lingering past its expiry.
	if a.ResetTokenConsumed ||
		(!a.ResetTokenExpiresAt.IsZero() && !a.ResetTokenExpiresAt.After(time.Now())) {
		return nil, ErrResetTokenInvalid
	}
	// Invariant: MFA-enrolled accounts must present a valid second factor before
	// a session is issued.
	if a.MFAEnrolled && !a.SecondFactorVerified {
		return nil, ErrSecondFactorRequired
	}

	evt := PasswordResetRequestedEvent{
		UserID:     a.ID,
		TenantID:   a.TenantID,
		Email:      cmd.Email,
		ResetToken: newResetToken(),
		ExpiresAt:  time.Now().Add(resetTokenTTL),
	}

	a.applyPasswordResetRequested(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// applyPasswordResetRequested mutates aggregate state from a
// PasswordResetRequestedEvent. Issuing a token records its expiry and marks it
// unconsumed, so the single-use invariant can be enforced on the next attempt.
func (a *UserAccountAggregate) applyPasswordResetRequested(evt PasswordResetRequestedEvent) {
	a.Email = evt.Email
	a.ResetTokenExpiresAt = evt.ExpiresAt
	a.ResetTokenConsumed = false
}

// newResetToken produces an opaque, single-use password-reset token. It reads
// from a cryptographically secure source; on the rare read failure it falls
// back to a timestamp-derived token so a reset can still be issued.
func newResetToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "reset-" + strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(b)
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

// Package model holds the aggregates for the identity-and-access bounded
// context. UserAccountAggregate is a platform user account; commands are
// dispatched through Execute.
package model

import (
	"crypto/rand"
	"encoding/hex"
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

	// CredentialVerified reports whether the presented password matched the
	// stored credential for the current attempt. Like SecondFactorVerified, it
	// carries the *result* of verification performed by the auth infrastructure
	// (which owns the slow-KDF hash and its constant-time comparison); the domain
	// never holds or compares raw credential material. AuthenticateUserCmd reads
	// this to decide between a successful authentication and a tracked failure.
	CredentialVerified bool
	// FailedLoginAttempts is the running count of failed login attempts, tracked
	// by AuthenticateUserCmd toward the credential-stuffing lockout threshold.
	FailedLoginAttempts int
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. Unrecognized command types return shared.ErrUnknownCommand.
func (a *UserAccountAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch c := cmd.(type) {
	case RegisterUserCmd:
		return a.registerUser(c)
	case LockAccountCmd:
		return a.lockAccount(c)
	case AuthenticateUserCmd:
		return a.authenticateUser(c)
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

// lockoutWindow is how long a credential-stuffing lock holds before the account
// may authenticate again.
const lockoutWindow = 15 * time.Minute

// lockAccount handles LockAccountCmd: it validates the command input, enforces
// the account invariants, then emits a UserAccountLockedEvent and buffers it on
// the aggregate. The invariant guards mirror registerUser so the strongest
// prohibitions (a duplicate email or an already-active lockout) are reported
// before the credential and MFA checks.
func (a *UserAccountAggregate) lockAccount(cmd LockAccountCmd) ([]shared.DomainEvent, error) {
	if strings.TrimSpace(cmd.AccountId) == "" {
		return nil, ErrMissingAccountID
	}
	if strings.TrimSpace(cmd.Reason) == "" {
		return nil, ErrMissingReason
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

	evt := UserAccountLockedEvent{
		UserID:      a.ID,
		Reason:      cmd.Reason,
		LockedUntil: time.Now().Add(lockoutWindow),
	}

	a.applyLocked(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// applyLocked mutates aggregate state from a UserAccountLockedEvent, opening the
// credential-stuffing lockout window. Keeping mutation here lets the same event
// drive both command handling and future replay when rehydrating from the store.
func (a *UserAccountAggregate) applyLocked(evt UserAccountLockedEvent) {
	a.LockedUntil = evt.LockedUntil
}

// authenticateUser handles AuthenticateUserCmd: it validates the presented
// credentials, enforces the account invariants, then verifies the credential.
// The invariant guards mirror registerUser so the strongest prohibitions (a
// duplicate email or an active lockout) are reported before the credential and
// MFA checks. A command that clears the invariants always executes successfully:
// a matching credential emits UserAuthenticatedEvent, while a mismatch emits
// UserLoginFailedEvent so the failed attempt is tracked toward the lockout
// threshold rather than surfacing as a domain error.
func (a *UserAccountAggregate) authenticateUser(cmd AuthenticateUserCmd) ([]shared.DomainEvent, error) {
	if strings.TrimSpace(cmd.Email) == "" {
		return nil, ErrMissingEmail
	}
	if strings.TrimSpace(cmd.Password) == "" {
		return nil, ErrMissingPassword
	}
	if strings.TrimSpace(cmd.MfaCode) == "" {
		return nil, ErrMissingMfaCode
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

	// A mismatched credential is not a domain error: it is a tracked failed
	// attempt that feeds credential-stuffing protection. Verification itself is
	// performed upstream by the auth infrastructure and surfaced as
	// CredentialVerified, so no raw password is compared here.
	if !a.CredentialVerified {
		evt := UserLoginFailedEvent{
			UserID:         a.ID,
			Email:          cmd.Email,
			FailedAttempts: a.FailedLoginAttempts + 1,
		}

		a.applyLoginFailed(evt)
		a.AddEvent(evt)
		a.Version++

		return []shared.DomainEvent{evt}, nil
	}

	evt := UserAuthenticatedEvent{
		UserID:   a.ID,
		TenantID: a.TenantID,
		Email:    cmd.Email,
	}

	a.applyAuthenticated(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// applyAuthenticated mutates aggregate state from a UserAuthenticatedEvent,
// clearing the failed-attempt counter now that a login has succeeded. Keeping
// mutation here lets the same event drive both command handling and future
// replay when rehydrating from the store.
func (a *UserAccountAggregate) applyAuthenticated(evt UserAuthenticatedEvent) {
	a.FailedLoginAttempts = 0
}

// applyLoginFailed mutates aggregate state from a UserLoginFailedEvent, advancing
// the failed-attempt counter that credential-stuffing protection reads. Keeping
// mutation here lets the same event drive both command handling and future
// replay when rehydrating from the store.
func (a *UserAccountAggregate) applyLoginFailed(evt UserLoginFailedEvent) {
	a.FailedLoginAttempts = evt.FailedAttempts
}

// resetTokenWindow is how long an issued password-reset token stays valid before
// it expires and can no longer change the credential.
const resetTokenWindow = 30 * time.Minute

// initiatePasswordReset handles InitiatePasswordResetCmd: it validates the
// command input, enforces the account invariants, then issues a single-use reset
// token by emitting UserPasswordResetRequestedEvent and buffering it on the
// aggregate. The invariant guards mirror the other handlers so the strongest
// prohibitions (a duplicate email or an active lockout) are reported before the
// reset-token and MFA checks; in particular the reset-token guard rejects a
// request while a prior token has already been consumed or remains unexpired, so
// a fresh token is only minted once no other pending token is in play.
func (a *UserAccountAggregate) initiatePasswordReset(cmd InitiatePasswordResetCmd) ([]shared.DomainEvent, error) {
	if strings.TrimSpace(cmd.Email) == "" {
		return nil, ErrMissingEmail
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
		(!a.ResetTokenExpiresAt.IsZero() && a.ResetTokenExpiresAt.After(time.Now())) {
		return nil, ErrResetTokenInvalid
	}
	// Invariant: MFA-enrolled accounts must present a valid second factor before
	// a session is issued.
	if a.MFAEnrolled && !a.SecondFactorVerified {
		return nil, ErrSecondFactorRequired
	}

	evt := UserPasswordResetRequestedEvent{
		UserID:    a.ID,
		Email:     cmd.Email,
		Token:     newResetToken(),
		ExpiresAt: time.Now().Add(resetTokenWindow),
	}

	a.applyPasswordResetRequested(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// applyPasswordResetRequested mutates aggregate state from a
// UserPasswordResetRequestedEvent, opening a fresh single-use reset-token window.
// Keeping mutation here lets the same event drive both command handling and
// future replay when rehydrating from the store.
func (a *UserAccountAggregate) applyPasswordResetRequested(evt UserPasswordResetRequestedEvent) {
	a.ResetTokenExpiresAt = evt.ExpiresAt
	a.ResetTokenConsumed = false
}

// newResetToken mints an unpredictable single-use password-reset token. The
// domain issues the token but never stores raw credential material; downstream
// infrastructure delivers it and later validates the presented value.
func newResetToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand should not fail; fall back to a time-seeded token so the
		// reset can still proceed rather than blocking the account.
		return "reset-" + time.Now().UTC().Format("20060102T150405.000000000")
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

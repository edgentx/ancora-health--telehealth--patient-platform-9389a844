// Package model holds the aggregates for the identity-and-access bounded
// context. UserAccountAggregate models a platform user account and handles
// authentication through the Execute(cmd) pattern.
package model

import (
	"crypto/sha256"
	"crypto/subtle"
	"strings"
	"time"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

// maxFailedLoginAttempts is the number of consecutive failed logins that trips
// credential-stuffing protection and locks the account.
const maxFailedLoginAttempts = 5

// lockoutWindow is how long an account stays locked once credential-stuffing
// protection engages.
const lockoutWindow = 15 * time.Minute

// UserAccountAggregate is the aggregate root for an identity-and-access user
// account. It embeds shared.AggregateRoot for version tracking and an
// uncommitted-event buffer, carries its own identity in ID, and holds the
// credential and protection state that AuthenticateUserCmd invariants read.
type UserAccountAggregate struct {
	shared.AggregateRoot
	ID string

	// TenantID scopes the account to a single tenant.
	TenantID string
	// Email is the account's login address.
	Email string
	// Active reports whether this account is the single active owner of its
	// email within the tenant. Only an active, uniquely-owned email may
	// authenticate.
	Active bool

	// PasswordHash is the SHA-256 hash of the account's current password.
	PasswordHash string

	// MFAEnrolled reports whether the account requires a second factor.
	MFAEnrolled bool
	// MFACodeHash is the SHA-256 hash of the currently valid second-factor code.
	MFACodeHash string

	// FailedAttempts is the count of consecutive failed logins since the last
	// success. It drives credential-stuffing lockout.
	FailedAttempts int
	// LockedUntil is the instant an active lockout window ends. A zero value
	// means the account is not locked.
	LockedUntil time.Time

	// ResetTokenPending reports whether an outstanding password reset token
	// exists for the account.
	ResetTokenPending bool
	// ResetTokenConsumed reports whether the pending reset token has already been
	// used. A reset token is single-use.
	ResetTokenConsumed bool
	// ResetTokenExpiresAt is when the pending reset token expires. A zero value
	// means it does not expire.
	ResetTokenExpiresAt time.Time
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. Unrecognized command types return shared.ErrUnknownCommand.
func (a *UserAccountAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch c := cmd.(type) {
	case AuthenticateUserCmd:
		return a.authenticateUser(c)
	default:
		return nil, shared.ErrUnknownCommand
	}
}

// authenticateUser handles AuthenticateUserCmd: it validates input, enforces the
// account invariants, verifies the credential and (when enrolled) the second
// factor, and tallies failed attempts. Guards are ordered so the strongest
// prohibitions (unusable account, active lockout) are reported before the
// credential is ever checked.
//
// A wrong password is a normal business outcome, not a rejection: it emits a
// UserLoginFailedEvent (tallying the attempt and possibly engaging the lockout)
// and returns with no error. Invariant violations, by contrast, return a domain
// error and emit nothing.
func (a *UserAccountAggregate) authenticateUser(cmd AuthenticateUserCmd) ([]shared.DomainEvent, error) {
	if strings.TrimSpace(cmd.Email) == "" || cmd.Password == "" {
		return nil, ErrMissingCredentials
	}

	// Invariant: an email must be unique per tenant and owned by an active
	// account; a deactivated or reused email cannot authenticate.
	if !a.Active {
		return nil, ErrEmailNotUnique
	}

	// Invariant: an account locked by credential-stuffing protection cannot
	// authenticate until the lockout window elapses.
	if a.isLocked() {
		return nil, ErrAccountLocked
	}

	// Invariant: a pending password reset token is single-use and must be
	// unexpired; a consumed or expired token has left the credential in a state
	// that cannot establish a session.
	if a.ResetTokenPending && (a.ResetTokenConsumed || a.resetTokenExpired()) {
		return nil, ErrResetTokenInvalid
	}

	// Verify the credential. A mismatch is tallied and emitted as a login
	// failure rather than rejected outright.
	if !a.credentialMatches(cmd.Password) {
		evt := a.buildLoginFailed("invalid_password")
		a.apply(evt)
		a.AddEvent(evt)
		a.Version++
		return []shared.DomainEvent{evt}, nil
	}

	// Invariant: MFA-enrolled accounts must present a valid second factor before
	// a session is issued.
	if a.MFAEnrolled && !a.secondFactorMatches(cmd.MFACode) {
		return nil, ErrMFARequired
	}

	evt := UserAuthenticatedEvent{
		UserID:   a.ID,
		TenantID: a.TenantID,
		Email:    a.Email,
	}
	a.apply(evt)
	a.AddEvent(evt)
	a.Version++
	return []shared.DomainEvent{evt}, nil
}

// apply mutates aggregate state from a domain event. It is the single place
// state changes, so the same function serves both command handling and future
// event replay when rehydrating the aggregate from the store.
func (a *UserAccountAggregate) apply(evt shared.DomainEvent) {
	switch evt.(type) {
	case UserAuthenticatedEvent:
		// A successful login clears any accumulated failure pressure.
		a.FailedAttempts = 0
		a.LockedUntil = time.Time{}
	case UserLoginFailedEvent:
		a.FailedAttempts++
		// Engage the lockout window once the threshold is reached.
		if a.FailedAttempts >= maxFailedLoginAttempts {
			a.LockedUntil = time.Now().Add(lockoutWindow)
		}
	}
}

// buildLoginFailed constructs the failure event for the *upcoming* attempt,
// reflecting the failed-attempt count as it will stand once apply increments it.
func (a *UserAccountAggregate) buildLoginFailed(reason string) UserLoginFailedEvent {
	return UserLoginFailedEvent{
		UserID:         a.ID,
		TenantID:       a.TenantID,
		Email:          a.Email,
		FailedAttempts: a.FailedAttempts + 1,
		Reason:         reason,
	}
}

// isLocked reports whether the account is inside an active lockout window.
func (a *UserAccountAggregate) isLocked() bool {
	return !a.LockedUntil.IsZero() && a.LockedUntil.After(time.Now())
}

// resetTokenExpired reports whether the pending reset token's expiry has passed.
func (a *UserAccountAggregate) resetTokenExpired() bool {
	return !a.ResetTokenExpiresAt.IsZero() && !a.ResetTokenExpiresAt.After(time.Now())
}

// credentialMatches reports whether the presented password hashes to the stored
// credential, using a constant-time comparison to avoid timing leaks.
func (a *UserAccountAggregate) credentialMatches(password string) bool {
	return constantTimeEquals(hashSecret(password), a.PasswordHash)
}

// secondFactorMatches reports whether the presented code hashes to the stored
// second factor, using a constant-time comparison.
func (a *UserAccountAggregate) secondFactorMatches(code string) bool {
	if strings.TrimSpace(code) == "" {
		return false
	}
	return constantTimeEquals(hashSecret(code), a.MFACodeHash)
}

// hashSecret returns the lowercase hex SHA-256 of a secret. It is the single
// hashing routine used for both passwords and second-factor codes so stored and
// presented values are compared on the same footing.
func hashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	const hexdigits = "0123456789abcdef"
	out := make([]byte, len(sum)*2)
	for i, b := range sum {
		out[i*2] = hexdigits[b>>4]
		out[i*2+1] = hexdigits[b&0x0f]
	}
	return string(out)
}

// constantTimeEquals compares two hex digests without leaking their similarity
// through timing.
func constantTimeEquals(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// Compile-time assertion that UserAccountAggregate satisfies the domain
// aggregate contract for Execute and version tracking. ID is exposed as a field
// rather than a method, matching the other aggregates in this codebase.
var _ interface {
	Execute(cmd interface{}) ([]shared.DomainEvent, error)
	GetVersion() int
} = (*UserAccountAggregate)(nil)

package model

import "errors"

var (
	// ErrMissingCredentials is returned when AuthenticateUserCmd omits the email
	// or password required to attempt authentication.
	ErrMissingCredentials = errors.New("user account: email and password are required")

	// ErrEmailNotUnique is returned when authentication targets an account whose
	// email is not owned by a single active account. Invariant: an email must be
	// unique per tenant and cannot be reused by an active account.
	ErrEmailNotUnique = errors.New("user account: email is not unique for an active account")

	// ErrAccountLocked is returned when authentication is attempted during an
	// active lockout window. Invariant: an account locked by credential-stuffing
	// protection cannot authenticate until the lockout window elapses.
	ErrAccountLocked = errors.New("user account: account is locked by credential-stuffing protection")

	// ErrResetTokenInvalid is returned when a pending password reset token has
	// already been consumed or has expired. Invariant: a password reset token is
	// single-use and must be unexpired to change the credential.
	ErrResetTokenInvalid = errors.New("user account: password reset token is consumed or expired")

	// ErrMFARequired is returned when an MFA-enrolled account presents no valid
	// second factor. Invariant: MFA-enrolled accounts must present a valid second
	// factor before a session is issued.
	ErrMFARequired = errors.New("user account: a valid second factor is required")
)

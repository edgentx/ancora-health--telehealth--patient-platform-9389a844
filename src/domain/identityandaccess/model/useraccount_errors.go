package model

import "errors"

var (
	// ErrEmailAlreadyRegistered is returned when registration is attempted with
	// an email already claimed by an active account in the tenant. Invariant:
	// email must be unique per tenant and cannot be reused by an active account.
	ErrEmailAlreadyRegistered = errors.New("user account: email is already registered for this tenant")

	// ErrAccountLocked is returned when registration is attempted on an account
	// still inside a credential-stuffing lockout window. Invariant: an account
	// locked by credential-stuffing protection cannot authenticate until the
	// lockout window elapses.
	ErrAccountLocked = errors.New("user account: account is locked by credential-stuffing protection")

	// ErrResetTokenInvalid is returned when a pending password-reset token has
	// already been used or has expired. Invariant: a password reset token is
	// single-use and must be unexpired to change the credential.
	ErrResetTokenInvalid = errors.New("user account: password reset token is used or expired")

	// ErrSecondFactorRequired is returned when an MFA-enrolled account has not
	// presented a valid second factor. Invariant: MFA-enrolled accounts must
	// present a valid second factor before a session is issued.
	ErrSecondFactorRequired = errors.New("user account: a valid second factor is required")

	// ErrMissingEmail is returned when RegisterUserCmd omits the email.
	ErrMissingEmail = errors.New("user account: email is required")

	// ErrMissingPassword is returned when RegisterUserCmd omits the password.
	ErrMissingPassword = errors.New("user account: password is required")

	// ErrMissingRole is returned when RegisterUserCmd omits the role.
	ErrMissingRole = errors.New("user account: role is required")

	// ErrMissingTenant is returned when RegisterUserCmd omits the tenant id.
	ErrMissingTenant = errors.New("user account: tenant id is required")

	// ErrMissingAccountID is returned when LockAccountCmd omits the account id.
	ErrMissingAccountID = errors.New("user account: account id is required")

	// ErrMissingReason is returned when LockAccountCmd omits the lock reason.
	ErrMissingReason = errors.New("user account: lock reason is required")
)

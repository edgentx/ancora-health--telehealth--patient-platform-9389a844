package model

// AuthenticateUserCmd requests that a UserAccount verify a login attempt. It
// carries the email identifying the account, the presented password, and the
// second-factor code (empty when the account is not MFA-enrolled).
type AuthenticateUserCmd struct {
	// Email identifies the account being authenticated.
	Email string
	// Password is the plaintext credential presented for verification. It is
	// hashed before comparison and never persisted.
	Password string
	// MFACode is the one-time second-factor code. MFA-enrolled accounts must
	// present a valid code before a session is issued.
	MFACode string
}

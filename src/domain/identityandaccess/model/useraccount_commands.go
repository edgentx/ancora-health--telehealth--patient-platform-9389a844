package model

// RegisterUserCmd requests that a UserAccount be registered on the platform. It
// carries the login credential, the role that drives post-registration routing,
// and the owning tenant that scopes email uniqueness.
type RegisterUserCmd struct {
	// Email is the account's login email. It must be unique per tenant.
	Email string
	// Password is the initial credential supplied at registration.
	Password string
	// Role is the account role (e.g. "patient", "clinician", "admin") used to
	// route the account after registration.
	Role string
	// TenantId scopes the account, and its email uniqueness, to a single tenant.
	TenantId string
}

// AuthenticateUserCmd requests that a UserAccount authenticate a login attempt.
// It carries the presented credentials — the login email, the password, and the
// second-factor code for MFA-enrolled accounts. The handler verifies the
// credentials and MFA, emitting user.authenticated on success or, when the
// presented password does not match, user.login.failed so the failed attempt is
// tracked toward the credential-stuffing lockout threshold.
type AuthenticateUserCmd struct {
	// Email is the login email presented for the authentication attempt.
	Email string
	// Password is the credential presented for the authentication attempt.
	Password string
	// MfaCode is the second-factor code presented for MFA-enrolled accounts.
	MfaCode string
}

// LockAccountCmd requests that a UserAccount be locked, typically after its
// failed-attempt threshold is crossed by credential-stuffing protection. It
// carries the identity of the account to lock and the reason the lock is being
// applied, which is recorded on the emitted event for audit.
type LockAccountCmd struct {
	// AccountId identifies the UserAccount to lock.
	AccountId string
	// Reason records why the account is being locked (e.g. "failed-attempt
	// threshold exceeded").
	Reason string
}

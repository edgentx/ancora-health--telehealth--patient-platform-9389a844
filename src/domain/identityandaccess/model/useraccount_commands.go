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

// InitiatePasswordResetCmd requests that a single-use password-reset token be
// issued for a UserAccount. It carries the login email the reset was requested
// for; the account identity is taken from the aggregate the command is executed
// against.
type InitiatePasswordResetCmd struct {
	// Email is the login email the password reset was requested for. It must be
	// present for the command to be accepted.
	Email string
}

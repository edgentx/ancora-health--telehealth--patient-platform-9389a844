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

// InitiatePasswordResetCmd requests that a single-use password-reset token be
// issued for the account identified by Email. Executing it emits a
// user.password.reset.requested event once the account invariants hold.
type InitiatePasswordResetCmd struct {
	// Email identifies the account whose credential is being reset. It must
	// belong to an active account within the aggregate's tenant.
	Email string
}

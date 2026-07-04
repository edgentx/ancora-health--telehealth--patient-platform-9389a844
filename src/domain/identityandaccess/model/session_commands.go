package model

import "time"

// IssueSessionCmd requests that a Session be issued for an account that has just
// authenticated. It carries the authenticated account's identity, the role that
// determines the session's maximum lifetime, the device fingerprint bound to the
// session, and the requested lifetime the caller wants the session to hold.
type IssueSessionCmd struct {
	// AccountId identifies the authenticated account the session is issued to.
	AccountId string
	// Role is the account role (e.g. "patient", "clinician", "admin"). It selects
	// the configured per-role maximum session lifetime.
	Role string
	// DeviceFingerprint binds the session to the device that authenticated, so a
	// stolen token replayed from another device can be detected.
	DeviceFingerprint string
	// RequestedLifetime is how long the caller asks the session to remain valid.
	// It must not exceed the configured per-role maximum lifetime.
	RequestedLifetime time.Duration
}

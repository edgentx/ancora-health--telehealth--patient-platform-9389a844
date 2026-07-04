package model

// IssueDataKeyCmd requests that the CryptoKeyEnvelope generate and wrap a new
// field-level data encryption key (DEK). It carries the owning tenant and the
// PHI field classification the resulting key will protect.
type IssueDataKeyCmd struct {
	// TenantId scopes the issued data key to a single tenant.
	TenantId string
	// FieldClass identifies the PHI field classification (e.g. "ssn",
	// "diagnosis") the data key will encrypt.
	FieldClass string
}

// RotateMasterKeyCmd requests that the CryptoKeyEnvelope rewrap its active data
// encryption keys under a new master (wrapping) key. It carries the identity of
// the new master key that will back the envelope after rotation.
type RotateMasterKeyCmd struct {
	// NewMasterKeyId identifies the master key the envelope's data keys are
	// rewrapped under. It must be non-empty.
	NewMasterKeyId string
}

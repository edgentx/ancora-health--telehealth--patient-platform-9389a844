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

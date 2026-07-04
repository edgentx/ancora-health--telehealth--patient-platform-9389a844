package model

// DataKeyIssuedEvent records that a field-level data encryption key was
// successfully generated and wrapped by the envelope's active master key. Its
// Type() is the wire contract "crypto.datakey.issued".
type DataKeyIssuedEvent struct {
	// EnvelopeID is the identity of the CryptoKeyEnvelope that issued the key.
	EnvelopeID string
	// TenantID scopes the issued data key to a single tenant.
	TenantID string
	// FieldClass identifies the PHI field classification the data key protects.
	FieldClass string
}

// Type returns the wire event name emitted when a data key is issued.
func (e DataKeyIssuedEvent) Type() string { return "crypto.datakey.issued" }

// AggregateID ties the event back to the CryptoKeyEnvelope that produced it.
func (e DataKeyIssuedEvent) AggregateID() string { return e.EnvelopeID }

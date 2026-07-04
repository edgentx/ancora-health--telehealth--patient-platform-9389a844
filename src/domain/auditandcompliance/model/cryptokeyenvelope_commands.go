package model

// RotateMasterKeyCmd instructs a CryptoKeyEnvelopeAggregate to rewrap its active
// data encryption keys under a new master key. NewMasterKeyID identifies the
// master key the envelope's data keys should be rewrapped beneath; it must be a
// non-empty identifier.
type RotateMasterKeyCmd struct {
	NewMasterKeyID string
}

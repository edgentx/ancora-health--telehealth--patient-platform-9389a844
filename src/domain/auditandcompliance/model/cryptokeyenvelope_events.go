package model

// EventTypeMasterKeyRotated is the canonical type string emitted when a
// CryptoKeyEnvelopeAggregate successfully rotates its master key.
const EventTypeMasterKeyRotated = "crypto.masterkey.rotated"

// MasterKeyRotatedEvent records that an envelope's data keys were rewrapped from
// PreviousMasterKeyID to NewMasterKeyID. RewrappedKeyIDs lists the data
// encryption keys that were rewrapped as part of the rotation.
type MasterKeyRotatedEvent struct {
	EnvelopeID          string
	PreviousMasterKeyID string
	NewMasterKeyID      string
	RewrappedKeyIDs     []string
}

// Type returns the canonical event type string.
func (e MasterKeyRotatedEvent) Type() string { return EventTypeMasterKeyRotated }

// AggregateID ties the event back to the CryptoKeyEnvelopeAggregate that
// produced it.
func (e MasterKeyRotatedEvent) AggregateID() string { return e.EnvelopeID }

// Package model holds the aggregates for the audit-and-compliance bounded
// context. CryptoKeyEnvelopeAggregate models a wrapped cryptographic key
// envelope and handles the commands that mutate it.
package model

import (
	"errors"
	"time"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

// Domain errors returned when a command would violate a CryptoKeyEnvelope
// invariant. Each corresponds to a compliance rule enforced before any PHI key
// material is rewrapped or issued.
var (
	// ErrMasterKeyIDRequired is returned when a rotation is requested without a
	// target master key identifier.
	ErrMasterKeyIDRequired = errors.New("crypto key envelope: new master key id is required")

	// ErrNoActiveMasterKey enforces: a data encryption key must be wrapped by an
	// active master key before it can be issued.
	ErrNoActiveMasterKey = errors.New("crypto key envelope: envelope has no active master key")

	// ErrEnvelopeExpired enforces: PHI ciphertext may only be produced with a
	// non-expired, non-revoked key envelope.
	ErrEnvelopeExpired = errors.New("crypto key envelope: envelope is expired")

	// ErrEnvelopeRevoked enforces: a revoked key envelope can never be used to
	// encrypt new PHI.
	ErrEnvelopeRevoked = errors.New("crypto key envelope: envelope is revoked")
)

// WrappedDataKey is a data encryption key held by the envelope, recorded
// together with the master key that currently wraps it.
type WrappedDataKey struct {
	KeyID                string
	WrappedByMasterKeyID string
}

// CryptoKeyEnvelopeAggregate is the aggregate root for an audit-and-compliance
// crypto key envelope. It embeds shared.AggregateRoot for version tracking and
// an uncommitted-event buffer, and tracks the master key that wraps its data
// keys along with the lifecycle state that governs whether it may still be used
// to protect PHI.
type CryptoKeyEnvelopeAggregate struct {
	shared.AggregateRoot
	ID string

	// MasterKeyID is the master key that currently wraps this envelope's data
	// keys. MasterKeyActive reports whether that master key is active.
	MasterKeyID     string
	MasterKeyActive bool

	// Revoked marks the envelope as permanently unusable for new PHI.
	Revoked bool

	// ExpiresAt is the envelope's expiry instant. The zero value means the
	// envelope does not expire.
	ExpiresAt time.Time

	// DataKeys are the data encryption keys wrapped by this envelope.
	DataKeys []WrappedDataKey

	// now supplies the current time for expiry checks; it defaults to
	// time.Now when nil, and can be overridden for deterministic testing.
	now func() time.Time
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. Recognized commands are dispatched to their handlers; any other
// command type falls through to shared.ErrUnknownCommand.
func (a *CryptoKeyEnvelopeAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch c := cmd.(type) {
	case RotateMasterKeyCmd:
		return a.rotateMasterKey(c)
	default:
		return nil, shared.ErrUnknownCommand
	}
}

// rotateMasterKey rewraps every data key under cmd.NewMasterKeyID and emits a
// crypto.masterkey.rotated event. It enforces the envelope invariants first and
// rejects the command with a domain error if any are violated, leaving the
// aggregate unchanged.
func (a *CryptoKeyEnvelopeAggregate) rotateMasterKey(cmd RotateMasterKeyCmd) ([]shared.DomainEvent, error) {
	if cmd.NewMasterKeyID == "" {
		return nil, ErrMasterKeyIDRequired
	}
	if err := a.ensureUsableForPHI(); err != nil {
		return nil, err
	}

	previousMasterKeyID := a.MasterKeyID
	rewrapped := make([]string, 0, len(a.DataKeys))
	for i := range a.DataKeys {
		a.DataKeys[i].WrappedByMasterKeyID = cmd.NewMasterKeyID
		rewrapped = append(rewrapped, a.DataKeys[i].KeyID)
	}
	a.MasterKeyID = cmd.NewMasterKeyID
	a.MasterKeyActive = true
	a.Version++

	event := MasterKeyRotatedEvent{
		EnvelopeID:          a.ID,
		PreviousMasterKeyID: previousMasterKeyID,
		NewMasterKeyID:      cmd.NewMasterKeyID,
		RewrappedKeyIDs:     rewrapped,
	}
	a.AddEvent(event)
	return []shared.DomainEvent{event}, nil
}

// ensureUsableForPHI checks the invariants that must hold before an envelope may
// be used to protect new PHI: it must be wrapped by an active master key, and it
// must be neither revoked nor expired.
func (a *CryptoKeyEnvelopeAggregate) ensureUsableForPHI() error {
	if a.Revoked {
		return ErrEnvelopeRevoked
	}
	if a.MasterKeyID == "" || !a.MasterKeyActive {
		return ErrNoActiveMasterKey
	}
	if !a.ExpiresAt.IsZero() && !a.ExpiresAt.After(a.clock()) {
		return ErrEnvelopeExpired
	}
	return nil
}

// clock returns the aggregate's time source, defaulting to time.Now.
func (a *CryptoKeyEnvelopeAggregate) clock() time.Time {
	if a.now != nil {
		return a.now()
	}
	return time.Now()
}

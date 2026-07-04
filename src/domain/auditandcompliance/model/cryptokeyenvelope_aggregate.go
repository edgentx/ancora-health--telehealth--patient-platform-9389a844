// Package model holds the aggregates for the audit-and-compliance bounded
// context. CryptoKeyEnvelopeAggregate is the scaffold for a wrapped
// cryptographic key envelope; command handling is filled in by later stories.
package model

import (
	"time"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

// CryptoKeyEnvelopeAggregate is the aggregate root for an audit-and-compliance
// crypto key envelope. It embeds shared.AggregateRoot for version tracking and
// an uncommitted-event buffer, and carries its own identity in ID.
//
// The remaining fields capture the envelope state that command invariants read:
// whether the wrapping master key is active, when the envelope expires, and
// whether it has been revoked.
type CryptoKeyEnvelopeAggregate struct {
	shared.AggregateRoot
	ID string

	// MasterKeyActive reports whether the master (wrapping) key backing this
	// envelope is currently active. A data key can only be wrapped by an active
	// master key.
	MasterKeyActive bool
	// ExpiresAt is the instant the envelope expires. A zero value means the
	// envelope does not expire.
	ExpiresAt time.Time
	// Revoked reports whether the envelope has been revoked. A revoked envelope
	// can never encrypt new PHI.
	Revoked bool
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. Unrecognized command types return shared.ErrUnknownCommand.
func (a *CryptoKeyEnvelopeAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch c := cmd.(type) {
	case IssueDataKeyCmd:
		return a.issueDataKey(c)
	default:
		return nil, shared.ErrUnknownCommand
	}
}

// issueDataKey handles IssueDataKeyCmd: it validates the command input, enforces
// the envelope invariants, then emits a DataKeyIssuedEvent and buffers it on the
// aggregate. Guards are ordered so revocation (the strongest prohibition) is
// reported before expiry, and both before the master-key check.
func (a *CryptoKeyEnvelopeAggregate) issueDataKey(cmd IssueDataKeyCmd) ([]shared.DomainEvent, error) {
	if cmd.TenantId == "" {
		return nil, ErrMissingTenant
	}
	if cmd.FieldClass == "" {
		return nil, ErrMissingFieldClass
	}

	// Invariant: a revoked envelope can never be used to encrypt new PHI.
	if a.Revoked {
		return nil, ErrEnvelopeRevoked
	}
	// Invariant: PHI ciphertext may only be produced with a non-expired envelope.
	if !a.ExpiresAt.IsZero() && !a.ExpiresAt.After(time.Now()) {
		return nil, ErrEnvelopeExpired
	}
	// Invariant: a data key must be wrapped by an active master key.
	if !a.MasterKeyActive {
		return nil, ErrMasterKeyInactive
	}

	event := DataKeyIssuedEvent{
		EnvelopeID: a.ID,
		TenantID:   cmd.TenantId,
		FieldClass: cmd.FieldClass,
	}
	a.AddEvent(event)
	return []shared.DomainEvent{event}, nil
}

package model

import "errors"

var (
	// ErrMasterKeyInactive is returned when a data key issue is attempted on an
	// envelope whose master (wrapping) key is not active. Invariant: a data
	// encryption key must be wrapped by an active master key before it can be
	// issued.
	ErrMasterKeyInactive = errors.New("crypto key envelope: master key is not active")

	// ErrEnvelopeExpired is returned when issuing would produce PHI ciphertext
	// under an expired envelope. Invariant: PHI ciphertext may only be produced
	// with a non-expired, non-revoked key envelope.
	ErrEnvelopeExpired = errors.New("crypto key envelope: envelope is expired")

	// ErrEnvelopeRevoked is returned when issuing is attempted on a revoked
	// envelope. Invariant: a revoked key envelope can never be used to encrypt
	// new PHI.
	ErrEnvelopeRevoked = errors.New("crypto key envelope: envelope is revoked")

	// ErrMissingTenant is returned when IssueDataKeyCmd omits the tenant id.
	ErrMissingTenant = errors.New("crypto key envelope: tenant id is required")

	// ErrMissingFieldClass is returned when IssueDataKeyCmd omits the field
	// classification.
	ErrMissingFieldClass = errors.New("crypto key envelope: field class is required")
)

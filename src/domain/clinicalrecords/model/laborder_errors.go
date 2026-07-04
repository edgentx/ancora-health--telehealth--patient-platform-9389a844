package model

import "errors"

var (
	// ErrMissingLabPatient is returned when PlaceLabOrderCmd omits the patient id.
	ErrMissingLabPatient = errors.New("laborder: patient id is required")

	// ErrMissingLabProvider is returned when PlaceLabOrderCmd omits the provider id.
	ErrMissingLabProvider = errors.New("laborder: provider id is required")

	// ErrMissingTestCode is returned when PlaceLabOrderCmd omits the test code.
	ErrMissingTestCode = errors.New("laborder: test code is required")

	// ErrMissingOrderId is returned when AttachLabResultCmd omits the order id.
	ErrMissingOrderId = errors.New("laborder: order id is required")

	// ErrMissingResultDocumentRef is returned when AttachLabResultCmd omits the
	// result document reference.
	ErrMissingResultDocumentRef = errors.New("laborder: result document reference is required")

	// ErrMissingResultedAt is returned when AttachLabResultCmd omits the resulted-at
	// time.
	ErrMissingResultedAt = errors.New("laborder: resulted-at time is required")

	// ErrProviderNotInCare is returned when the ordering provider does not hold an
	// active care relationship to the patient. Invariant: a lab order must be
	// placed by a provider with an active care relationship to the patient.
	ErrProviderNotInCare = errors.New("laborder: a lab order must be placed by a provider with an active care relationship to the patient")

	// ErrOrderCancelled is returned when acting on a cancelled order. Invariant:
	// results may only be attached to an existing, non-cancelled order.
	ErrOrderCancelled = errors.New("laborder: results may only be attached to an existing, non-cancelled order")

	// ErrResultedCannotRevert is returned when placing would push an already
	// resulted order back to the ordered state. Invariant: a resulted order cannot
	// be reverted to the ordered state.
	ErrResultedCannotRevert = errors.New("laborder: a resulted order cannot be reverted to the ordered state")
)

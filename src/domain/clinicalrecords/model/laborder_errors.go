package model

import "errors"

var (
	// ErrMissingOrder is returned when AttachLabResultCmd omits the order id.
	ErrMissingOrder = errors.New("laborder: order id is required")

	// ErrMissingResultDocumentRef is returned when AttachLabResultCmd omits the
	// result document reference.
	ErrMissingResultDocumentRef = errors.New("laborder: result document ref is required")

	// ErrMissingResultedAt is returned when AttachLabResultCmd omits the resulted
	// timestamp.
	ErrMissingResultedAt = errors.New("laborder: resulted at is required")

	// ErrNoActiveCareRelationship is returned when the ordering provider has no
	// active care relationship with the patient. Invariant: a lab order must be
	// placed by a provider with an active care relationship to the patient.
	ErrNoActiveCareRelationship = errors.New("laborder: a lab order must be placed by a provider with an active care relationship to the patient")

	// ErrOrderCancelled is returned when a result is attached to a cancelled
	// order. Invariant: results may only be attached to an existing, non-cancelled
	// order.
	ErrOrderCancelled = errors.New("laborder: results may only be attached to an existing, non-cancelled order")

	// ErrOrderAlreadyResulted is returned when a result is attached to an order
	// that is already resulted. Invariant: a resulted order cannot be reverted to
	// the ordered state.
	ErrOrderAlreadyResulted = errors.New("laborder: a resulted order cannot be reverted to the ordered state")
)

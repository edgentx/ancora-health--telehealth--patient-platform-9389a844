package model

import "errors"

var (
	// ErrMissingEncounter is returned when GenerateInvoiceCmd omits the encounter
	// id.
	ErrMissingEncounter = errors.New("invoice: encounter id is required")

	// ErrMissingLineItems is returned when GenerateInvoiceCmd omits the line
	// items.
	ErrMissingLineItems = errors.New("invoice: at least one line item is required")

	// ErrMissingPolicy is returned when GenerateInvoiceCmd omits the policy id.
	ErrMissingPolicy = errors.New("invoice: policy id is required")

	// ErrMissingInvoiceID is returned when VoidInvoiceCmd omits the invoice id.
	ErrMissingInvoiceID = errors.New("invoice: invoice id is required")

	// ErrMissingVoidReason is returned when VoidInvoiceCmd omits the void reason.
	ErrMissingVoidReason = errors.New("invoice: void reason is required")

	// ErrEncounterNotCompleted is returned when the invoice is generated from an
	// encounter that is not completed. Invariant: an invoice may only be
	// generated from a completed encounter.
	ErrEncounterNotCompleted = errors.New("invoice: an invoice may only be generated from a completed encounter")

	// ErrPatientResponsibilityMismatch is returned when the recorded patient
	// responsibility does not equal charges minus the verified insurance
	// adjustment and copay. Invariant: patient responsibility must equal charges
	// minus verified insurance adjustment and copay.
	ErrPatientResponsibilityMismatch = errors.New("invoice: patient responsibility must equal charges minus verified insurance adjustment and copay")

	// ErrPaymentExceedsOutstanding is returned when the invoice would be marked
	// paid for more than its outstanding balance. Invariant: an invoice cannot be
	// marked paid for more than its outstanding balance.
	ErrPaymentExceedsOutstanding = errors.New("invoice: an invoice cannot be marked paid for more than its outstanding balance")

	// ErrVoidedInvoicePayment is returned when a voided invoice would receive a
	// further payment. Invariant: a voided invoice cannot receive further
	// payments.
	ErrVoidedInvoicePayment = errors.New("invoice: a voided invoice cannot receive further payments")
)

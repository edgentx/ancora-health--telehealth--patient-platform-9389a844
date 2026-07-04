package model

import "errors"

var (
	// ErrMissingInvoiceID is returned when ApplyInsuranceAdjustmentCmd omits the
	// invoice id.
	ErrMissingInvoiceID = errors.New("invoice: invoice id is required")

	// ErrNegativeAdjustment is returned when the eligibility result carries a
	// negative verified adjustment; verified coverage cannot be negative.
	ErrNegativeAdjustment = errors.New("invoice: verified insurance adjustment must not be negative")

	// ErrNegativeCopay is returned when the eligibility result carries a negative
	// copay; a verified copay cannot be negative.
	ErrNegativeCopay = errors.New("invoice: verified copay must not be negative")

	// ErrEncounterNotCompleted is returned when the invoice was not generated from
	// a completed encounter. Invariant: an invoice may only be generated from a
	// completed encounter.
	ErrEncounterNotCompleted = errors.New("invoice: an invoice may only be generated from a completed encounter")

	// ErrPatientResponsibilityMismatch is returned when the invoice's recorded
	// patient responsibility does not reconcile with its charges, verified
	// insurance adjustment and copay. Invariant: patient responsibility must equal
	// charges minus verified insurance adjustment and copay.
	ErrPatientResponsibilityMismatch = errors.New("invoice: patient responsibility must equal charges minus verified insurance adjustment and copay")

	// ErrOverpaid is returned when the invoice has been marked paid for more than
	// its outstanding balance. Invariant: an invoice cannot be marked paid for
	// more than its outstanding balance.
	ErrOverpaid = errors.New("invoice: an invoice cannot be marked paid for more than its outstanding balance")

	// ErrVoidedInvoice is returned when an adjustment is applied to a voided
	// invoice. Invariant: a voided invoice cannot receive further payments.
	ErrVoidedInvoice = errors.New("invoice: a voided invoice cannot receive further payments")
)

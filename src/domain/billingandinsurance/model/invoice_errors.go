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

	// ErrMissingInvoiceID is returned when ApplyInsuranceAdjustmentCmd omits the
	// invoice id.
	ErrMissingInvoiceID = errors.New("invoice: invoice id is required")

	// ErrUnverifiedEligibility is returned when ApplyInsuranceAdjustmentCmd is
	// given an eligibility result that has not been verified.
	ErrUnverifiedEligibility = errors.New("invoice: eligibility result must be verified")

	// ErrNegativeAdjustment is returned when ApplyInsuranceAdjustmentCmd is given
	// a negative coverage or copay amount.
	ErrNegativeAdjustment = errors.New("invoice: coverage and copay may not be negative")

	// ErrMissingVoidReason is returned when VoidInvoiceCmd omits the void reason.
	ErrMissingVoidReason = errors.New("invoice: void reason is required")
)

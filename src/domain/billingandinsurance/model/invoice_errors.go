package model

import "errors"

var (
	// ErrMissingInvoiceID is returned when VoidInvoiceCmd omits the invoice id.
	ErrMissingInvoiceID = errors.New("invoice: invoice id is required")

	// ErrMissingVoidReason is returned when VoidInvoiceCmd omits the void reason.
	ErrMissingVoidReason = errors.New("invoice: void reason is required")

	// ErrInvoiceNotFromCompletedEncounter is returned when the invoice was not
	// generated from a completed encounter. Invariant: an invoice may only be
	// generated from a completed encounter.
	ErrInvoiceNotFromCompletedEncounter = errors.New("invoice: an invoice may only be generated from a completed encounter")

	// ErrPatientResponsibilityMismatch is returned when the invoice's patient
	// responsibility does not reconcile. Invariant: patient responsibility must
	// equal charges minus verified insurance adjustment and copay.
	ErrPatientResponsibilityMismatch = errors.New("invoice: patient responsibility must equal charges minus verified insurance adjustment and copay")

	// ErrPaidOverOutstandingBalance is returned when the invoice was marked paid
	// for more than its outstanding balance. Invariant: an invoice cannot be
	// marked paid for more than its outstanding balance.
	ErrPaidOverOutstandingBalance = errors.New("invoice: an invoice cannot be marked paid for more than its outstanding balance")

	// ErrVoidedInvoiceReceivedPayment is returned when a voided invoice has
	// received a further payment. Invariant: a voided invoice cannot receive
	// further payments.
	ErrVoidedInvoiceReceivedPayment = errors.New("invoice: a voided invoice cannot receive further payments")
)

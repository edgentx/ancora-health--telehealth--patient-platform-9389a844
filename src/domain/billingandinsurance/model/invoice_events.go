package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// InvoiceAdjustedEventType is the stable wire name emitted when a verified
// insurance eligibility result is applied to an invoice.
const InvoiceAdjustedEventType = "invoice.adjusted"

// InvoiceAdjustedEvent is emitted when an ApplyInsuranceAdjustmentCmd succeeds.
// It records the verified insurance adjustment and copay applied to the invoice
// along with the patient responsibility that remains after them — charges minus
// the verified adjustment and copay.
type InvoiceAdjustedEvent struct {
	// InvoiceID is the identity of the InvoiceAggregate that produced the event.
	InvoiceID string
	// InsuranceAdjustment is the verified coverage, in cents, applied against the
	// invoice charges.
	InsuranceAdjustment int64
	// Copay is the verified copay, in cents, the patient owes.
	Copay int64
	// PatientResponsibility is the amount, in cents, left to the patient after the
	// verified adjustment and copay are applied to the charges.
	PatientResponsibility int64
}

// Type identifies the event kind.
func (e InvoiceAdjustedEvent) Type() string { return InvoiceAdjustedEventType }

// AggregateID ties the event back to the invoice that produced it.
func (e InvoiceAdjustedEvent) AggregateID() string { return e.InvoiceID }

// Compile-time assertion that InvoiceAdjustedEvent satisfies the DomainEvent
// contract.
var _ shared.DomainEvent = InvoiceAdjustedEvent{}

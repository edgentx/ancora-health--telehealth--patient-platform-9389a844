package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// InvoiceGeneratedEventType is the stable wire name emitted when an invoice is
// generated from a completed encounter.
const InvoiceGeneratedEventType = "invoice.generated"

// InvoiceGeneratedEvent is emitted when a GenerateInvoiceCmd succeeds. It
// records the encounter the invoice was generated from, the billable line items
// captured on it, and the insurance policy its claim is adjudicated against.
type InvoiceGeneratedEvent struct {
	// InvoiceID is the identity of the InvoiceAggregate that produced the event.
	InvoiceID string
	// EncounterID is the completed encounter the invoice was generated from.
	EncounterID string
	// LineItems are the billable charges captured on the invoice.
	LineItems []InvoiceLineItem
	// PolicyID is the insurance policy the invoice's claim is adjudicated
	// against.
	PolicyID string
}

// Type identifies the event kind.
func (e InvoiceGeneratedEvent) Type() string { return InvoiceGeneratedEventType }

// AggregateID ties the event back to the invoice that produced it.
func (e InvoiceGeneratedEvent) AggregateID() string { return e.InvoiceID }

// Compile-time assertion that InvoiceGeneratedEvent satisfies the DomainEvent
// contract.
var _ shared.DomainEvent = InvoiceGeneratedEvent{}

// InvoiceAdjustedEventType is the stable wire name emitted when an invoice's
// verified insurance coverage and copay are applied.
const InvoiceAdjustedEventType = "invoice.adjusted"

// InvoiceAdjustedEvent is emitted when an ApplyInsuranceAdjustmentCmd succeeds.
// It records the verified insurance adjustment and copay applied to the invoice,
// both in whole cents.
type InvoiceAdjustedEvent struct {
	// InvoiceID is the identity of the InvoiceAggregate that produced the event.
	InvoiceID string
	// CoverageCents is the verified insurance adjustment applied, in whole cents.
	CoverageCents int64
	// CopayCents is the patient copay applied, in whole cents.
	CopayCents int64
}

// Type identifies the event kind.
func (e InvoiceAdjustedEvent) Type() string { return InvoiceAdjustedEventType }

// AggregateID ties the event back to the invoice that produced it.
func (e InvoiceAdjustedEvent) AggregateID() string { return e.InvoiceID }

// Compile-time assertion that InvoiceAdjustedEvent satisfies the DomainEvent
// contract.
var _ shared.DomainEvent = InvoiceAdjustedEvent{}

// InvoiceVoidedEventType is the stable wire name emitted when an issued invoice
// is voided.
const InvoiceVoidedEventType = "invoice.voided"

// InvoiceVoidedEvent is emitted when a VoidInvoiceCmd succeeds. It records the
// reason the invoice was voided, carried through for the audit trail.
type InvoiceVoidedEvent struct {
	// InvoiceID is the identity of the InvoiceAggregate that produced the event.
	InvoiceID string
	// Reason records why the invoice was voided.
	Reason string
}

// Type identifies the event kind.
func (e InvoiceVoidedEvent) Type() string { return InvoiceVoidedEventType }

// AggregateID ties the event back to the invoice that produced it.
func (e InvoiceVoidedEvent) AggregateID() string { return e.InvoiceID }

// Compile-time assertion that InvoiceVoidedEvent satisfies the DomainEvent
// contract.
var _ shared.DomainEvent = InvoiceVoidedEvent{}

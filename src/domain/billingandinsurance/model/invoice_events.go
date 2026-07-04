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

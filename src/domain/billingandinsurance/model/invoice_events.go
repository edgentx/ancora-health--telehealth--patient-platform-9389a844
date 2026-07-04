package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// InvoiceVoidedEventType is the stable wire name emitted when an issued invoice
// is voided.
const InvoiceVoidedEventType = "invoice.voided"

// InvoiceVoidedEvent is emitted when a VoidInvoiceCmd succeeds. It records the
// identity of the voided invoice along with the reason it was withdrawn, which
// is retained for the billing audit trail. Once this event is applied the
// invoice is closed and can receive no further payments.
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

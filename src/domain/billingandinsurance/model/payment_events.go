package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// PaymentInitiatedEventType is the stable wire name emitted when a tokenized
// payment is initiated against an invoice.
const PaymentInitiatedEventType = "payment.initiated"

// PaymentInitiatedEvent is emitted when an InitiatePaymentCmd succeeds. It
// records the invoice the payment is captured against, the gateway token used
// to charge it, and the amount captured in whole cents.
type PaymentInitiatedEvent struct {
	// PaymentID is the identity of the PaymentAggregate that produced the event.
	PaymentID string
	// InvoiceID is the invoice the payment is captured against.
	InvoiceID string
	// PaymentToken is the gateway token the payment was charged with. Raw card
	// data is never recorded — only this tokenized reference.
	PaymentToken string
	// AmountCents is the amount captured, in whole cents.
	AmountCents int64
}

// Type identifies the event kind.
func (e PaymentInitiatedEvent) Type() string { return PaymentInitiatedEventType }

// AggregateID ties the event back to the payment that produced it.
func (e PaymentInitiatedEvent) AggregateID() string { return e.PaymentID }

// Compile-time assertion that PaymentInitiatedEvent satisfies the DomainEvent
// contract.
var _ shared.DomainEvent = PaymentInitiatedEvent{}

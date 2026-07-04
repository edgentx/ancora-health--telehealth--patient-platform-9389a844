// Package model holds the aggregates for the billing-and-insurance bounded
// context. PaymentAggregate models a tokenized patient/payer payment captured
// against an invoice.
package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// PaymentStatus is the lifecycle state of a payment. The zero value is a
// payment that has not yet been initiated, which is what InitiatePaymentCmd
// acts on.
type PaymentStatus string

const (
	// PaymentStatusNew is a payment that has not yet been initiated. It is the
	// zero value, so a freshly constructed aggregate is new.
	PaymentStatusNew PaymentStatus = ""
	// PaymentStatusInitiated is a payment whose tokenized charge has been started
	// against an invoice and is awaiting gateway confirmation.
	PaymentStatusInitiated PaymentStatus = "initiated"
)

// PaymentAggregate is the aggregate root for a billing-and-insurance payment.
// It embeds shared.AggregateRoot for version tracking and an uncommitted-event
// buffer, and carries its own identity in ID.
//
// Beyond identity it tracks the state that command invariants read: its
// lifecycle status, the invoice it is captured against, the gateway token it is
// charged with, the amount in whole cents, and the flags describing whether raw
// card data would be persisted, whether the invoice has an outstanding balance,
// and whether a status advance is backed by an HMAC-verified gateway webhook.
//
// The invariant flags follow the repository convention that a freshly
// constructed aggregate is valid: their zero value is the compliant state, and a
// non-zero value marks a violation the guards reject.
type PaymentAggregate struct {
	shared.AggregateRoot
	ID string

	// Status is the payment's lifecycle state.
	Status PaymentStatus

	// InvoiceID is the invoice the payment is captured against. It is empty until
	// the payment is initiated.
	InvoiceID string

	// PaymentToken is the gateway token the payment is charged with. It is empty
	// until the payment is initiated.
	PaymentToken string

	// AmountCents is the amount captured, in whole cents. It is zero until the
	// payment is initiated.
	AmountCents int64

	// RawCardDataPresent reports that raw card data would be persisted instead of
	// a gateway token. Invariant: raw card data is never persisted — only gateway
	// tokens are stored.
	RawCardDataPresent bool

	// NoOutstandingBalance reports that the target invoice has no outstanding
	// balance to capture against. Invariant: a payment can only be captured
	// against an outstanding invoice balance.
	NoOutstandingBalance bool

	// WebhookNotVerified reports that a status advance is not backed by an
	// HMAC-verified webhook from the gateway. Invariant: payment status may only
	// advance on an HMAC-verified webhook from the gateway.
	WebhookNotVerified bool
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. Unrecognized command types return shared.ErrUnknownCommand.
func (a *PaymentAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch c := cmd.(type) {
	case InitiatePaymentCmd:
		return a.initiatePayment(c)
	default:
		return nil, shared.ErrUnknownCommand
	}
}

// initiatePayment handles InitiatePaymentCmd: it validates the command input,
// enforces the payment invariants, then emits a PaymentInitiatedEvent and
// buffers it on the aggregate.
//
// The guards enforce, in order:
//
//   - Completeness: the invoice, payment token and a positive amount must all be
//     present.
//   - Tokenization: raw card data is never persisted — only gateway tokens are
//     stored.
//   - Outstanding balance: a payment can only be captured against an outstanding
//     invoice balance.
//   - Verified webhook: payment status may only advance on an HMAC-verified
//     webhook from the gateway.
func (a *PaymentAggregate) initiatePayment(cmd InitiatePaymentCmd) ([]shared.DomainEvent, error) {
	if cmd.InvoiceId == "" {
		return nil, ErrMissingInvoice
	}
	if cmd.PaymentToken == "" {
		return nil, ErrMissingPaymentToken
	}
	if cmd.AmountCents <= 0 {
		return nil, ErrNonPositiveAmount
	}

	// Invariant: raw card data is never persisted — only gateway tokens are
	// stored.
	if a.RawCardDataPresent {
		return nil, ErrRawCardData
	}

	// Invariant: a payment can only be captured against an outstanding invoice
	// balance.
	if a.NoOutstandingBalance {
		return nil, ErrNoOutstandingBalance
	}

	// Invariant: payment status may only advance on an HMAC-verified webhook from
	// the gateway.
	if a.WebhookNotVerified {
		return nil, ErrWebhookNotVerified
	}

	evt := PaymentInitiatedEvent{
		PaymentID:    a.ID,
		InvoiceID:    cmd.InvoiceId,
		PaymentToken: cmd.PaymentToken,
		AmountCents:  cmd.AmountCents,
	}

	a.apply(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// apply mutates aggregate state from a domain event. It is the single place
// state changes, so the same function serves both command handling and future
// event replay when rehydrating the aggregate from the store.
func (a *PaymentAggregate) apply(evt PaymentInitiatedEvent) {
	a.Status = PaymentStatusInitiated
	a.InvoiceID = evt.InvoiceID
	a.PaymentToken = evt.PaymentToken
	a.AmountCents = evt.AmountCents
}

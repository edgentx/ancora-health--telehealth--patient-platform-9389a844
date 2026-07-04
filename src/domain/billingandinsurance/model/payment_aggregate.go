// Package model holds the aggregates for the billing-and-insurance bounded
// context. PaymentAggregate tracks a patient/payer payment through its
// reconciliation lifecycle.
package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// PaymentStatus is the lifecycle state of a payment. The zero value is a pending
// payment that has not yet been reconciled against a gateway settlement.
type PaymentStatus string

const (
	// PaymentStatusPending is a payment that has not yet been reconciled. It is
	// the zero value, so a freshly constructed aggregate is pending.
	PaymentStatusPending PaymentStatus = ""
	// PaymentStatusReconciled is a payment whose status has advanced from a
	// verified gateway webhook settlement.
	PaymentStatusReconciled PaymentStatus = "reconciled"
)

// PaymentAggregate is the aggregate root for a billing-and-insurance payment.
// It embeds shared.AggregateRoot for version tracking and an uncommitted-event
// buffer, and carries its own identity in ID.
//
// Beyond identity it tracks the state that command invariants read: its
// lifecycle status and the flags describing whether the reconciling webhook has
// been HMAC-verified, whether the target invoice has an outstanding balance, and
// whether raw card data is present on the aggregate.
//
// The invariant flags follow the repository convention that a freshly
// constructed aggregate is valid: their zero value is the compliant state, and a
// non-zero value marks a violation the guards reject.
type PaymentAggregate struct {
	shared.AggregateRoot
	ID string

	// Status is the payment's lifecycle state.
	Status PaymentStatus

	// WebhookNotVerified reports that the reconciling webhook's HMAC signature has
	// not been verified against the gateway secret. Invariant: payment status may
	// only advance on an HMAC-verified webhook from the gateway.
	WebhookNotVerified bool

	// NoOutstandingBalance reports that the invoice the payment captures against
	// has no outstanding balance. Invariant: a payment can only be captured
	// against an outstanding invoice balance.
	NoOutstandingBalance bool

	// RawCardDataPresent reports that raw card data is present on the payment.
	// Invariant: raw card data is never persisted — only gateway tokens are
	// stored.
	RawCardDataPresent bool
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. Unrecognized command types return shared.ErrUnknownCommand.
func (a *PaymentAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch c := cmd.(type) {
	case ReconcilePaymentCmd:
		return a.reconcilePayment(c)
	default:
		return nil, shared.ErrUnknownCommand
	}
}

// reconcilePayment handles ReconcilePaymentCmd: it validates the command input,
// enforces the payment invariants, then emits a PaymentReconciledEvent and
// buffers it on the aggregate.
//
// The guards enforce, in order:
//
//   - Completeness: the payment, webhook payload and signature must all be
//     present.
//   - HMAC-verified webhook: the payment's status may only advance on a webhook
//     whose HMAC signature has been verified against the gateway secret.
//   - Outstanding balance: a payment can only be captured against an outstanding
//     invoice balance.
//   - Card-data safety: raw card data is never persisted — only gateway tokens
//     are stored.
func (a *PaymentAggregate) reconcilePayment(cmd ReconcilePaymentCmd) ([]shared.DomainEvent, error) {
	if cmd.PaymentId == "" {
		return nil, ErrMissingPayment
	}
	if cmd.WebhookPayload == "" {
		return nil, ErrMissingWebhookPayload
	}
	if cmd.Signature == "" {
		return nil, ErrMissingSignature
	}

	// Invariant: payment status may only advance on an HMAC-verified webhook from
	// the gateway.
	if a.WebhookNotVerified {
		return nil, ErrWebhookNotVerified
	}

	// Invariant: a payment can only be captured against an outstanding invoice
	// balance.
	if a.NoOutstandingBalance {
		return nil, ErrNoOutstandingBalance
	}

	// Invariant: raw card data is never persisted — only gateway tokens are
	// stored.
	if a.RawCardDataPresent {
		return nil, ErrRawCardDataPersisted
	}

	evt := PaymentReconciledEvent{
		PaymentID:      a.ID,
		WebhookPayload: cmd.WebhookPayload,
		Signature:      cmd.Signature,
	}

	a.apply(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// apply mutates aggregate state from a domain event. It is the single place
// state changes, so the same function serves both command handling and future
// event replay when rehydrating the aggregate from the store.
func (a *PaymentAggregate) apply(evt PaymentReconciledEvent) {
	a.Status = PaymentStatusReconciled
}

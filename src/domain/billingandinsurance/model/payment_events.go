package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// PaymentReconciledEventType is the stable wire name emitted when a payment is
// reconciled from a verified gateway webhook.
const PaymentReconciledEventType = "payment.reconciled"

// PaymentReconciledEvent is emitted when a ReconcilePaymentCmd succeeds. It
// records the webhook payload the settlement was applied from and the HMAC
// signature that authenticated it, so the reconciliation is auditable.
type PaymentReconciledEvent struct {
	// PaymentID is the identity of the PaymentAggregate that produced the event.
	PaymentID string
	// WebhookPayload is the verified gateway callback body the reconciliation was
	// applied from.
	WebhookPayload string
	// Signature is the HMAC that authenticated the webhook payload.
	Signature string
}

// Type identifies the event kind.
func (e PaymentReconciledEvent) Type() string { return PaymentReconciledEventType }

// AggregateID ties the event back to the payment that produced it.
func (e PaymentReconciledEvent) AggregateID() string { return e.PaymentID }

// Compile-time assertion that PaymentReconciledEvent satisfies the DomainEvent
// contract.
var _ shared.DomainEvent = PaymentReconciledEvent{}

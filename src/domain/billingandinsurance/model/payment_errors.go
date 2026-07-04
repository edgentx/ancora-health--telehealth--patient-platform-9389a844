package model

import "errors"

var (
	// ErrMissingPayment is returned when ReconcilePaymentCmd omits the payment id.
	ErrMissingPayment = errors.New("payment: payment id is required")

	// ErrMissingWebhookPayload is returned when ReconcilePaymentCmd omits the
	// webhook payload.
	ErrMissingWebhookPayload = errors.New("payment: webhook payload is required")

	// ErrMissingSignature is returned when ReconcilePaymentCmd omits the webhook
	// signature.
	ErrMissingSignature = errors.New("payment: webhook signature is required")

	// ErrWebhookNotVerified is returned when the payment would advance on a
	// webhook whose HMAC signature has not been verified. Invariant: payment
	// status may only advance on an HMAC-verified webhook from the gateway.
	ErrWebhookNotVerified = errors.New("payment: payment status may only advance on an HMAC-verified webhook from the gateway")

	// ErrNoOutstandingBalance is returned when a payment would be captured against
	// an invoice with no outstanding balance. Invariant: a payment can only be
	// captured against an outstanding invoice balance.
	ErrNoOutstandingBalance = errors.New("payment: a payment can only be captured against an outstanding invoice balance")

	// ErrRawCardDataPersisted is returned when the payment carries raw card data.
	// Invariant: raw card data is never persisted — only gateway tokens are
	// stored.
	ErrRawCardDataPersisted = errors.New("payment: raw card data is never persisted — only gateway tokens are stored")
)

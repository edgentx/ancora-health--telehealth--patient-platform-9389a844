package model

import "errors"

var (
	// ErrMissingInvoice is returned when InitiatePaymentCmd omits the invoice id.
	ErrMissingInvoice = errors.New("payment: invoice id is required")

	// ErrMissingPaymentToken is returned when InitiatePaymentCmd omits the gateway
	// payment token.
	ErrMissingPaymentToken = errors.New("payment: payment token is required")

	// ErrNonPositiveAmount is returned when InitiatePaymentCmd carries an amount
	// that is not a positive number of cents.
	ErrNonPositiveAmount = errors.New("payment: amount must be a positive number of cents")

	// ErrRawCardData is returned when a payment would persist raw card data
	// instead of a gateway token. Invariant: raw card data is never persisted —
	// only gateway tokens are stored.
	ErrRawCardData = errors.New("payment: raw card data is never persisted — only gateway tokens are stored")

	// ErrNoOutstandingBalance is returned when a payment is captured against an
	// invoice with no outstanding balance. Invariant: a payment can only be
	// captured against an outstanding invoice balance.
	ErrNoOutstandingBalance = errors.New("payment: a payment can only be captured against an outstanding invoice balance")

	// ErrWebhookNotVerified is returned when a payment's status would advance
	// without an HMAC-verified webhook from the gateway. Invariant: payment status
	// may only advance on an HMAC-verified webhook from the gateway.
	ErrWebhookNotVerified = errors.New("payment: payment status may only advance on an HMAC-verified webhook from the gateway")
)

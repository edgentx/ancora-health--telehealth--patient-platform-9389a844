package model

// InitiatePaymentCmd requests that a tokenized Payment be started against an
// invoice. It carries the invoice being paid, the gateway token standing in for
// the patient's card, and the amount to capture in whole cents.
//
// Initiating a payment is the act that starts a charge against an outstanding
// invoice balance. It never carries raw card data — only a gateway token — it
// may only be captured against an outstanding invoice balance, and any later
// status advance must ride an HMAC-verified webhook from the gateway. InvoiceId
// identifies the invoice being paid, PaymentToken the gateway token, and
// AmountCents the charge. All three are mandatory.
type InitiatePaymentCmd struct {
	// InvoiceId identifies the invoice the payment is captured against.
	InvoiceId string
	// PaymentToken is the gateway token standing in for the patient's card. Raw
	// card data is never accepted here — only a tokenized reference.
	PaymentToken string
	// AmountCents is the amount to capture, in whole cents. It must be positive.
	AmountCents int64
}

// ReconcilePaymentCmd applies a verified gateway webhook result to a Payment,
// advancing its lifecycle to reconciled. It carries the payment being
// reconciled, the raw webhook payload delivered by the gateway, and the HMAC
// signature the gateway computed over that payload.
//
// Reconciliation is a status advance, so it is bound by the same invariants as
// initiation: it never persists raw card data — only gateway tokens — it may
// only be captured against an outstanding invoice balance, and, crucially, the
// status may only advance on an HMAC-verified webhook from the gateway.
// PaymentId identifies the payment, WebhookPayload is the gateway's message,
// and Signature is its HMAC. All three are mandatory.
type ReconcilePaymentCmd struct {
	// PaymentId identifies the payment being reconciled.
	PaymentId string
	// WebhookPayload is the raw message the gateway delivered describing the
	// charge outcome. It is the bytes the HMAC signature is computed over.
	WebhookPayload string
	// Signature is the HMAC the gateway computed over WebhookPayload. A payment's
	// status may only advance on a webhook whose signature verifies.
	Signature string
}

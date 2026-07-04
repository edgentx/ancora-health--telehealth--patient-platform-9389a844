package model

// ReconcilePaymentCmd requests that a Payment be reconciled by applying a
// verified gateway webhook result, advancing the payment's status to reflect
// the settlement the gateway reported.
//
// Reconciliation is the act that lets an external payment gateway drive the
// payment's lifecycle: the payment's status may only advance on a webhook whose
// HMAC signature has been verified against the gateway's shared secret, the
// capture it settles must be against an outstanding invoice balance, and no raw
// card data may ever be persisted — only gateway tokens are stored. PaymentId
// identifies the payment being reconciled, WebhookPayload is the raw gateway
// callback body, and Signature is the HMAC that authenticates it. All three are
// mandatory.
type ReconcilePaymentCmd struct {
	// PaymentId identifies the payment being reconciled.
	PaymentId string
	// WebhookPayload is the raw gateway callback body describing the settlement
	// result being applied to the payment.
	WebhookPayload string
	// Signature is the HMAC that authenticates the webhook payload against the
	// gateway's shared secret.
	Signature string
}

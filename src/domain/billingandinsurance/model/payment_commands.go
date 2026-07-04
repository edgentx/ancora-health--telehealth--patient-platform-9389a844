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

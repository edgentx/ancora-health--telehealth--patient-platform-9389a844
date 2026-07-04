package model

// VoidInvoiceCmd requests that an already-issued Invoice be voided, capturing
// the reason the invoice is being withdrawn.
//
// Voiding is the act that permanently withdraws an issued invoice from the
// billing lifecycle: a voided invoice is closed to further activity, so it may
// no longer receive payments. The invoice must have been well-formed when it was
// issued — generated from a completed encounter, with patient responsibility
// reconciled against charges, insurance adjustment and copay, and never marked
// paid beyond its outstanding balance — before it can be cleanly voided.
// InvoiceId identifies the invoice and Reason records why it is being voided;
// both are mandatory.
type VoidInvoiceCmd struct {
	// InvoiceId identifies the invoice to void.
	InvoiceId string
	// Reason records why the invoice is being voided; it is retained on the
	// emitted event for the billing audit trail.
	Reason string
}

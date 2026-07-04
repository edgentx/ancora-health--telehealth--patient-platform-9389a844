package model

// InvoiceLineItem is a single billable line captured on an invoice: a
// human-readable description and the charge it contributes, in whole cents.
// Together the line items make up the invoice's total charges.
type InvoiceLineItem struct {
	// Description is the human-readable name of the billed item or service.
	Description string
	// AmountCents is the charge this line contributes to the invoice, in whole
	// cents.
	AmountCents int64
}

// GenerateInvoiceCmd requests that an Invoice be generated from a completed
// encounter, capturing the charges to bill and the insurance policy the claim
// is adjudicated against.
//
// Generating an invoice is the act that turns a finished visit into a billable
// artifact: it may only be generated from a completed encounter, the patient
// responsibility it records must equal charges minus the verified insurance
// adjustment and copay, and once generated it may not be marked paid beyond its
// outstanding balance nor receive payments after being voided. EncounterId
// identifies the encounter being billed, LineItems the charges, and PolicyId
// the insurance policy. All three are mandatory.
type GenerateInvoiceCmd struct {
	// EncounterId identifies the completed encounter the invoice is generated
	// from.
	EncounterId string
	// LineItems are the billable charges captured on the invoice. At least one is
	// required.
	LineItems []InvoiceLineItem
	// PolicyId identifies the insurance policy the invoice's claim is adjudicated
	// against.
	PolicyId string
}

// VoidInvoiceCmd requests that an issued Invoice be voided, cancelling it so it
// can no longer be paid.
//
// Voiding is the act that retires a billable invoice: it still enforces the
// invoice invariants — the source encounter must be completed, patient
// responsibility must reconcile against charges, the invoice must not have been
// marked paid beyond its outstanding balance, and an already-voided invoice
// cannot be acted on further. InvoiceId identifies the invoice being voided and
// Reason records why. Both are mandatory.
type VoidInvoiceCmd struct {
	// InvoiceId identifies the invoice being voided.
	InvoiceId string
	// Reason records why the invoice is being voided.
	Reason string
}

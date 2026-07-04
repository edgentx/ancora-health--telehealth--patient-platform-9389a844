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

// EligibilityResult is the verified outcome of an insurance eligibility check:
// the coverage the payer will adjudicate against the invoice's charges and the
// copay the patient owes, both in whole cents. It is the input that
// ApplyInsuranceAdjustmentCmd applies to reconcile patient responsibility.
//
// Verified reports whether the result came back from a completed eligibility
// verification; only a verified result may be applied. The amounts are the
// verified insurance adjustment (CoverageCents) and the patient copay
// (CopayCents), and neither may be negative.
type EligibilityResult struct {
	// Verified reports that the eligibility check completed and its coverage and
	// copay are trustworthy. An unverified result may not be applied.
	Verified bool
	// CoverageCents is the verified insurance adjustment the payer covers, in
	// whole cents. It may not be negative.
	CoverageCents int64
	// CopayCents is the patient copay owed, in whole cents. It may not be
	// negative.
	CopayCents int64
}

// ApplyInsuranceAdjustmentCmd requests that a generated Invoice have its verified
// insurance coverage and copay applied, reconciling the patient responsibility
// against the invoice's charges.
//
// Applying the adjustment is the act that turns a raw claim into a
// patient-owed balance: the invoice must have been generated from a completed
// encounter, the resulting patient responsibility must equal charges minus the
// verified insurance adjustment and copay, it may not be marked paid beyond its
// outstanding balance, and a voided invoice may not receive further payments.
// InvoiceId identifies the invoice being adjusted and Eligibility carries the
// verified coverage and copay. Both are mandatory and the eligibility result
// must be verified.
type ApplyInsuranceAdjustmentCmd struct {
	// InvoiceId identifies the generated invoice the adjustment is applied to.
	InvoiceId string
	// Eligibility is the verified coverage and copay to apply.
	Eligibility EligibilityResult
}

// VoidInvoiceCmd requests that an issued Invoice be voided, cancelling it so it
// can no longer be adjusted or paid against.
//
// Voiding is the act that retires an invoice: it must have been generated from
// a completed encounter, its patient responsibility must reconcile against
// charges, it may not have been marked paid beyond its outstanding balance, and
// an already-voided invoice cannot be voided again to receive further payments.
// InvoiceId identifies the invoice being voided and Reason records why, for the
// audit trail. Both are mandatory.
type VoidInvoiceCmd struct {
	// InvoiceId identifies the invoice being voided.
	InvoiceId string
	// Reason records why the invoice is being voided, for the audit trail.
	Reason string
}

package model

// EligibilityResult carries the verified outcome of an insurance eligibility
// check. It is the input ApplyInsuranceAdjustmentCmd applies to an invoice: the
// portion of charges the insurer has verified it will cover, and the copay the
// patient owes at the point of care.
//
// Both amounts are expressed in minor currency units (cents) and must be
// non-negative; a payer cannot verify a negative coverage or a negative copay.
type EligibilityResult struct {
	// VerifiedAdjustment is the amount, in cents, the insurer has verified it will
	// cover — the insurance adjustment applied against the invoice charges.
	VerifiedAdjustment int64
	// Copay is the amount, in cents, the patient owes at the point of care under
	// the verified benefits.
	Copay int64
}

// ApplyInsuranceAdjustmentCmd requests that an Invoice have a verified insurance
// eligibility result applied to it: the insurer's verified coverage becomes the
// invoice's insurance adjustment and the verified copay is recorded, leaving the
// remainder as the patient's responsibility.
//
// InvoiceId identifies the target invoice and Eligibility supplies the verified
// coverage and copay. Both are mandatory, and the eligibility amounts must be
// non-negative.
type ApplyInsuranceAdjustmentCmd struct {
	// InvoiceId identifies the invoice the adjustment is applied to.
	InvoiceId string
	// Eligibility is the verified eligibility result — the coverage and copay to
	// apply to the invoice.
	Eligibility EligibilityResult
}

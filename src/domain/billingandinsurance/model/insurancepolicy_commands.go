package model

// EffectiveDates is the coverage window of an insurance policy: the date the
// policy takes effect and the date it lapses. Both bounds are required for the
// window to be valid, and an empty End marks an open-ended term.
type EffectiveDates struct {
	// Start is the date the policy's coverage begins (RFC 3339 date).
	Start string
	// End is the date the policy's coverage ends (RFC 3339 date).
	End string
}

// RegisterInsurancePolicyCmd requests that a patient's insurance policy be
// registered, capturing the patient it covers, the payer that underwrites it,
// and the coverage window it is effective for.
//
// Registering a policy is the act that admits a patient's coverage into
// billing: the policy must carry a verified eligibility result before it can
// adjust an invoice, at most one active primary policy may exist per patient at
// a time, and an expired policy cannot be used for new eligibility checks.
// PatientId identifies the covered patient, PayerIdentifier the underwriting
// payer, and EffectiveDates the coverage window. All three are mandatory.
type RegisterInsurancePolicyCmd struct {
	// PatientId identifies the patient the policy covers.
	PatientId string
	// PayerIdentifier identifies the payer underwriting the policy.
	PayerIdentifier string
	// EffectiveDates is the coverage window the policy is effective for.
	EffectiveDates EffectiveDates
}

// VerifyEligibilityCmd requests payer eligibility verification for a registered
// policy as of a given service date. It carries the policy to check and the
// date the coverage is being verified for.
//
// Verifying eligibility is the act that admits a policy's coverage into billing:
// the policy must not already carry an unverified eligibility result blocking an
// invoice adjustment, at most one active primary policy may exist per patient at
// a time, and an expired policy cannot be used for new eligibility checks.
// PolicyId identifies the policy to verify and ServiceDate the date coverage is
// verified for. Both are mandatory.
type VerifyEligibilityCmd struct {
	// PolicyId identifies the policy whose eligibility is being verified.
	PolicyId string
	// ServiceDate is the date coverage is being verified for (RFC 3339 date).
	ServiceDate string
}

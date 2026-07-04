package model

import "errors"

var (
	// ErrMissingPatient is returned when RegisterInsurancePolicyCmd omits the
	// patient id.
	ErrMissingPatient = errors.New("insurancepolicy: patient id is required")

	// ErrMissingPayerIdentifier is returned when RegisterInsurancePolicyCmd omits
	// the payer identifier.
	ErrMissingPayerIdentifier = errors.New("insurancepolicy: payer identifier is required")

	// ErrMissingEffectiveDates is returned when RegisterInsurancePolicyCmd omits
	// the coverage window's start or end date.
	ErrMissingEffectiveDates = errors.New("insurancepolicy: effective dates are required")

	// ErrEligibilityNotVerified is returned when a policy without a verified
	// eligibility result would be registered. Invariant: a policy must have a
	// verified eligibility result before it can adjust an invoice.
	ErrEligibilityNotVerified = errors.New("insurancepolicy: a policy must have a verified eligibility result before it can adjust an invoice")

	// ErrActivePrimaryPolicyExists is returned when registering the policy would
	// leave the patient with more than one active primary policy. Invariant: only
	// one active primary policy may exist per patient at a time.
	ErrActivePrimaryPolicyExists = errors.New("insurancepolicy: only one active primary policy may exist per patient at a time")

	// ErrPolicyExpired is returned when an expired policy would be used for a new
	// eligibility check. Invariant: an expired policy cannot be used for new
	// eligibility checks.
	ErrPolicyExpired = errors.New("insurancepolicy: an expired policy cannot be used for new eligibility checks")
)

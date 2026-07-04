// Package model holds the aggregates for the billing-and-insurance bounded
// context. InsurancePolicyAggregate models a patient's insurance policy;
// RegisterInsurancePolicyCmd admits the policy into billing.
package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// PolicyStatus is the lifecycle state of an insurance policy. The zero value is
// a policy that has not yet been registered, which is what
// RegisterInsurancePolicyCmd acts on.
type PolicyStatus string

const (
	// PolicyStatusNew is a policy that has not yet been registered. It is the zero
	// value, so a freshly constructed aggregate is new.
	PolicyStatusNew PolicyStatus = ""
	// PolicyStatusRegistered is a policy that has been admitted into billing for a
	// patient.
	PolicyStatusRegistered PolicyStatus = "registered"
)

// InsurancePolicyAggregate is the aggregate root for a billing-and-insurance
// policy. It embeds shared.AggregateRoot for version tracking and an
// uncommitted-event buffer, and carries its own identity in ID.
//
// Beyond identity it tracks the state that command invariants read: its
// lifecycle status, the patient it covers, the payer that underwrites it, the
// coverage window, and the flags describing whether its eligibility result is
// verified, whether the patient already has an active primary policy, and
// whether the policy has expired.
//
// The invariant flags follow the repository convention that a freshly
// constructed aggregate is valid: their zero value is the compliant state, and a
// non-zero value marks a violation the guards reject.
type InsurancePolicyAggregate struct {
	shared.AggregateRoot
	ID string

	// Status is the policy's lifecycle state.
	Status PolicyStatus

	// PatientID is the patient the policy covers. It is empty until the policy is
	// registered.
	PatientID string

	// PayerIdentifier is the payer underwriting the policy. It is empty until the
	// policy is registered.
	PayerIdentifier string

	// EffectiveDates is the coverage window the policy is effective for. It is the
	// zero value until the policy is registered.
	EffectiveDates EffectiveDates

	// EligibilityNotVerified reports that the policy has no verified eligibility
	// result. Invariant: a policy must have a verified eligibility result before it
	// can adjust an invoice.
	EligibilityNotVerified bool

	// ActivePrimaryPolicyExists reports that the patient already holds an active
	// primary policy. Invariant: only one active primary policy may exist per
	// patient at a time.
	ActivePrimaryPolicyExists bool

	// PolicyExpired reports that the policy has expired. Invariant: an expired
	// policy cannot be used for new eligibility checks.
	PolicyExpired bool
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. Unrecognized command types return shared.ErrUnknownCommand.
func (a *InsurancePolicyAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch c := cmd.(type) {
	case RegisterInsurancePolicyCmd:
		return a.registerInsurancePolicy(c)
	default:
		return nil, shared.ErrUnknownCommand
	}
}

// registerInsurancePolicy handles RegisterInsurancePolicyCmd: it validates the
// command input, enforces the policy invariants, then emits a
// PolicyRegisteredEvent and buffers it on the aggregate.
//
// The guards enforce, in order:
//
//   - Completeness: the patient, payer identifier and both coverage-window bounds
//     must all be present.
//   - Verified eligibility: a policy must have a verified eligibility result
//     before it can adjust an invoice.
//   - Single active primary: only one active primary policy may exist per patient
//     at a time.
//   - Not expired: an expired policy cannot be used for new eligibility checks.
func (a *InsurancePolicyAggregate) registerInsurancePolicy(cmd RegisterInsurancePolicyCmd) ([]shared.DomainEvent, error) {
	if cmd.PatientId == "" {
		return nil, ErrMissingPatient
	}
	if cmd.PayerIdentifier == "" {
		return nil, ErrMissingPayerIdentifier
	}
	if cmd.EffectiveDates.Start == "" || cmd.EffectiveDates.End == "" {
		return nil, ErrMissingEffectiveDates
	}

	// Invariant: a policy must have a verified eligibility result before it can
	// adjust an invoice.
	if a.EligibilityNotVerified {
		return nil, ErrEligibilityNotVerified
	}

	// Invariant: only one active primary policy may exist per patient at a time.
	if a.ActivePrimaryPolicyExists {
		return nil, ErrActivePrimaryPolicyExists
	}

	// Invariant: an expired policy cannot be used for new eligibility checks.
	if a.PolicyExpired {
		return nil, ErrPolicyExpired
	}

	evt := PolicyRegisteredEvent{
		PolicyID:        a.ID,
		PatientID:       cmd.PatientId,
		PayerIdentifier: cmd.PayerIdentifier,
		EffectiveDates:  cmd.EffectiveDates,
	}

	a.apply(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// apply mutates aggregate state from a domain event. It is the single place
// state changes, so the same function serves both command handling and future
// event replay when rehydrating the aggregate from the store.
func (a *InsurancePolicyAggregate) apply(evt PolicyRegisteredEvent) {
	a.Status = PolicyStatusRegistered
	a.PatientID = evt.PatientID
	a.PayerIdentifier = evt.PayerIdentifier
	a.EffectiveDates = evt.EffectiveDates
}

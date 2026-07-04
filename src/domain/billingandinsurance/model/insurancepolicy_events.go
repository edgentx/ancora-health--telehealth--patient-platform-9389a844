package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// PolicyRegisteredEventType is the stable wire name emitted when a patient's
// insurance policy is registered.
const PolicyRegisteredEventType = "policy.registered"

// PolicyRegisteredEvent is emitted when a RegisterInsurancePolicyCmd succeeds.
// It records the patient the policy covers, the payer that underwrites it, and
// the coverage window it is effective for.
type PolicyRegisteredEvent struct {
	// PolicyID is the identity of the InsurancePolicyAggregate that produced the
	// event.
	PolicyID string
	// PatientID is the patient the policy covers.
	PatientID string
	// PayerIdentifier is the payer underwriting the policy.
	PayerIdentifier string
	// EffectiveDates is the coverage window the policy is effective for.
	EffectiveDates EffectiveDates
}

// Type identifies the event kind.
func (e PolicyRegisteredEvent) Type() string { return PolicyRegisteredEventType }

// AggregateID ties the event back to the policy that produced it.
func (e PolicyRegisteredEvent) AggregateID() string { return e.PolicyID }

// Compile-time assertion that PolicyRegisteredEvent satisfies the DomainEvent
// contract.
var _ shared.DomainEvent = PolicyRegisteredEvent{}

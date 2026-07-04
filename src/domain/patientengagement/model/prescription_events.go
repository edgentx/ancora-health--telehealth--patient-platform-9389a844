package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// PrescriptionSafetyCheckedEventType is the stable wire name emitted when a
// prescription clears allergy and drug-interaction verification.
const PrescriptionSafetyCheckedEventType = "prescription.safety.checked"

// PrescriptionSafetyCheckedEvent is emitted when a RunSafetyCheckCmd succeeds. It
// records the prescription that was checked, along with the issuing provider and
// the patient it is scoped to.
type PrescriptionSafetyCheckedEvent struct {
	// PrescriptionID is the identity of the PrescriptionAggregate that produced the
	// event.
	PrescriptionID string
	// ProviderID is the issuing provider; it is authenticated and holds an active
	// care relationship to the patient.
	ProviderID string
	// PatientID is the patient the prescription is written for.
	PatientID string
}

// Type identifies the event kind.
func (e PrescriptionSafetyCheckedEvent) Type() string { return PrescriptionSafetyCheckedEventType }

// AggregateID ties the event back to the prescription that produced it.
func (e PrescriptionSafetyCheckedEvent) AggregateID() string { return e.PrescriptionID }

// Compile-time assertion that PrescriptionSafetyCheckedEvent satisfies the
// DomainEvent contract.
var _ shared.DomainEvent = PrescriptionSafetyCheckedEvent{}

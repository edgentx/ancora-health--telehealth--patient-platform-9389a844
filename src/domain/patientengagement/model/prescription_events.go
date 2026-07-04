package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// PrescriptionComposedEventType is the stable wire name emitted when a
// prescription is drafted for a patient.
const PrescriptionComposedEventType = "prescription.composed"

// PrescriptionComposedEvent is emitted when a ComposePrescriptionCmd succeeds. It
// records the patient and issuing provider the prescription is scoped to, along
// with the drafted medication and its dosage.
type PrescriptionComposedEvent struct {
	// PrescriptionID is the identity of the PrescriptionAggregate that produced
	// the event.
	PrescriptionID string
	// PatientID is the patient the prescription is drafted for.
	PatientID string
	// ProviderID is the provider who issued the prescription; it is the
	// participant the prescription is scoped to.
	ProviderID string
	// Medication is the medication that was prescribed.
	Medication string
	// Dosage is the drafted dosage instruction for the medication.
	Dosage string
}

// Type identifies the event kind.
func (e PrescriptionComposedEvent) Type() string { return PrescriptionComposedEventType }

// AggregateID ties the event back to the prescription that produced it.
func (e PrescriptionComposedEvent) AggregateID() string { return e.PrescriptionID }

// Compile-time assertion that PrescriptionComposedEvent satisfies the DomainEvent
// contract.
var _ shared.DomainEvent = PrescriptionComposedEvent{}

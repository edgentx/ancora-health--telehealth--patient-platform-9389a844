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

// PrescriptionTransmittedEventType is the stable wire name emitted when a
// prescription is sent to the pharmacy gateway.
const PrescriptionTransmittedEventType = "prescription.transmitted"

// PrescriptionTransmittedEvent is emitted when a TransmitPrescriptionCmd
// succeeds. It records the pharmacy gateway the prescription was dispatched to.
// Its emission marks the prescription as sealed: thereafter it is immutable and
// may only be superseded by a cancellation.
type PrescriptionTransmittedEvent struct {
	// PrescriptionID is the identity of the PrescriptionAggregate that produced
	// the event.
	PrescriptionID string
	// PharmacyID is the pharmacy gateway the prescription was transmitted to.
	PharmacyID string
}

// Type identifies the event kind.
func (e PrescriptionTransmittedEvent) Type() string { return PrescriptionTransmittedEventType }

// AggregateID ties the event back to the prescription that produced it.
func (e PrescriptionTransmittedEvent) AggregateID() string { return e.PrescriptionID }

// Compile-time assertion that PrescriptionTransmittedEvent satisfies the
// DomainEvent contract.
var _ shared.DomainEvent = PrescriptionTransmittedEvent{}

// PrescriptionSafetyCheckedEventType is the stable wire name emitted when a
// prescription has been run through allergy and interaction verification.
const PrescriptionSafetyCheckedEventType = "prescription.safety.checked"

// PrescriptionSafetyCheckedEvent is emitted when a RunSafetyCheckCmd succeeds. It
// records that the prescription has cleared allergy and interaction verification
// and is therefore safe to transmit.
type PrescriptionSafetyCheckedEvent struct {
	// PrescriptionID is the identity of the PrescriptionAggregate that produced
	// the event.
	PrescriptionID string
}

// Type identifies the event kind.
func (e PrescriptionSafetyCheckedEvent) Type() string { return PrescriptionSafetyCheckedEventType }

// AggregateID ties the event back to the prescription that produced it.
func (e PrescriptionSafetyCheckedEvent) AggregateID() string { return e.PrescriptionID }

// Compile-time assertion that PrescriptionSafetyCheckedEvent satisfies the
// DomainEvent contract.
var _ shared.DomainEvent = PrescriptionSafetyCheckedEvent{}

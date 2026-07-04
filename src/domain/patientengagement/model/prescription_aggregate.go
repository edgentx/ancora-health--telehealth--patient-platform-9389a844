package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// PrescriptionStatus is the lifecycle state of a prescription. The zero value is
// an undrafted prescription, which is what ComposePrescriptionCmd acts on.
type PrescriptionStatus string

const (
	// PrescriptionStatusDraft is a prescription that has not yet been drafted. It
	// is the zero value, so a freshly constructed aggregate is a draft.
	PrescriptionStatusDraft PrescriptionStatus = ""
	// PrescriptionStatusComposed is a drafted prescription awaiting transmission.
	PrescriptionStatusComposed PrescriptionStatus = "composed"
	// PrescriptionStatusTransmitted is a prescription that has been transmitted to
	// the pharmacy. Once transmitted it is immutable and may only be superseded by
	// a cancellation.
	PrescriptionStatusTransmitted PrescriptionStatus = "transmitted"
)

// PrescriptionAggregate is the patient-engagement Prescription aggregate. It
// embeds shared.AggregateRoot for version tracking and an uncommitted-event
// buffer, and carries its own string identity.
//
// Beyond identity it tracks the state that command invariants read: its
// lifecycle status, the patient/provider scoped to it, the drafted medication
// and dosage, and the flags describing whether the issuing provider is
// authorized and whether an allergy/interaction safety check has been cleared.
//
// The invariant flags follow the repository convention that a freshly
// constructed aggregate is valid: their zero value is the compliant state, and a
// non-zero value marks a violation the guards reject.
type PrescriptionAggregate struct {
	shared.AggregateRoot
	ID string

	// Status is the prescription's lifecycle state.
	Status PrescriptionStatus

	// ScopedPatientID and ScopedProviderID are the participants the prescription
	// is bound to. They are empty until the prescription is composed, at which
	// point it is scoped to the drafting patient and provider.
	ScopedPatientID  string
	ScopedProviderID string

	// Medication and Dosage are the drafted order. They are empty until the
	// prescription is composed.
	Medication string
	Dosage     string

	// ProviderUnauthorized reports that the issuing provider is not an
	// authenticated provider with an active care relationship to the patient.
	// Invariant: a prescription may only be issued by such a provider.
	ProviderUnauthorized bool

	// SafetyCheckFailed reports that the prescription failed an allergy or
	// interaction check, and SafetyOverrideAcknowledged reports that such a
	// failure has been acknowledged or overridden. Invariant: a prescription
	// failing a safety check cannot proceed until acknowledged/overridden.
	SafetyCheckFailed          bool
	SafetyOverrideAcknowledged bool
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. Unrecognized command types return shared.ErrUnknownCommand.
func (a *PrescriptionAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch c := cmd.(type) {
	case ComposePrescriptionCmd:
		return a.composePrescription(c)
	default:
		return nil, shared.ErrUnknownCommand
	}
}

// composePrescription handles ComposePrescriptionCmd: it validates the command
// input, enforces the prescription invariants, then emits a
// PrescriptionComposedEvent and buffers it on the aggregate.
//
// The guards enforce, in order:
//
//   - Completeness: patient, provider, medication and dosage must all be present.
//   - Provider authorization: only an authenticated provider with an active care
//     relationship to the patient may issue the prescription.
//   - Safety checks: a prescription failing an allergy or interaction check may
//     not proceed until the failure is acknowledged/overridden.
//   - Immutability: a transmitted prescription is sealed and may only be
//     superseded by a cancellation, never re-composed.
func (a *PrescriptionAggregate) composePrescription(cmd ComposePrescriptionCmd) ([]shared.DomainEvent, error) {
	if cmd.PatientId == "" {
		return nil, ErrMissingPatient
	}
	if cmd.ProviderId == "" {
		return nil, ErrMissingProvider
	}
	if cmd.Medication == "" {
		return nil, ErrMissingMedication
	}
	if cmd.Dosage == "" {
		return nil, ErrMissingDosage
	}

	// Invariant: a prescription may only be issued by an authenticated provider
	// with an active care relationship to the patient.
	if a.ProviderUnauthorized {
		return nil, ErrProviderNotAuthorized
	}

	// Invariant: a prescription failing an allergy or interaction check cannot
	// proceed until the failure has been acknowledged or overridden.
	if a.SafetyCheckFailed && !a.SafetyOverrideAcknowledged {
		return nil, ErrSafetyCheckUnacknowledged
	}

	// Invariant: a transmitted prescription is immutable. Re-composing one would
	// mutate a sealed order, so it is rejected — a change must be a cancellation
	// that supersedes it.
	if a.Status == PrescriptionStatusTransmitted {
		return nil, ErrTransmittedImmutable
	}

	evt := PrescriptionComposedEvent{
		PrescriptionID: a.ID,
		PatientID:      cmd.PatientId,
		ProviderID:     cmd.ProviderId,
		Medication:     cmd.Medication,
		Dosage:         cmd.Dosage,
	}

	a.apply(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// apply mutates aggregate state from a domain event. It is the single place
// state changes, so the same function serves both command handling and future
// event replay when rehydrating the aggregate from the store.
func (a *PrescriptionAggregate) apply(evt PrescriptionComposedEvent) {
	a.Status = PrescriptionStatusComposed
	a.ScopedPatientID = evt.PatientID
	a.ScopedProviderID = evt.ProviderID
	a.Medication = evt.Medication
	a.Dosage = evt.Dosage
}

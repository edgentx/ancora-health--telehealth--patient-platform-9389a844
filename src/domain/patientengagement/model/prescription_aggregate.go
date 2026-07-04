package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// PrescriptionStatus is the lifecycle state of a prescription. The zero value is
// a draft prescription, which is what RunSafetyCheckCmd acts on.
type PrescriptionStatus string

const (
	// PrescriptionStatusDraft is a not-yet-transmitted prescription. It is the zero
	// value, so a freshly constructed aggregate is a draft.
	PrescriptionStatusDraft PrescriptionStatus = ""
	// PrescriptionStatusChecked is a prescription that has cleared allergy and
	// interaction verification and is ready for transmission.
	PrescriptionStatusChecked PrescriptionStatus = "checked"
	// PrescriptionStatusTransmitted is a prescription that has been transmitted to
	// the pharmacy; it is immutable and may only be superseded by a cancellation.
	PrescriptionStatusTransmitted PrescriptionStatus = "transmitted"
	// PrescriptionStatusCancelled is a cancelled prescription; it is terminal.
	PrescriptionStatusCancelled PrescriptionStatus = "cancelled"
)

// PrescriptionAggregate is the patient-engagement aggregate that tracks a
// prescription through its lifecycle. It embeds shared.AggregateRoot for version
// tracking and event buffering, and carries its own string identity.
//
// Beyond identity it tracks the state that command invariants read: its
// lifecycle status, the patient/provider scoped to it, whether the issuing
// provider is authenticated with an active care relationship, and whether an
// outstanding allergy or interaction failure has been acknowledged or
// overridden.
type PrescriptionAggregate struct {
	shared.AggregateRoot
	ID string

	// Status is the prescription's lifecycle state.
	Status PrescriptionStatus

	// ScopedPatientID and ScopedProviderID are the participants the prescription is
	// bound to. When non-empty they constrain who may issue or act on it; an empty
	// value means the participant is bound on the first successful command.
	ScopedPatientID  string
	ScopedProviderID string

	// ProviderAuthenticated reports whether the scoped provider is authenticated.
	// Only an authenticated provider may issue or act on a prescription.
	ProviderAuthenticated bool

	// CareRelationshipActive reports whether the scoped provider holds an active
	// care relationship to the scoped patient.
	CareRelationshipActive bool

	// SafetyFailurePending reports whether the prescription has an allergy or
	// interaction failure outstanding. When true it must be acknowledged or
	// overridden before the prescription may progress.
	SafetyFailurePending bool

	// SafetyFailureAcknowledged reports whether an outstanding safety failure has
	// been acknowledged or overridden by the provider.
	SafetyFailureAcknowledged bool
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. Unrecognized command types return shared.ErrUnknownCommand.
func (a *PrescriptionAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch c := cmd.(type) {
	case RunSafetyCheckCmd:
		return a.runSafetyCheck(c)
	default:
		return nil, shared.ErrUnknownCommand
	}
}

// runSafetyCheck handles RunSafetyCheckCmd: it validates the command input,
// enforces the prescription invariants, then emits a
// PrescriptionSafetyCheckedEvent and buffers it on the aggregate.
//
// The guards enforce, in order:
//
//   - Completeness: prescription, provider and patient ids must all be present.
//   - Authenticated provider in care: a prescription may only be issued by an
//     authenticated provider with an active care relationship to the patient.
//   - Acknowledged safety failure: a prescription failing an allergy or
//     interaction check cannot progress until the failure is acknowledged or
//     overridden.
//   - Immutability: a transmitted prescription is immutable and may only be
//     superseded by a cancellation, so it cannot be acted on.
func (a *PrescriptionAggregate) runSafetyCheck(cmd RunSafetyCheckCmd) ([]shared.DomainEvent, error) {
	if cmd.PrescriptionId == "" {
		return nil, ErrMissingPrescriptionID
	}
	if cmd.ProviderId == "" {
		return nil, ErrMissingPrescriptionProvider
	}
	if cmd.PatientId == "" {
		return nil, ErrMissingPrescriptionPatient
	}

	// Invariant: a prescription may only be issued by an authenticated provider
	// with an active care relationship. When the prescription is already scoped,
	// the command must name the same participants, and that provider must still be
	// authenticated with an active care relationship.
	if a.ScopedProviderID != "" && a.ScopedProviderID != cmd.ProviderId {
		return nil, ErrProviderNotInCare
	}
	if a.ScopedPatientID != "" && a.ScopedPatientID != cmd.PatientId {
		return nil, ErrProviderNotInCare
	}
	if a.ScopedProviderID != "" && (!a.ProviderAuthenticated || !a.CareRelationshipActive) {
		return nil, ErrProviderNotInCare
	}

	// Invariant: a prescription failing an allergy or interaction check cannot be
	// transmitted until the failure is acknowledged or overridden. An outstanding,
	// unacknowledged failure blocks the safety check from clearing.
	if a.SafetyFailurePending && !a.SafetyFailureAcknowledged {
		return nil, ErrUnacknowledgedSafetyFailure
	}

	// Invariant: a transmitted prescription is immutable and may only be superseded
	// by a cancellation. Both terminal states reject any further command.
	if a.Status == PrescriptionStatusTransmitted || a.Status == PrescriptionStatusCancelled {
		return nil, ErrPrescriptionTransmitted
	}

	evt := PrescriptionSafetyCheckedEvent{
		PrescriptionID: a.ID,
		ProviderID:     cmd.ProviderId,
		PatientID:      cmd.PatientId,
	}

	a.apply(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// apply mutates aggregate state from a PrescriptionSafetyCheckedEvent. It is the
// single place these state changes happen, so the same function serves both
// command handling and future event replay when rehydrating the aggregate from
// the store.
func (a *PrescriptionAggregate) apply(evt PrescriptionSafetyCheckedEvent) {
	a.Status = PrescriptionStatusChecked
	a.ScopedProviderID = evt.ProviderID
	a.ScopedPatientID = evt.PatientID
	a.ProviderAuthenticated = true
	a.CareRelationshipActive = true
}

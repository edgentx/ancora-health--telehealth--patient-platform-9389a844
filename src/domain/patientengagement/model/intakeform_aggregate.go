package model

import (
	"strings"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

// IntakeFormStatus is the lifecycle state of an intake form. The zero value is a
// draft form still being completed, which is what SubmitIntakeFormCmd acts on.
type IntakeFormStatus string

const (
	// IntakeFormStatusDraft is an intake form that has not yet been submitted. It
	// is the zero value, so a freshly constructed aggregate is a draft.
	IntakeFormStatusDraft IntakeFormStatus = ""
	// IntakeFormStatusSubmitted is an intake form whose data has been submitted to
	// seed the chart. Once submitted it is immutable and may only be corrected by
	// a new submission.
	IntakeFormStatusSubmitted IntakeFormStatus = "submitted"
)

// IntakeFormAggregate is the patient-engagement IntakeForm aggregate. It embeds
// shared.AggregateRoot for version tracking and an uncommitted-event buffer, and
// carries its own string identity in ID.
//
// Beyond identity it tracks the state that command invariants read: its
// lifecycle status, the patient it is scoped to, the captured history and
// demographics, and the flag reporting whether the form is still incomplete or
// unvalidated.
//
// The invariant flag follows the repository convention that a freshly
// constructed aggregate is valid: its zero value is the compliant state, and a
// non-zero value marks a violation the guards reject.
type IntakeFormAggregate struct {
	shared.AggregateRoot
	ID string

	// Status is the intake form's lifecycle state.
	Status IntakeFormStatus

	// ScopedPatientID is the patient the intake form is bound to. It is empty
	// until the form is submitted, at which point it is scoped to the completing
	// patient.
	ScopedPatientID string

	// History and Demographics are the submitted intake data. They are empty until
	// the form is submitted.
	History      string
	Demographics string

	// Incomplete reports that the intake form has not yet been fully completed and
	// validated. Invariant: an intake form must be complete and validated before
	// it can seed the chart, so submission is rejected while this is set.
	Incomplete bool
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. Unrecognized command types return shared.ErrUnknownCommand.
func (a *IntakeFormAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch c := cmd.(type) {
	case SubmitIntakeFormCmd:
		return a.submitIntakeForm(c)
	default:
		return nil, shared.ErrUnknownCommand
	}
}

// submitIntakeForm handles SubmitIntakeFormCmd: it validates the command input,
// enforces the intake-form invariants, then emits an IntakeFormSubmittedEvent and
// buffers it on the aggregate.
//
// The guards enforce, in order:
//
//   - Completeness of input: patient, history and demographics must all be
//     present on the command.
//   - Validation: the form must be complete and validated before it can seed the
//     chart; a form still flagged incomplete is rejected.
//   - Immutability: submitted intake data is sealed and may only be corrected via
//     a new submission, so an already-submitted form is never re-submitted.
func (a *IntakeFormAggregate) submitIntakeForm(cmd SubmitIntakeFormCmd) ([]shared.DomainEvent, error) {
	if strings.TrimSpace(cmd.PatientId) == "" {
		return nil, ErrMissingIntakePatient
	}
	if strings.TrimSpace(cmd.History) == "" {
		return nil, ErrMissingIntakeHistory
	}
	if strings.TrimSpace(cmd.Demographics) == "" {
		return nil, ErrMissingIntakeDemographics
	}

	// Invariant: an intake form must be complete and validated before it can seed
	// the chart.
	if a.Incomplete {
		return nil, ErrIntakeFormIncomplete
	}

	// Invariant: submitted intake data is immutable. Re-submitting a sealed form
	// would mutate chart-seeding data already in flight, so it is rejected — a
	// correction must be a new submission that supersedes it.
	if a.Status == IntakeFormStatusSubmitted {
		return nil, ErrIntakeFormSubmittedImmutable
	}

	evt := IntakeFormSubmittedEvent{
		IntakeFormID: a.ID,
		PatientID:    cmd.PatientId,
		History:      cmd.History,
		Demographics: cmd.Demographics,
	}

	a.apply(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// apply mutates aggregate state from an IntakeFormSubmittedEvent. It is the
// single place state changes, so the same function serves both command handling
// and future event replay when rehydrating the aggregate from the store.
func (a *IntakeFormAggregate) apply(evt IntakeFormSubmittedEvent) {
	a.Status = IntakeFormStatusSubmitted
	a.ScopedPatientID = evt.PatientID
	a.History = evt.History
	a.Demographics = evt.Demographics
}

package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// IntakeStatus is the lifecycle state of an intake form. The zero value is a
// not-yet-seeded intake, which is the state SeedChartFromIntakeCmd acts on.
type IntakeStatus string

const (
	// IntakeStatusPending is an intake form that has been submitted but whose data
	// has not yet seeded the chart. It is the zero value, so a freshly constructed
	// aggregate is pending.
	IntakeStatusPending IntakeStatus = ""
	// IntakeStatusSeeded is an intake whose data has been projected into the
	// patient's chart. Once seeded the submission is immutable and may only be
	// corrected via a new submission.
	IntakeStatusSeeded IntakeStatus = "seeded"
)

// IntakeFormAggregate is the patient-engagement IntakeForm aggregate. It embeds
// shared.AggregateRoot for version tracking and an uncommitted-event buffer, and
// carries its own string identity.
//
// Beyond identity it tracks the state that command invariants read: its
// lifecycle status, the patient the seeded chart is scoped to, and a flag
// describing whether the form has been completed and validated.
//
// The invariant flag follows the repository convention that a freshly
// constructed aggregate is valid: its zero value is the compliant state, and a
// non-zero value marks a violation the guards reject.
type IntakeFormAggregate struct {
	shared.AggregateRoot
	ID string

	// Status is the intake form's lifecycle state.
	Status IntakeStatus

	// ScopedPatientID is the patient the seeded chart is bound to. It is empty
	// until the intake seeds the chart, at which point it is scoped to that
	// patient.
	ScopedPatientID string

	// NotValidated reports that the intake form has not been completed and
	// validated. Invariant: an intake form must be complete and validated before
	// it can seed the chart.
	NotValidated bool
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. Unrecognized command types return shared.ErrUnknownCommand.
func (a *IntakeFormAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch c := cmd.(type) {
	case SeedChartFromIntakeCmd:
		return a.seedChartFromIntake(c)
	default:
		return nil, shared.ErrUnknownCommand
	}
}

// seedChartFromIntake handles SeedChartFromIntakeCmd: it validates the command
// input, enforces the intake invariants, then emits an IntakeChartSeededEvent
// and buffers it on the aggregate.
//
// The guards enforce, in order:
//
//   - Completeness: the intake and patient must both be identified.
//   - Validation: an intake form must be complete and validated before it can
//     seed the chart.
//   - Immutability: submitted intake data is immutable, so an intake that has
//     already seeded the chart may only be corrected via a new submission, never
//     re-seeded.
func (a *IntakeFormAggregate) seedChartFromIntake(cmd SeedChartFromIntakeCmd) ([]shared.DomainEvent, error) {
	if cmd.IntakeId == "" {
		return nil, ErrMissingIntake
	}
	if cmd.PatientId == "" {
		return nil, ErrMissingIntakePatient
	}

	// Invariant: an intake form must be complete and validated before it can seed
	// the chart.
	if a.NotValidated {
		return nil, ErrIntakeNotValidated
	}

	// Invariant: submitted intake data is immutable. An intake that has already
	// seeded the chart cannot be re-seeded — a correction must be a new
	// submission that supersedes it.
	if a.Status == IntakeStatusSeeded {
		return nil, ErrIntakeImmutable
	}

	evt := IntakeChartSeededEvent{
		IntakeID:  a.ID,
		PatientID: cmd.PatientId,
	}

	a.apply(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// apply mutates aggregate state from a domain event. It is the single place
// state changes, so the same function serves both command handling and future
// event replay when rehydrating the aggregate from the store.
func (a *IntakeFormAggregate) apply(evt IntakeChartSeededEvent) {
	a.Status = IntakeStatusSeeded
	a.ScopedPatientID = evt.PatientID
}

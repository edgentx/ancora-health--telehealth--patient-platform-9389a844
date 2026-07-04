// Package model holds the aggregates for the billing-and-insurance bounded
// context.
package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// InvoiceStatus is the lifecycle state of an invoice. The zero value is a draft
// invoice that has not yet been voided, which is the state
// ApplyInsuranceAdjustmentCmd acts on.
type InvoiceStatus string

const (
	// InvoiceStatusDraft is an invoice that is open for adjustment and payment. It
	// is the zero value, so a freshly constructed aggregate is a draft.
	InvoiceStatusDraft InvoiceStatus = ""
	// InvoiceStatusAdjusted is an invoice that has had a verified insurance
	// eligibility result applied to it.
	InvoiceStatusAdjusted InvoiceStatus = "adjusted"
	// InvoiceStatusVoided is an invoice that has been voided. A voided invoice is
	// closed and cannot receive further payments or adjustments.
	InvoiceStatusVoided InvoiceStatus = "voided"
)

// InvoiceAggregate is the billing-and-insurance aggregate that tracks an invoice
// through its lifecycle. It embeds shared.AggregateRoot for version tracking and
// event buffering.
//
// Beyond identity it carries the state the command invariants read: its
// lifecycle status, whether it was generated from a completed encounter, the
// money fields that must reconcile (charges, insurance adjustment, copay and
// patient responsibility, all in cents), and the amounts already paid against
// the outstanding balance.
//
// Money fields follow the repository convention that a freshly constructed
// aggregate is valid: the zero value reconciles under every invariant
// (0 == 0 - 0 - 0, 0 paid against 0 outstanding), so a new aggregate passes all
// guards and only a deliberately inconsistent aggregate is rejected.
type InvoiceAggregate struct {
	shared.AggregateRoot
	ID string

	// Status is the invoice's lifecycle state.
	Status InvoiceStatus

	// EncounterIncomplete reports that the invoice was generated from an encounter
	// that is not yet complete. Invariant: an invoice may only be generated from a
	// completed encounter, so a command is rejected while this is set.
	EncounterIncomplete bool

	// Charges is the total billed amount, in cents, before insurance.
	Charges int64
	// InsuranceAdjustment is the verified coverage, in cents, applied against the
	// charges.
	InsuranceAdjustment int64
	// Copay is the patient copay, in cents.
	Copay int64
	// PatientResponsibility is the amount, in cents, owed by the patient. It must
	// reconcile to Charges - InsuranceAdjustment - Copay.
	PatientResponsibility int64

	// OutstandingBalance is the amount, in cents, still due on the invoice.
	OutstandingBalance int64
	// AmountPaid is the amount, in cents, already paid against the invoice. It may
	// never exceed OutstandingBalance.
	AmountPaid int64
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. Unrecognized command types return shared.ErrUnknownCommand.
func (a *InvoiceAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch c := cmd.(type) {
	case ApplyInsuranceAdjustmentCmd:
		return a.applyInsuranceAdjustment(c)
	default:
		return nil, shared.ErrUnknownCommand
	}
}

// applyInsuranceAdjustment handles ApplyInsuranceAdjustmentCmd: it validates the
// command input, enforces the invoice invariants, then emits an
// InvoiceAdjustedEvent and buffers it on the aggregate.
//
// The guards enforce, in order:
//
//   - Input validity: the invoice id is present and the eligibility amounts are
//     non-negative.
//   - Encounter completion: an invoice may only be generated from a completed
//     encounter.
//   - Reconciliation: the recorded patient responsibility must equal charges
//     minus the verified insurance adjustment and copay.
//   - No overpayment: an invoice cannot be marked paid for more than its
//     outstanding balance.
//   - Void state: a voided invoice cannot receive further payments.
func (a *InvoiceAggregate) applyInsuranceAdjustment(cmd ApplyInsuranceAdjustmentCmd) ([]shared.DomainEvent, error) {
	if cmd.InvoiceId == "" {
		return nil, ErrMissingInvoiceID
	}
	if cmd.Eligibility.VerifiedAdjustment < 0 {
		return nil, ErrNegativeAdjustment
	}
	if cmd.Eligibility.Copay < 0 {
		return nil, ErrNegativeCopay
	}

	// Invariant: an invoice may only be generated from a completed encounter.
	if a.EncounterIncomplete {
		return nil, ErrEncounterNotCompleted
	}

	// Invariant: patient responsibility must equal charges minus verified
	// insurance adjustment and copay.
	if a.PatientResponsibility != a.Charges-a.InsuranceAdjustment-a.Copay {
		return nil, ErrPatientResponsibilityMismatch
	}

	// Invariant: an invoice cannot be marked paid for more than its outstanding
	// balance.
	if a.AmountPaid > a.OutstandingBalance {
		return nil, ErrOverpaid
	}

	// Invariant: a voided invoice cannot receive further payments.
	if a.Status == InvoiceStatusVoided {
		return nil, ErrVoidedInvoice
	}

	evt := InvoiceAdjustedEvent{
		InvoiceID:             a.ID,
		InsuranceAdjustment:   cmd.Eligibility.VerifiedAdjustment,
		Copay:                 cmd.Eligibility.Copay,
		PatientResponsibility: a.Charges - cmd.Eligibility.VerifiedAdjustment - cmd.Eligibility.Copay,
	}

	a.apply(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// apply mutates aggregate state from an InvoiceAdjustedEvent. It is the single
// place state changes, so the same function serves both command handling and
// future event replay when rehydrating the aggregate from the store.
func (a *InvoiceAggregate) apply(evt InvoiceAdjustedEvent) {
	a.Status = InvoiceStatusAdjusted
	a.InsuranceAdjustment = evt.InsuranceAdjustment
	a.Copay = evt.Copay
	a.PatientResponsibility = evt.PatientResponsibility
}

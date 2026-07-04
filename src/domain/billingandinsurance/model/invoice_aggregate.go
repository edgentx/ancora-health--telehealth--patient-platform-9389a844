// Package model holds the aggregates for the billing-and-insurance bounded
// context.
package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// InvoiceStatus is the lifecycle state of an invoice. The zero value is an
// ungenerated invoice, which is what GenerateInvoiceCmd acts on.
type InvoiceStatus string

const (
	// InvoiceStatusNew is an invoice that has not yet been generated. It is the
	// zero value, so a freshly constructed aggregate is new.
	InvoiceStatusNew InvoiceStatus = ""
	// InvoiceStatusGenerated is an invoice that has been generated from a
	// completed encounter and is now billable.
	InvoiceStatusGenerated InvoiceStatus = "generated"
	// InvoiceStatusAdjusted is a generated invoice whose verified insurance
	// coverage and copay have been applied, reconciling patient responsibility.
	InvoiceStatusAdjusted InvoiceStatus = "adjusted"
)

// InvoiceAggregate is the billing-and-insurance aggregate that tracks an invoice
// through its lifecycle. It embeds shared.AggregateRoot for version tracking and
// event buffering, and carries its own identity in ID.
//
// Beyond identity it tracks the state that command invariants read: its
// lifecycle status, the encounter it was generated from, the billable line
// items, the insurance policy its claim is adjudicated against, and the flags
// describing whether the source encounter is completed, whether the patient
// responsibility reconciles against charges, whether a payment would exceed the
// outstanding balance, and whether the invoice has been voided.
//
// The invariant flags follow the repository convention that a freshly
// constructed aggregate is valid: their zero value is the compliant state, and a
// non-zero value marks a violation the guards reject.
type InvoiceAggregate struct {
	shared.AggregateRoot
	ID string

	// Status is the invoice's lifecycle state.
	Status InvoiceStatus

	// EncounterID is the encounter the invoice was generated from. It is empty
	// until the invoice is generated.
	EncounterID string

	// LineItems are the billable charges captured on the invoice. They are empty
	// until the invoice is generated.
	LineItems []InvoiceLineItem

	// PolicyID is the insurance policy the invoice's claim is adjudicated
	// against. It is empty until the invoice is generated.
	PolicyID string

	// EncounterNotCompleted reports that the source encounter is not completed.
	// Invariant: an invoice may only be generated from a completed encounter.
	EncounterNotCompleted bool

	// PatientResponsibilityMismatch reports that the recorded patient
	// responsibility does not reconcile against charges. Invariant: patient
	// responsibility must equal charges minus verified insurance adjustment and
	// copay.
	PatientResponsibilityMismatch bool

	// PaymentExceedsOutstanding reports that a payment would exceed the invoice's
	// outstanding balance. Invariant: an invoice cannot be marked paid for more
	// than its outstanding balance.
	PaymentExceedsOutstanding bool

	// Voided reports that the invoice has been voided and so cannot receive
	// further payments. Invariant: a voided invoice cannot receive further
	// payments.
	Voided bool

	// CoverageCents is the verified insurance adjustment applied to the invoice,
	// in whole cents. It is zero until the adjustment is applied.
	CoverageCents int64

	// CopayCents is the patient copay applied to the invoice, in whole cents. It
	// is zero until the adjustment is applied.
	CopayCents int64
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. Unrecognized command types return shared.ErrUnknownCommand.
func (a *InvoiceAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch c := cmd.(type) {
	case GenerateInvoiceCmd:
		return a.generateInvoice(c)
	case ApplyInsuranceAdjustmentCmd:
		return a.applyInsuranceAdjustment(c)
	default:
		return nil, shared.ErrUnknownCommand
	}
}

// generateInvoice handles GenerateInvoiceCmd: it validates the command input,
// enforces the invoice invariants, then emits an InvoiceGeneratedEvent and
// buffers it on the aggregate.
//
// The guards enforce, in order:
//
//   - Completeness: the encounter, line items and policy must all be present.
//   - Completed encounter: an invoice may only be generated from a completed
//     encounter.
//   - Patient responsibility: it must equal charges minus the verified insurance
//     adjustment and copay.
//   - Outstanding balance: an invoice cannot be marked paid for more than its
//     outstanding balance.
//   - Voided invoice: a voided invoice cannot receive further payments.
func (a *InvoiceAggregate) generateInvoice(cmd GenerateInvoiceCmd) ([]shared.DomainEvent, error) {
	if cmd.EncounterId == "" {
		return nil, ErrMissingEncounter
	}
	if len(cmd.LineItems) == 0 {
		return nil, ErrMissingLineItems
	}
	if cmd.PolicyId == "" {
		return nil, ErrMissingPolicy
	}

	// Invariant: an invoice may only be generated from a completed encounter.
	if a.EncounterNotCompleted {
		return nil, ErrEncounterNotCompleted
	}

	// Invariant: patient responsibility must equal charges minus verified
	// insurance adjustment and copay.
	if a.PatientResponsibilityMismatch {
		return nil, ErrPatientResponsibilityMismatch
	}

	// Invariant: an invoice cannot be marked paid for more than its outstanding
	// balance.
	if a.PaymentExceedsOutstanding {
		return nil, ErrPaymentExceedsOutstanding
	}

	// Invariant: a voided invoice cannot receive further payments.
	if a.Voided {
		return nil, ErrVoidedInvoicePayment
	}

	evt := InvoiceGeneratedEvent{
		InvoiceID:   a.ID,
		EncounterID: cmd.EncounterId,
		LineItems:   cmd.LineItems,
		PolicyID:    cmd.PolicyId,
	}

	a.apply(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// apply mutates aggregate state from a domain event. It is the single place
// state changes, so the same function serves both command handling and future
// event replay when rehydrating the aggregate from the store.
func (a *InvoiceAggregate) apply(evt InvoiceGeneratedEvent) {
	a.Status = InvoiceStatusGenerated
	a.EncounterID = evt.EncounterID
	a.LineItems = evt.LineItems
	a.PolicyID = evt.PolicyID
}

// applyInsuranceAdjustment handles ApplyInsuranceAdjustmentCmd: it validates the
// command input, enforces the invoice invariants, then emits an
// InvoiceAdjustedEvent and buffers it on the aggregate.
//
// The guards enforce, in order:
//
//   - Completeness: the invoice id and a verified, non-negative eligibility
//     result must be present.
//   - Completed encounter: an invoice may only be generated from a completed
//     encounter.
//   - Patient responsibility: it must equal charges minus the verified insurance
//     adjustment and copay.
//   - Outstanding balance: an invoice cannot be marked paid for more than its
//     outstanding balance.
//   - Voided invoice: a voided invoice cannot receive further payments.
func (a *InvoiceAggregate) applyInsuranceAdjustment(cmd ApplyInsuranceAdjustmentCmd) ([]shared.DomainEvent, error) {
	if cmd.InvoiceId == "" {
		return nil, ErrMissingInvoiceID
	}
	if !cmd.Eligibility.Verified {
		return nil, ErrUnverifiedEligibility
	}
	if cmd.Eligibility.CoverageCents < 0 || cmd.Eligibility.CopayCents < 0 {
		return nil, ErrNegativeAdjustment
	}

	// Invariant: an invoice may only be generated from a completed encounter.
	if a.EncounterNotCompleted {
		return nil, ErrEncounterNotCompleted
	}

	// Invariant: patient responsibility must equal charges minus verified
	// insurance adjustment and copay.
	if a.PatientResponsibilityMismatch {
		return nil, ErrPatientResponsibilityMismatch
	}

	// Invariant: an invoice cannot be marked paid for more than its outstanding
	// balance.
	if a.PaymentExceedsOutstanding {
		return nil, ErrPaymentExceedsOutstanding
	}

	// Invariant: a voided invoice cannot receive further payments.
	if a.Voided {
		return nil, ErrVoidedInvoicePayment
	}

	evt := InvoiceAdjustedEvent{
		InvoiceID:     a.ID,
		CoverageCents: cmd.Eligibility.CoverageCents,
		CopayCents:    cmd.Eligibility.CopayCents,
	}

	a.applyAdjusted(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// applyAdjusted mutates aggregate state from an InvoiceAdjustedEvent. Like
// apply, it is the single place adjustment state changes, so the same function
// serves both command handling and future event replay.
func (a *InvoiceAggregate) applyAdjusted(evt InvoiceAdjustedEvent) {
	a.Status = InvoiceStatusAdjusted
	a.CoverageCents = evt.CoverageCents
	a.CopayCents = evt.CopayCents
}

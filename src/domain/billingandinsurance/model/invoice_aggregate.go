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
	// InvoiceStatusVoided is an invoice that has been voided and can no longer be
	// paid.
	InvoiceStatusVoided InvoiceStatus = "voided"
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
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. Unrecognized command types return shared.ErrUnknownCommand.
func (a *InvoiceAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch c := cmd.(type) {
	case GenerateInvoiceCmd:
		return a.generateInvoice(c)
	case VoidInvoiceCmd:
		return a.voidInvoice(c)
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

// voidInvoice handles VoidInvoiceCmd: it validates the command input, enforces
// the invoice invariants, then emits an InvoiceVoidedEvent and buffers it on
// the aggregate.
//
// The guards enforce, in order:
//
//   - Completeness: the invoice id and a void reason must both be present.
//   - Completed encounter: an invoice may only be generated from a completed
//     encounter.
//   - Patient responsibility: it must equal charges minus the verified insurance
//     adjustment and copay.
//   - Outstanding balance: an invoice cannot be marked paid for more than its
//     outstanding balance.
//   - Voided invoice: a voided invoice cannot receive further payments.
func (a *InvoiceAggregate) voidInvoice(cmd VoidInvoiceCmd) ([]shared.DomainEvent, error) {
	if cmd.InvoiceId == "" {
		return nil, ErrMissingInvoiceID
	}
	if cmd.Reason == "" {
		return nil, ErrMissingVoidReason
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

	evt := InvoiceVoidedEvent{
		InvoiceID: a.ID,
		Reason:    cmd.Reason,
	}

	a.applyVoided(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// applyVoided mutates aggregate state from an InvoiceVoidedEvent. Like apply it
// is the single place voided-state changes, so it serves both command handling
// and future event replay when rehydrating the aggregate from the store.
func (a *InvoiceAggregate) applyVoided(evt InvoiceVoidedEvent) {
	a.Status = InvoiceStatusVoided
	a.Voided = true
}

// Package model holds the aggregates for the billing-and-insurance bounded
// context.
package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// InvoiceStatus is the lifecycle state of an invoice. The zero value is an
// issued invoice, which is what VoidInvoiceCmd acts on.
type InvoiceStatus string

const (
	// InvoiceStatusIssued is an invoice that has been issued and is live in the
	// billing lifecycle. It is the zero value, so a freshly constructed aggregate
	// is an issued invoice ready to be voided.
	InvoiceStatusIssued InvoiceStatus = ""
	// InvoiceStatusVoided is an invoice that has been voided. Once voided it is
	// closed and can receive no further payments.
	InvoiceStatusVoided InvoiceStatus = "voided"
)

// InvoiceAggregate is the billing-and-insurance aggregate that tracks an invoice
// through its lifecycle. It embeds shared.AggregateRoot for version tracking and
// event buffering, and carries its own identity in ID.
//
// Beyond identity it tracks its lifecycle status and the flags describing the
// billing invariants that command guards read: whether the invoice was
// generated from a completed encounter, whether patient responsibility
// reconciles against charges/insurance/copay, whether it was ever marked paid
// beyond its outstanding balance, and whether a voided invoice has received a
// further payment.
//
// The invariant flags follow the repository convention that a freshly
// constructed aggregate is valid: their zero value is the compliant state, and a
// non-zero value marks a violation the guards reject.
type InvoiceAggregate struct {
	shared.AggregateRoot
	ID string

	// Status is the invoice's lifecycle state.
	Status InvoiceStatus

	// NotFromCompletedEncounter reports that the invoice was not generated from a
	// completed encounter. Invariant: an invoice may only be generated from a
	// completed encounter.
	NotFromCompletedEncounter bool

	// PatientResponsibilityMismatch reports that the invoice's patient
	// responsibility does not reconcile. Invariant: patient responsibility must
	// equal charges minus verified insurance adjustment and copay.
	PatientResponsibilityMismatch bool

	// PaidOverOutstandingBalance reports that the invoice was marked paid for more
	// than its outstanding balance. Invariant: an invoice cannot be marked paid
	// for more than its outstanding balance.
	PaidOverOutstandingBalance bool

	// VoidedInvoiceReceivedPayment reports that a voided invoice has received a
	// further payment. Invariant: a voided invoice cannot receive further
	// payments.
	VoidedInvoiceReceivedPayment bool
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. Unrecognized command types return shared.ErrUnknownCommand.
func (a *InvoiceAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch c := cmd.(type) {
	case VoidInvoiceCmd:
		return a.voidInvoice(c)
	default:
		return nil, shared.ErrUnknownCommand
	}
}

// voidInvoice handles VoidInvoiceCmd: it validates the command input, enforces
// the invoice invariants, then emits an InvoiceVoidedEvent and buffers it on the
// aggregate.
//
// The guards enforce, in order:
//
//   - Completeness: the invoice id and void reason must both be present.
//   - Encounter provenance: the invoice may only have been generated from a
//     completed encounter.
//   - Responsibility reconciliation: patient responsibility must equal charges
//     minus verified insurance adjustment and copay.
//   - Payment ceiling: the invoice cannot have been marked paid for more than its
//     outstanding balance.
//   - Void integrity: a voided invoice cannot have received further payments.
func (a *InvoiceAggregate) voidInvoice(cmd VoidInvoiceCmd) ([]shared.DomainEvent, error) {
	if cmd.InvoiceId == "" {
		return nil, ErrMissingInvoiceID
	}
	if cmd.Reason == "" {
		return nil, ErrMissingVoidReason
	}

	// Invariant: an invoice may only be generated from a completed encounter.
	if a.NotFromCompletedEncounter {
		return nil, ErrInvoiceNotFromCompletedEncounter
	}

	// Invariant: patient responsibility must equal charges minus verified
	// insurance adjustment and copay.
	if a.PatientResponsibilityMismatch {
		return nil, ErrPatientResponsibilityMismatch
	}

	// Invariant: an invoice cannot be marked paid for more than its outstanding
	// balance.
	if a.PaidOverOutstandingBalance {
		return nil, ErrPaidOverOutstandingBalance
	}

	// Invariant: a voided invoice cannot receive further payments.
	if a.VoidedInvoiceReceivedPayment {
		return nil, ErrVoidedInvoiceReceivedPayment
	}

	evt := InvoiceVoidedEvent{
		InvoiceID: a.ID,
		Reason:    cmd.Reason,
	}

	a.apply(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// apply mutates aggregate state from a domain event. It is the single place
// state changes, so the same function serves both command handling and future
// event replay when rehydrating the aggregate from the store.
func (a *InvoiceAggregate) apply(evt InvoiceVoidedEvent) {
	a.Status = InvoiceStatusVoided
}

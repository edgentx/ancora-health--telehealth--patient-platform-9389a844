package model

import (
	"errors"
	"testing"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

func validInvoiceAggregate() *InvoiceAggregate {
	return &InvoiceAggregate{ID: "invoice-1"}
}

func validGenerateInvoiceCmd() GenerateInvoiceCmd {
	return GenerateInvoiceCmd{
		EncounterId: "encounter-1",
		LineItems:   []InvoiceLineItem{{Description: "office visit", AmountCents: 12000}},
		PolicyId:    "policy-1",
	}
}

func validApplyInsuranceAdjustmentCmd() ApplyInsuranceAdjustmentCmd {
	return ApplyInsuranceAdjustmentCmd{
		InvoiceId:   "invoice-1",
		Eligibility: EligibilityResult{Verified: true, CoverageCents: 8000, CopayCents: 2000},
	}
}

func validVoidInvoiceCmd() VoidInvoiceCmd {
	return VoidInvoiceCmd{InvoiceId: "invoice-1", Reason: "billed in error"}
}

// invoiceInvariantCases flips one shared invariant flag at a time and expects
// the matching sentinel error. These guards run after field-completeness checks
// in every handler.
func invoiceInvariantCases() []struct {
	name    string
	mutate  func(*InvoiceAggregate)
	wantErr error
} {
	return []struct {
		name    string
		mutate  func(*InvoiceAggregate)
		wantErr error
	}{
		{
			name:    "encounter not completed",
			mutate:  func(a *InvoiceAggregate) { a.EncounterNotCompleted = true },
			wantErr: ErrEncounterNotCompleted,
		},
		{
			name:    "patient responsibility mismatch",
			mutate:  func(a *InvoiceAggregate) { a.PatientResponsibilityMismatch = true },
			wantErr: ErrPatientResponsibilityMismatch,
		},
		{
			name:    "payment exceeds outstanding",
			mutate:  func(a *InvoiceAggregate) { a.PaymentExceedsOutstanding = true },
			wantErr: ErrPaymentExceedsOutstanding,
		},
		{
			name:    "voided invoice",
			mutate:  func(a *InvoiceAggregate) { a.Voided = true },
			wantErr: ErrVoidedInvoicePayment,
		},
	}
}

func assertInvoiceRejected(t *testing.T, agg *InvoiceAggregate, events []shared.DomainEvent, err error, wantErr error) {
	t.Helper()
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
	if len(events) != 0 {
		t.Fatalf("expected no events on rejection, got %d", len(events))
	}
	if got := agg.Events(); len(got) != 0 {
		t.Fatalf("expected no buffered events on rejection, got %d", len(got))
	}
	if agg.Version != 0 {
		t.Fatalf("expected version to remain 0 on rejection, got %d", agg.Version)
	}
}

func TestGenerateInvoiceEmitsInvoiceGeneratedEvent(t *testing.T) {
	agg := validInvoiceAggregate()
	cmd := validGenerateInvoiceCmd()

	events, err := agg.Execute(cmd)
	if err != nil {
		t.Fatalf("Execute(GenerateInvoiceCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	evt, ok := events[0].(InvoiceGeneratedEvent)
	if !ok {
		t.Fatalf("event type = %T, want InvoiceGeneratedEvent", events[0])
	}
	if evt.Type() != InvoiceGeneratedEventType || evt.Type() != "invoice.generated" {
		t.Fatalf("unexpected event type %q", evt.Type())
	}
	if evt.AggregateID() != agg.ID {
		t.Fatalf("event aggregate id = %q, want %q", evt.AggregateID(), agg.ID)
	}
	if evt.InvoiceID != agg.ID {
		t.Fatalf("event invoice id = %q, want %q", evt.InvoiceID, agg.ID)
	}
	if evt.EncounterID != cmd.EncounterId || evt.PolicyID != cmd.PolicyId {
		t.Fatalf("event fields not copied from command: %#v", evt)
	}
	if len(evt.LineItems) != 1 || evt.LineItems[0] != cmd.LineItems[0] {
		t.Fatalf("event line items = %#v, want %#v", evt.LineItems, cmd.LineItems)
	}

	if agg.Status != InvoiceStatusGenerated {
		t.Fatalf("aggregate status = %q, want %q", agg.Status, InvoiceStatusGenerated)
	}
	if agg.EncounterID != cmd.EncounterId || agg.PolicyID != cmd.PolicyId {
		t.Fatalf("aggregate not scoped to generation: %#v", agg)
	}
	if len(agg.LineItems) != 1 || agg.LineItems[0] != cmd.LineItems[0] {
		t.Fatalf("aggregate line items = %#v, want %#v", agg.LineItems, cmd.LineItems)
	}
	if agg.Version != 1 {
		t.Fatalf("aggregate version = %d, want 1", agg.Version)
	}
	if buffered := agg.Events(); len(buffered) != 1 {
		t.Fatalf("aggregate buffered %d events, want 1", len(buffered))
	}
}

func TestGenerateInvoiceRejectsMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		cmd     GenerateInvoiceCmd
		wantErr error
	}{
		{
			name:    "missing encounter",
			cmd:     GenerateInvoiceCmd{LineItems: []InvoiceLineItem{{Description: "x", AmountCents: 1}}, PolicyId: "policy-1"},
			wantErr: ErrMissingEncounter,
		},
		{
			name:    "missing line items",
			cmd:     GenerateInvoiceCmd{EncounterId: "encounter-1", PolicyId: "policy-1"},
			wantErr: ErrMissingLineItems,
		},
		{
			name:    "missing policy",
			cmd:     GenerateInvoiceCmd{EncounterId: "encounter-1", LineItems: []InvoiceLineItem{{Description: "x", AmountCents: 1}}},
			wantErr: ErrMissingPolicy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := validInvoiceAggregate()
			events, err := agg.Execute(tt.cmd)
			assertInvoiceRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestGenerateInvoiceRejectsInvariantViolations(t *testing.T) {
	for _, tt := range invoiceInvariantCases() {
		t.Run(tt.name, func(t *testing.T) {
			agg := validInvoiceAggregate()
			tt.mutate(agg)
			events, err := agg.Execute(validGenerateInvoiceCmd())
			assertInvoiceRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestApplyInsuranceAdjustmentEmitsInvoiceAdjustedEvent(t *testing.T) {
	agg := validInvoiceAggregate()
	cmd := validApplyInsuranceAdjustmentCmd()

	events, err := agg.Execute(cmd)
	if err != nil {
		t.Fatalf("Execute(ApplyInsuranceAdjustmentCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	evt, ok := events[0].(InvoiceAdjustedEvent)
	if !ok {
		t.Fatalf("event type = %T, want InvoiceAdjustedEvent", events[0])
	}
	if evt.Type() != InvoiceAdjustedEventType || evt.Type() != "invoice.adjusted" {
		t.Fatalf("unexpected event type %q", evt.Type())
	}
	if evt.AggregateID() != agg.ID {
		t.Fatalf("event aggregate id = %q, want %q", evt.AggregateID(), agg.ID)
	}
	if evt.InvoiceID != agg.ID {
		t.Fatalf("event invoice id = %q, want %q", evt.InvoiceID, agg.ID)
	}
	if evt.CoverageCents != cmd.Eligibility.CoverageCents || evt.CopayCents != cmd.Eligibility.CopayCents {
		t.Fatalf("event amounts = %#v, want coverage %d copay %d", evt, cmd.Eligibility.CoverageCents, cmd.Eligibility.CopayCents)
	}

	if agg.Status != InvoiceStatusAdjusted {
		t.Fatalf("aggregate status = %q, want %q", agg.Status, InvoiceStatusAdjusted)
	}
	if agg.CoverageCents != cmd.Eligibility.CoverageCents || agg.CopayCents != cmd.Eligibility.CopayCents {
		t.Fatalf("aggregate amounts = coverage %d copay %d", agg.CoverageCents, agg.CopayCents)
	}
	if agg.Version != 1 {
		t.Fatalf("aggregate version = %d, want 1", agg.Version)
	}
	if buffered := agg.Events(); len(buffered) != 1 {
		t.Fatalf("aggregate buffered %d events, want 1", len(buffered))
	}
}

func TestApplyInsuranceAdjustmentAllowsZeroAmounts(t *testing.T) {
	agg := validInvoiceAggregate()
	cmd := ApplyInsuranceAdjustmentCmd{
		InvoiceId:   "invoice-1",
		Eligibility: EligibilityResult{Verified: true, CoverageCents: 0, CopayCents: 0},
	}

	events, err := agg.Execute(cmd)
	if err != nil {
		t.Fatalf("Execute(ApplyInsuranceAdjustmentCmd) with zero amounts returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if agg.Status != InvoiceStatusAdjusted {
		t.Fatalf("aggregate status = %q, want %q", agg.Status, InvoiceStatusAdjusted)
	}
}

func TestApplyInsuranceAdjustmentRejectsMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		cmd     ApplyInsuranceAdjustmentCmd
		wantErr error
	}{
		{
			name:    "missing invoice id",
			cmd:     ApplyInsuranceAdjustmentCmd{Eligibility: EligibilityResult{Verified: true}},
			wantErr: ErrMissingInvoiceID,
		},
		{
			name:    "unverified eligibility",
			cmd:     ApplyInsuranceAdjustmentCmd{InvoiceId: "invoice-1", Eligibility: EligibilityResult{Verified: false}},
			wantErr: ErrUnverifiedEligibility,
		},
		{
			name:    "negative coverage",
			cmd:     ApplyInsuranceAdjustmentCmd{InvoiceId: "invoice-1", Eligibility: EligibilityResult{Verified: true, CoverageCents: -1}},
			wantErr: ErrNegativeAdjustment,
		},
		{
			name:    "negative copay",
			cmd:     ApplyInsuranceAdjustmentCmd{InvoiceId: "invoice-1", Eligibility: EligibilityResult{Verified: true, CopayCents: -1}},
			wantErr: ErrNegativeAdjustment,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := validInvoiceAggregate()
			events, err := agg.Execute(tt.cmd)
			assertInvoiceRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestApplyInsuranceAdjustmentRejectsInvariantViolations(t *testing.T) {
	for _, tt := range invoiceInvariantCases() {
		t.Run(tt.name, func(t *testing.T) {
			agg := validInvoiceAggregate()
			tt.mutate(agg)
			events, err := agg.Execute(validApplyInsuranceAdjustmentCmd())
			assertInvoiceRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestVoidInvoiceEmitsInvoiceVoidedEvent(t *testing.T) {
	agg := validInvoiceAggregate()
	cmd := validVoidInvoiceCmd()

	events, err := agg.Execute(cmd)
	if err != nil {
		t.Fatalf("Execute(VoidInvoiceCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	evt, ok := events[0].(InvoiceVoidedEvent)
	if !ok {
		t.Fatalf("event type = %T, want InvoiceVoidedEvent", events[0])
	}
	if evt.Type() != InvoiceVoidedEventType || evt.Type() != "invoice.voided" {
		t.Fatalf("unexpected event type %q", evt.Type())
	}
	if evt.AggregateID() != agg.ID {
		t.Fatalf("event aggregate id = %q, want %q", evt.AggregateID(), agg.ID)
	}
	if evt.InvoiceID != agg.ID {
		t.Fatalf("event invoice id = %q, want %q", evt.InvoiceID, agg.ID)
	}
	if evt.Reason != cmd.Reason {
		t.Fatalf("event reason = %q, want %q", evt.Reason, cmd.Reason)
	}

	if agg.Status != InvoiceStatusVoided {
		t.Fatalf("aggregate status = %q, want %q", agg.Status, InvoiceStatusVoided)
	}
	if !agg.Voided {
		t.Fatalf("expected voided flag set")
	}
	if agg.Version != 1 {
		t.Fatalf("aggregate version = %d, want 1", agg.Version)
	}
	if buffered := agg.Events(); len(buffered) != 1 {
		t.Fatalf("aggregate buffered %d events, want 1", len(buffered))
	}
}

func TestVoidInvoiceRejectsMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		cmd     VoidInvoiceCmd
		wantErr error
	}{
		{
			name:    "missing invoice id",
			cmd:     VoidInvoiceCmd{Reason: "billed in error"},
			wantErr: ErrMissingInvoiceID,
		},
		{
			name:    "missing reason",
			cmd:     VoidInvoiceCmd{InvoiceId: "invoice-1"},
			wantErr: ErrMissingVoidReason,
		},
		{
			name:    "blank reason",
			cmd:     VoidInvoiceCmd{InvoiceId: "invoice-1", Reason: "   "},
			wantErr: ErrMissingVoidReason,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := validInvoiceAggregate()
			events, err := agg.Execute(tt.cmd)
			assertInvoiceRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestVoidInvoiceRejectsInvariantViolations(t *testing.T) {
	for _, tt := range invoiceInvariantCases() {
		t.Run(tt.name, func(t *testing.T) {
			agg := validInvoiceAggregate()
			tt.mutate(agg)
			events, err := agg.Execute(validVoidInvoiceCmd())
			assertInvoiceRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestInvoiceExecuteRejectsUnknownCommand(t *testing.T) {
	agg := validInvoiceAggregate()

	events, err := agg.Execute(struct{ Unrecognized string }{Unrecognized: "x"})
	if !errors.Is(err, shared.ErrUnknownCommand) {
		t.Fatalf("error = %v, want %v", err, shared.ErrUnknownCommand)
	}
	if events != nil {
		t.Fatalf("expected nil events, got %v", events)
	}
	if len(agg.Events()) != 0 {
		t.Fatalf("expected no buffered events, got %d", len(agg.Events()))
	}
	if agg.Version != 0 {
		t.Fatalf("expected version 0, got %d", agg.Version)
	}
}

func TestInvoiceEventAccessors(t *testing.T) {
	generated := InvoiceGeneratedEvent{InvoiceID: "invoice-9"}
	if generated.Type() != InvoiceGeneratedEventType || generated.AggregateID() != "invoice-9" {
		t.Fatalf("InvoiceGeneratedEvent accessors = %q/%q", generated.Type(), generated.AggregateID())
	}

	adjusted := InvoiceAdjustedEvent{InvoiceID: "invoice-9"}
	if adjusted.Type() != InvoiceAdjustedEventType || adjusted.AggregateID() != "invoice-9" {
		t.Fatalf("InvoiceAdjustedEvent accessors = %q/%q", adjusted.Type(), adjusted.AggregateID())
	}

	voided := InvoiceVoidedEvent{InvoiceID: "invoice-9"}
	if voided.Type() != InvoiceVoidedEventType || voided.AggregateID() != "invoice-9" {
		t.Fatalf("InvoiceVoidedEvent accessors = %q/%q", voided.Type(), voided.AggregateID())
	}
}

func TestInvoiceAggregateRootHelpers(t *testing.T) {
	agg := validInvoiceAggregate()

	if _, err := agg.Execute(validGenerateInvoiceCmd()); err != nil {
		t.Fatalf("Execute(GenerateInvoiceCmd) returned error: %v", err)
	}

	if agg.GetVersion() != 1 {
		t.Fatalf("expected GetVersion 1, got %d", agg.GetVersion())
	}
	if len(agg.Events()) != 1 {
		t.Fatalf("expected 1 buffered event, got %d", len(agg.Events()))
	}

	agg.ClearEvents()
	if len(agg.Events()) != 0 {
		t.Fatalf("expected events cleared, got %d", len(agg.Events()))
	}
	if agg.GetVersion() != 1 {
		t.Fatalf("expected version unchanged after ClearEvents, got %d", agg.GetVersion())
	}
}

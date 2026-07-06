package model

import (
	"errors"
	"testing"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

func validPaymentAggregate() *PaymentAggregate {
	return &PaymentAggregate{ID: "payment-1"}
}

func validInitiatePaymentCmd() InitiatePaymentCmd {
	return InitiatePaymentCmd{
		InvoiceId:    "invoice-1",
		PaymentToken: "tok_123",
		AmountCents:  5000,
	}
}

func validReconcilePaymentCmd() ReconcilePaymentCmd {
	return ReconcilePaymentCmd{
		PaymentId:      "payment-1",
		WebhookPayload: `{"status":"succeeded"}`,
		Signature:      "sig_abc",
	}
}

// paymentInvariantCases flips one shared invariant flag at a time and expects
// the matching sentinel error. These guards run after field-completeness checks
// in every handler.
func paymentInvariantCases() []struct {
	name    string
	mutate  func(*PaymentAggregate)
	wantErr error
} {
	return []struct {
		name    string
		mutate  func(*PaymentAggregate)
		wantErr error
	}{
		{
			name:    "raw card data present",
			mutate:  func(a *PaymentAggregate) { a.RawCardDataPresent = true },
			wantErr: ErrRawCardData,
		},
		{
			name:    "no outstanding balance",
			mutate:  func(a *PaymentAggregate) { a.NoOutstandingBalance = true },
			wantErr: ErrNoOutstandingBalance,
		},
		{
			name:    "webhook not verified",
			mutate:  func(a *PaymentAggregate) { a.WebhookNotVerified = true },
			wantErr: ErrWebhookNotVerified,
		},
	}
}

func assertPaymentRejected(t *testing.T, agg *PaymentAggregate, events []shared.DomainEvent, err error, wantErr error) {
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

func TestInitiatePaymentEmitsPaymentInitiatedEvent(t *testing.T) {
	agg := validPaymentAggregate()
	cmd := validInitiatePaymentCmd()

	events, err := agg.Execute(cmd)
	if err != nil {
		t.Fatalf("Execute(InitiatePaymentCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	evt, ok := events[0].(PaymentInitiatedEvent)
	if !ok {
		t.Fatalf("event type = %T, want PaymentInitiatedEvent", events[0])
	}
	if evt.Type() != PaymentInitiatedEventType || evt.Type() != "payment.initiated" {
		t.Fatalf("unexpected event type %q", evt.Type())
	}
	if evt.AggregateID() != agg.ID {
		t.Fatalf("event aggregate id = %q, want %q", evt.AggregateID(), agg.ID)
	}
	if evt.PaymentID != agg.ID {
		t.Fatalf("event payment id = %q, want %q", evt.PaymentID, agg.ID)
	}
	if evt.InvoiceID != cmd.InvoiceId || evt.PaymentToken != cmd.PaymentToken || evt.AmountCents != cmd.AmountCents {
		t.Fatalf("event fields not copied from command: %#v", evt)
	}

	if agg.Status != PaymentStatusInitiated {
		t.Fatalf("aggregate status = %q, want %q", agg.Status, PaymentStatusInitiated)
	}
	if agg.InvoiceID != cmd.InvoiceId || agg.PaymentToken != cmd.PaymentToken || agg.AmountCents != cmd.AmountCents {
		t.Fatalf("aggregate not scoped to initiation: %#v", agg)
	}
	if agg.Version != 1 {
		t.Fatalf("aggregate version = %d, want 1", agg.Version)
	}
	if buffered := agg.Events(); len(buffered) != 1 {
		t.Fatalf("aggregate buffered %d events, want 1", len(buffered))
	}
}

func TestInitiatePaymentRejectsMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		cmd     InitiatePaymentCmd
		wantErr error
	}{
		{
			name:    "missing invoice",
			cmd:     InitiatePaymentCmd{PaymentToken: "tok_123", AmountCents: 5000},
			wantErr: ErrMissingInvoice,
		},
		{
			name:    "missing payment token",
			cmd:     InitiatePaymentCmd{InvoiceId: "invoice-1", AmountCents: 5000},
			wantErr: ErrMissingPaymentToken,
		},
		{
			name:    "zero amount",
			cmd:     InitiatePaymentCmd{InvoiceId: "invoice-1", PaymentToken: "tok_123", AmountCents: 0},
			wantErr: ErrNonPositiveAmount,
		},
		{
			name:    "negative amount",
			cmd:     InitiatePaymentCmd{InvoiceId: "invoice-1", PaymentToken: "tok_123", AmountCents: -5},
			wantErr: ErrNonPositiveAmount,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := validPaymentAggregate()
			events, err := agg.Execute(tt.cmd)
			assertPaymentRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestInitiatePaymentRejectsInvariantViolations(t *testing.T) {
	for _, tt := range paymentInvariantCases() {
		t.Run(tt.name, func(t *testing.T) {
			agg := validPaymentAggregate()
			tt.mutate(agg)
			events, err := agg.Execute(validInitiatePaymentCmd())
			assertPaymentRejected(t, agg, events, err, tt.wantErr)
			if agg.Status != PaymentStatusNew {
				t.Fatalf("status = %q, want new (unchanged)", agg.Status)
			}
		})
	}
}

func TestReconcilePaymentEmitsPaymentReconciledEvent(t *testing.T) {
	agg := validPaymentAggregate()
	cmd := validReconcilePaymentCmd()

	events, err := agg.Execute(cmd)
	if err != nil {
		t.Fatalf("Execute(ReconcilePaymentCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	evt, ok := events[0].(PaymentReconciledEvent)
	if !ok {
		t.Fatalf("event type = %T, want PaymentReconciledEvent", events[0])
	}
	if evt.Type() != PaymentReconciledEventType || evt.Type() != "payment.reconciled" {
		t.Fatalf("unexpected event type %q", evt.Type())
	}
	if evt.AggregateID() != agg.ID {
		t.Fatalf("event aggregate id = %q, want %q", evt.AggregateID(), agg.ID)
	}
	if evt.PaymentID != agg.ID {
		t.Fatalf("event payment id = %q, want %q", evt.PaymentID, agg.ID)
	}
	if evt.Signature != cmd.Signature {
		t.Fatalf("event signature = %q, want %q", evt.Signature, cmd.Signature)
	}

	if agg.Status != PaymentStatusReconciled {
		t.Fatalf("aggregate status = %q, want %q", agg.Status, PaymentStatusReconciled)
	}
	if agg.Version != 1 {
		t.Fatalf("aggregate version = %d, want 1", agg.Version)
	}
	if buffered := agg.Events(); len(buffered) != 1 {
		t.Fatalf("aggregate buffered %d events, want 1", len(buffered))
	}
}

func TestReconcilePaymentRejectsMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		cmd     ReconcilePaymentCmd
		wantErr error
	}{
		{
			name:    "missing payment id",
			cmd:     ReconcilePaymentCmd{WebhookPayload: "payload", Signature: "sig"},
			wantErr: ErrMissingPayment,
		},
		{
			name:    "missing webhook payload",
			cmd:     ReconcilePaymentCmd{PaymentId: "payment-1", Signature: "sig"},
			wantErr: ErrMissingWebhookPayload,
		},
		{
			name:    "missing signature",
			cmd:     ReconcilePaymentCmd{PaymentId: "payment-1", WebhookPayload: "payload"},
			wantErr: ErrMissingSignature,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := validPaymentAggregate()
			events, err := agg.Execute(tt.cmd)
			assertPaymentRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestReconcilePaymentRejectsInvariantViolations(t *testing.T) {
	for _, tt := range paymentInvariantCases() {
		t.Run(tt.name, func(t *testing.T) {
			agg := validPaymentAggregate()
			tt.mutate(agg)
			events, err := agg.Execute(validReconcilePaymentCmd())
			assertPaymentRejected(t, agg, events, err, tt.wantErr)
			if agg.Status != PaymentStatusNew {
				t.Fatalf("status = %q, want new (unchanged)", agg.Status)
			}
		})
	}
}

func TestPaymentExecuteRejectsUnknownCommand(t *testing.T) {
	agg := validPaymentAggregate()

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

func TestPaymentEventAccessors(t *testing.T) {
	initiated := PaymentInitiatedEvent{PaymentID: "payment-9"}
	if initiated.Type() != PaymentInitiatedEventType || initiated.AggregateID() != "payment-9" {
		t.Fatalf("PaymentInitiatedEvent accessors = %q/%q", initiated.Type(), initiated.AggregateID())
	}

	reconciled := PaymentReconciledEvent{PaymentID: "payment-9"}
	if reconciled.Type() != PaymentReconciledEventType || reconciled.AggregateID() != "payment-9" {
		t.Fatalf("PaymentReconciledEvent accessors = %q/%q", reconciled.Type(), reconciled.AggregateID())
	}
}

func TestPaymentAggregateRootHelpers(t *testing.T) {
	agg := validPaymentAggregate()

	if _, err := agg.Execute(validInitiatePaymentCmd()); err != nil {
		t.Fatalf("Execute(InitiatePaymentCmd) returned error: %v", err)
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

package temporal

import (
	"context"
	"testing"

	"go.temporal.io/sdk/testsuite"

	billmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/billingandinsurance/model"
	billrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/billingandinsurance/repository"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/crypto"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/integration/eligibility"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/integration/payment"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/persistence/mongodb"
)

// stubEligibility is a payer-eligibility Gateway returning a fixed active result.
type stubEligibility struct{ active bool }

func (s stubEligibility) CheckEligibility(_ context.Context, req eligibility.Request) (eligibility.Result, error) {
	return eligibility.Result{
		Active:          s.active,
		PatientID:       req.PatientID,
		PayerIdentifier: req.PayerIdentifier,
		ServiceDate:     req.ServiceDate,
	}, nil
}

// stubPaymentGateway is a PaymentGateway that either confirms the charge or
// declines it, so the saga's success and compensation paths can be driven.
type stubPaymentGateway struct{ err error }

func (s stubPaymentGateway) CreateCharge(_ context.Context, req payment.ChargeRequest) (payment.ChargeResult, error) {
	if s.err != nil {
		return payment.ChargeResult{}, s.err
	}
	return payment.ChargeResult{GatewayChargeID: "ch_1", Status: "succeeded", AmountCents: req.AmountCents}, nil
}

// sagaTestCipher builds a self-contained field cipher for the payment repository
// round-trip. The key is fixed and non-secret; it only exercises the
// encrypt/decrypt path hermetically.
func sagaTestCipher(t *testing.T) *crypto.FieldCipher {
	t.Helper()
	env, err := crypto.NewAESKeyEnvelope("saga-test", make([]byte, crypto.KeySize))
	if err != nil {
		t.Fatalf("NewAESKeyEnvelope: %v", err)
	}
	return crypto.NewFieldCipher(env)
}

// sagaActivitiesFixture wires real saga activities over in-memory repositories so
// the test drives the actual activity code, not mocks. The invoice repository is
// returned so the test can assert the invoice's terminal state.
func sagaActivitiesFixture(t *testing.T, pgw payment.PaymentGateway) (*Activities, billrepo.InvoiceRepository) {
	t.Helper()
	invoices := mongodb.NewInvoiceRepository(mongodb.NewMemStore())
	payments := mongodb.NewPaymentRepository(mongodb.NewMemStore(), sagaTestCipher(t))
	return &Activities{
		Invoices:       invoices,
		Payments:       payments,
		Eligibility:    stubEligibility{active: true},
		PaymentGateway: pgw,
	}, invoices
}

func sagaInput() SagaInput {
	return SagaInput{
		InvoiceID:       "inv-1",
		PaymentID:       "pay-1",
		PatientID:       "pat-1",
		PayerIdentifier: "payer-1",
		MemberID:        "mem-1",
		ServiceDate:     "2026-07-01",
		EncounterID:     "enc-1",
		LineItems:       []LineItem{{Description: "office visit", AmountCents: 12000}},
		PolicyID:        "pol-1",
		CoverageCents:   9000,
		CopayCents:      3000,
		PaymentToken:    "tok_visa",
		AmountCents:     3000,
		Currency:        "usd",
		IdempotencyKey:  "idem-1",
	}
}

// TestSaga_HappyPath verifies eligibility → invoice → payment all succeed and
// leave a generated-then-adjusted invoice and an initiated payment.
func TestSaga_HappyPath(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()

	acts, invoices := sagaActivitiesFixture(t, stubPaymentGateway{})
	env.RegisterActivity(acts)

	env.ExecuteWorkflow(BillingEligibilitySagaWorkflow, sagaInput())

	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow did not complete")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("workflow error: %v", err)
	}

	var result SagaResult
	if err := env.GetWorkflowResult(&result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.InvoiceID != "inv-1" || result.PaymentID != "pay-1" {
		t.Fatalf("unexpected result: %+v", result)
	}

	inv, err := invoices.FindByID(context.Background(), "inv-1")
	if err != nil {
		t.Fatalf("load invoice: %v", err)
	}
	if inv == nil || inv.Status != billmodel.InvoiceStatusAdjusted {
		t.Fatalf("expected adjusted invoice, got %+v", inv)
	}
}

// TestSaga_CompensatesOnPaymentFailure verifies that when the payment step
// fails, the compensation voids the invoice generated earlier, leaving
// consistent state.
func TestSaga_CompensatesOnPaymentFailure(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()

	acts, invoices := sagaActivitiesFixture(t, stubPaymentGateway{err: payment.ErrChargeDeclined})
	env.RegisterActivity(acts)

	env.ExecuteWorkflow(BillingEligibilitySagaWorkflow, sagaInput())

	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow did not complete")
	}
	if env.GetWorkflowError() == nil {
		t.Fatal("expected the saga to fail on the declined payment")
	}

	// The compensation must have voided the invoice generated in step 2.
	inv, err := invoices.FindByID(context.Background(), "inv-1")
	if err != nil {
		t.Fatalf("load invoice: %v", err)
	}
	if inv == nil || !inv.Voided || inv.Status != billmodel.InvoiceStatusVoided {
		t.Fatalf("expected compensated (voided) invoice, got %+v", inv)
	}
}

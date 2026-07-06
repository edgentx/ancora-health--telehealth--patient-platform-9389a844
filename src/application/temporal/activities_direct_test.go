package temporal

import (
	"context"
	"errors"
	"testing"

	adminmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/administrationandanalytics/model"
	billmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/billingandinsurance/model"
	schedmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/scheduling/model"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/integration/eligibility"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/integration/payment"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/persistence/mongodb"
)

var errBoom = errors.New("boom")

// --- test doubles -----------------------------------------------------------

// stubAppointmentRepo returns a fixed appointment (or error) for FindByID.
type stubAppointmentRepo struct {
	appt *schedmodel.AppointmentAggregate
	err  error
}

func (s stubAppointmentRepo) FindByID(_ context.Context, _ string) (*schedmodel.AppointmentAggregate, error) {
	return s.appt, s.err
}
func (s stubAppointmentRepo) Save(_ context.Context, _ *schedmodel.AppointmentAggregate) error {
	return nil
}

// errEligibility is an eligibility gateway that always errors.
type errEligibility struct{ err error }

func (e errEligibility) CheckEligibility(_ context.Context, _ eligibility.Request) (eligibility.Result, error) {
	return eligibility.Result{}, e.err
}

// stubInvoiceRepo is a configurable invoice repository double.
type stubInvoiceRepo struct {
	inv     *billmodel.InvoiceAggregate
	findErr error
	saveErr error
	saved   *billmodel.InvoiceAggregate
}

func (s *stubInvoiceRepo) FindByID(_ context.Context, _ string) (*billmodel.InvoiceAggregate, error) {
	return s.inv, s.findErr
}
func (s *stubInvoiceRepo) Save(_ context.Context, a *billmodel.InvoiceAggregate) error {
	s.saved = a
	return s.saveErr
}

// stubPaymentRepo is a configurable payment repository double.
type stubPaymentRepo struct{ saveErr error }

func (s stubPaymentRepo) FindByID(_ context.Context, _ string) (*billmodel.PaymentAggregate, error) {
	return nil, nil
}
func (s stubPaymentRepo) Save(_ context.Context, _ *billmodel.PaymentAggregate) error {
	return s.saveErr
}

// stubDashboardRepo records the saved dashboard or returns a save error.
type stubDashboardRepo struct {
	saveErr error
	saved   *adminmodel.AnalyticsDashboardAggregate
}

func (s *stubDashboardRepo) FindByID(_ context.Context, _ string) (*adminmodel.AnalyticsDashboardAggregate, error) {
	return nil, nil
}
func (s *stubDashboardRepo) Save(_ context.Context, a *adminmodel.AnalyticsDashboardAggregate) error {
	s.saved = a
	return s.saveErr
}

// --- SendAppointmentReminder ------------------------------------------------

func TestSendAppointmentReminder(t *testing.T) {
	ctx := context.Background()

	t.Run("unconfigured notifier", func(t *testing.T) {
		a := &Activities{}
		if err := a.SendAppointmentReminder(ctx, ReminderActivityInput{}); !errors.Is(err, errUnconfigured) {
			t.Fatalf("want errUnconfigured, got %v", err)
		}
	})

	t.Run("appointment lookup error", func(t *testing.T) {
		a := &Activities{
			Notifier:     &MemNotifier{},
			Appointments: stubAppointmentRepo{err: errBoom},
		}
		if err := a.SendAppointmentReminder(ctx, ReminderActivityInput{AppointmentID: "a1"}); err == nil {
			t.Fatal("expected error loading appointment")
		}
	})

	t.Run("cancelled appointment skips send", func(t *testing.T) {
		notifier := &MemNotifier{}
		a := &Activities{
			Notifier: notifier,
			Appointments: stubAppointmentRepo{appt: &schedmodel.AppointmentAggregate{
				ID:     "a1",
				Status: schedmodel.AppointmentStatusCancelled,
			}},
		}
		if err := a.SendAppointmentReminder(ctx, ReminderActivityInput{AppointmentID: "a1", PatientID: "p1"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := len(notifier.Delivered()); got != 0 {
			t.Fatalf("cancelled appointment should not notify, got %d", got)
		}
	})

	t.Run("active appointment sends", func(t *testing.T) {
		notifier := &MemNotifier{}
		a := &Activities{
			Notifier: notifier,
			Appointments: stubAppointmentRepo{appt: &schedmodel.AppointmentAggregate{
				ID:     "a1",
				Status: schedmodel.AppointmentStatusBooked,
			}},
		}
		if err := a.SendAppointmentReminder(ctx, ReminderActivityInput{
			AppointmentID: "a1", PatientID: "p1", LeadMinutes: 60, TimeSlot: "09:00",
		}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		d := notifier.Delivered()
		if len(d) != 1 {
			t.Fatalf("want 1 delivery, got %d", len(d))
		}
		if d[0].DedupeKey != "reminder:a1:60" {
			t.Fatalf("dedupe key = %q", d[0].DedupeKey)
		}
	})

	t.Run("nil appointment repo still sends", func(t *testing.T) {
		notifier := &MemNotifier{}
		a := &Activities{Notifier: notifier}
		if err := a.SendAppointmentReminder(ctx, ReminderActivityInput{AppointmentID: "a1", PatientID: "p1"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(notifier.Delivered()) != 1 {
			t.Fatal("expected a delivery when no appointment repo is wired")
		}
	})

	t.Run("nil appointment found still sends", func(t *testing.T) {
		notifier := &MemNotifier{}
		a := &Activities{Notifier: notifier, Appointments: stubAppointmentRepo{appt: nil}}
		if err := a.SendAppointmentReminder(ctx, ReminderActivityInput{AppointmentID: "a1", PatientID: "p1"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(notifier.Delivered()) != 1 {
			t.Fatal("expected a delivery when appointment is not found")
		}
	})
}

// --- CheckEligibility -------------------------------------------------------

func TestCheckEligibility(t *testing.T) {
	ctx := context.Background()

	t.Run("unconfigured gateway", func(t *testing.T) {
		a := &Activities{}
		if _, err := a.CheckEligibility(ctx, EligibilityCheckInput{}); !errors.Is(err, errUnconfigured) {
			t.Fatalf("want errUnconfigured, got %v", err)
		}
	})

	t.Run("gateway error", func(t *testing.T) {
		a := &Activities{Eligibility: errEligibility{err: errBoom}}
		if _, err := a.CheckEligibility(ctx, EligibilityCheckInput{}); !errors.Is(err, errBoom) {
			t.Fatalf("want errBoom, got %v", err)
		}
	})

	t.Run("inactive coverage is terminal", func(t *testing.T) {
		a := &Activities{Eligibility: stubEligibility{active: false}}
		_, err := a.CheckEligibility(ctx, EligibilityCheckInput{PayerIdentifier: "payer"})
		if err == nil {
			t.Fatal("expected inactive coverage to error")
		}
	})

	t.Run("active coverage", func(t *testing.T) {
		a := &Activities{Eligibility: stubEligibility{active: true}}
		res, err := a.CheckEligibility(ctx, EligibilityCheckInput{
			PayerIdentifier: "payer-9", ServiceDate: "2026-07-01",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !res.Active || res.PayerIdentifier != "payer-9" || res.ServiceDate != "2026-07-01" {
			t.Fatalf("unexpected result: %+v", res)
		}
	})
}

// --- GenerateInvoice --------------------------------------------------------

func genInput() GenerateInvoiceInput {
	return GenerateInvoiceInput{
		InvoiceID:     "inv-1",
		EncounterID:   "enc-1",
		LineItems:     []LineItem{{Description: "visit", AmountCents: 12000}},
		PolicyID:      "pol-1",
		CoverageCents: 9000,
		CopayCents:    3000,
	}
}

func TestGenerateInvoice(t *testing.T) {
	ctx := context.Background()

	t.Run("unconfigured repo", func(t *testing.T) {
		a := &Activities{}
		if _, err := a.GenerateInvoice(ctx, genInput()); !errors.Is(err, errUnconfigured) {
			t.Fatalf("want errUnconfigured, got %v", err)
		}
	})

	t.Run("persist error", func(t *testing.T) {
		a := &Activities{Invoices: &stubInvoiceRepo{saveErr: errBoom}}
		if _, err := a.GenerateInvoice(ctx, genInput()); !errors.Is(err, errBoom) {
			t.Fatalf("want errBoom, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		repo := &stubInvoiceRepo{}
		a := &Activities{Invoices: repo}
		id, err := a.GenerateInvoice(ctx, genInput())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "inv-1" || repo.saved == nil {
			t.Fatalf("invoice not generated/persisted: id=%q saved=%v", id, repo.saved)
		}
	})
}

// --- CapturePayment ---------------------------------------------------------

func capInput() CapturePaymentInput {
	return CapturePaymentInput{
		PaymentID:    "pay-1",
		InvoiceID:    "inv-1",
		PaymentToken: "tok_visa",
		AmountCents:  3000,
		Currency:     "usd",
	}
}

func TestCapturePayment(t *testing.T) {
	ctx := context.Background()

	t.Run("unconfigured", func(t *testing.T) {
		a := &Activities{}
		if _, err := a.CapturePayment(ctx, capInput()); !errors.Is(err, errUnconfigured) {
			t.Fatalf("want errUnconfigured, got %v", err)
		}
	})

	t.Run("gateway declines", func(t *testing.T) {
		a := &Activities{
			PaymentGateway: stubPaymentGateway{err: payment.ErrChargeDeclined},
			Payments:       stubPaymentRepo{},
		}
		if _, err := a.CapturePayment(ctx, capInput()); !errors.Is(err, payment.ErrChargeDeclined) {
			t.Fatalf("want ErrChargeDeclined, got %v", err)
		}
	})

	t.Run("persist error", func(t *testing.T) {
		a := &Activities{
			PaymentGateway: stubPaymentGateway{},
			Payments:       stubPaymentRepo{saveErr: errBoom},
		}
		if _, err := a.CapturePayment(ctx, capInput()); !errors.Is(err, errBoom) {
			t.Fatalf("want errBoom, got %v", err)
		}
	})

	t.Run("success with default idempotency key", func(t *testing.T) {
		a := &Activities{
			PaymentGateway: stubPaymentGateway{},
			Payments:       stubPaymentRepo{},
		}
		id, err := a.CapturePayment(ctx, capInput()) // empty IdempotencyKey => default
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "pay-1" {
			t.Fatalf("payment id = %q", id)
		}
	})

	t.Run("success with explicit idempotency key", func(t *testing.T) {
		in := capInput()
		in.IdempotencyKey = "idem-9"
		a := &Activities{
			PaymentGateway: stubPaymentGateway{},
			Payments:       stubPaymentRepo{},
		}
		if _, err := a.CapturePayment(ctx, in); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

// --- VoidInvoice ------------------------------------------------------------

// newGeneratedInvoice builds a persisted, generated (non-voided) invoice using
// the real activity path, so VoidInvoice can be exercised against valid state.
func newGeneratedInvoice(t *testing.T) (*Activities, billmodel.InvoiceStatus) {
	t.Helper()
	repo := mongodb.NewInvoiceRepository(mongodb.NewMemStore())
	a := &Activities{Invoices: repo}
	if _, err := a.GenerateInvoice(context.Background(), genInput()); err != nil {
		t.Fatalf("seed invoice: %v", err)
	}
	inv, err := repo.FindByID(context.Background(), "inv-1")
	if err != nil || inv == nil {
		t.Fatalf("load seeded invoice: %v", err)
	}
	return a, inv.Status
}

func TestVoidInvoice(t *testing.T) {
	ctx := context.Background()

	t.Run("unconfigured", func(t *testing.T) {
		a := &Activities{}
		if err := a.VoidInvoice(ctx, VoidInvoiceInput{InvoiceID: "inv-1"}); !errors.Is(err, errUnconfigured) {
			t.Fatalf("want errUnconfigured, got %v", err)
		}
	})

	t.Run("load error", func(t *testing.T) {
		a := &Activities{Invoices: &stubInvoiceRepo{findErr: errBoom}}
		if err := a.VoidInvoice(ctx, VoidInvoiceInput{InvoiceID: "inv-1"}); !errors.Is(err, errBoom) {
			t.Fatalf("want errBoom, got %v", err)
		}
	})

	t.Run("missing invoice is a no-op", func(t *testing.T) {
		a := &Activities{Invoices: &stubInvoiceRepo{inv: nil}}
		if err := a.VoidInvoice(ctx, VoidInvoiceInput{InvoiceID: "inv-1"}); err != nil {
			t.Fatalf("expected no-op, got %v", err)
		}
	})

	t.Run("already voided is a no-op", func(t *testing.T) {
		a := &Activities{Invoices: &stubInvoiceRepo{inv: &billmodel.InvoiceAggregate{ID: "inv-1", Voided: true}}}
		if err := a.VoidInvoice(ctx, VoidInvoiceInput{InvoiceID: "inv-1"}); err != nil {
			t.Fatalf("expected no-op, got %v", err)
		}
	})

	t.Run("success with default reason", func(t *testing.T) {
		a, status := newGeneratedInvoice(t)
		if status == billmodel.InvoiceStatusVoided {
			t.Fatal("seeded invoice should not be voided")
		}
		// Empty Reason exercises the default-reason branch.
		if err := a.VoidInvoice(ctx, VoidInvoiceInput{InvoiceID: "inv-1"}); err != nil {
			t.Fatalf("void: %v", err)
		}
		inv, _ := a.Invoices.FindByID(ctx, "inv-1")
		if inv == nil || !inv.Voided {
			t.Fatalf("expected voided invoice, got %+v", inv)
		}
		// A second void is a no-op via the already-voided guard.
		if err := a.VoidInvoice(ctx, VoidInvoiceInput{InvoiceID: "inv-1"}); err != nil {
			t.Fatalf("second void: %v", err)
		}
	})
}

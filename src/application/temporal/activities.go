package temporal

import (
	"context"
	"errors"
	"fmt"
	"time"

	sdktemporal "go.temporal.io/sdk/temporal"

	adminmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/administrationandanalytics/model"
	adminrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/administrationandanalytics/repository"
	billmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/billingandinsurance/model"
	billrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/billingandinsurance/repository"
	schedmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/scheduling/model"
	schedrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/scheduling/repository"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/integration/eligibility"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/integration/payment"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/persistence/mongodb"
)

// errUnconfigured marks an activity whose backing port was not wired into the
// worker. A partially-wired worker fails the affected activity cleanly instead
// of panicking on a nil dependency.
var errUnconfigured = errors.New("temporal: activity dependency not configured")

// errNotEligible is the type string of the non-retryable error the eligibility
// activity raises when a payer reports coverage is inactive. It is terminal:
// re-checking will not make an inactive policy active, so the saga fails fast.
const errNotEligible = "NotEligible"

// Activities is the receiver whose exported methods are the worker's registered
// activities. Every method is a side-effecting adapter call kept out of the
// deterministic workflow layer: repository reads/writes (S-69/S-70), external
// integrations (S-72), notification fan-out, and analytics persistence.
//
// Each port is optional so a worker can be wired for a subset of workflows; an
// activity whose port is nil returns errUnconfigured rather than panicking.
type Activities struct {
	// Appointments loads appointment aggregates for reminder gating.
	Appointments schedrepo.AppointmentRepository
	// Invoices persists invoice aggregates through the saga and its compensation.
	Invoices billrepo.InvoiceRepository
	// Payments persists payment aggregates captured by the saga.
	Payments billrepo.PaymentRepository
	// Dashboards persists the analytics dashboard aggregate a rollup emits.
	Dashboards adminrepo.AnalyticsDashboardRepository
	// Eligibility is the payer eligibility adapter (S-72).
	Eligibility eligibility.Gateway
	// Payments gateway creates tokenized charges (S-72).
	PaymentGateway payment.PaymentGateway
	// Facts supplies the scheduling/billing facts the rollups aggregate over.
	Facts mongodb.FactSource
	// Rollups persists computed rollup metrics for the AnalyticsDashboard.
	Rollups RollupStore
	// Notifier delivers reminders and results-ready events through the realtime
	// gateway.
	Notifier Notifier
}

// ReminderActivityInput is the payload for SendAppointmentReminder.
type ReminderActivityInput struct {
	AppointmentID string
	PatientID     string
	// LeadMinutes is the configured lead time this reminder fires at, before the
	// appointment. It scopes the idempotency key so each lead time is delivered
	// at most once.
	LeadMinutes int
	// TimeSlot is the appointment's slot, echoed into the reminder body.
	TimeSlot string
}

// SendAppointmentReminder delivers one appointment reminder, idempotently. It
// skips the send when the appointment is no longer active (cancelled), so a
// reminder is never delivered for a slot the patient already dropped. Delivery
// is deduplicated on (appointment, lead time) so a retried attempt does not
// double-notify.
func (a *Activities) SendAppointmentReminder(ctx context.Context, in ReminderActivityInput) error {
	if a.Notifier == nil {
		return fmt.Errorf("%w: notifier", errUnconfigured)
	}

	// Gate on live appointment state when a repository is wired: a cancelled
	// appointment must not be reminded. A worker wired without the appointment
	// repository skips the gate and always sends.
	if a.Appointments != nil {
		appt, err := a.Appointments.FindByID(ctx, in.AppointmentID)
		if err != nil {
			return fmt.Errorf("reminder: load appointment: %w", err)
		}
		if appt != nil && appt.Status == schedmodel.AppointmentStatusCancelled {
			return nil
		}
	}

	return a.Notifier.Notify(ctx, Notification{
		Kind:        "appointment_reminder",
		RecipientID: in.PatientID,
		Subject:     "Upcoming appointment reminder",
		Body:        fmt.Sprintf("You have an appointment at %s.", in.TimeSlot),
		DedupeKey:   fmt.Sprintf("reminder:%s:%d", in.AppointmentID, in.LeadMinutes),
	})
}

// EligibilityCheckInput is the payload for CheckEligibility.
type EligibilityCheckInput struct {
	PatientID       string
	PayerIdentifier string
	MemberID        string
	// ServiceDate is the RFC-3339 date coverage is verified for.
	ServiceDate string
}

// EligibilityCheckResult is the saga-facing outcome of an eligibility check.
type EligibilityCheckResult struct {
	Active          bool
	PayerIdentifier string
	ServiceDate     string
}

// CheckEligibility confirms active payer coverage for the service date. Inactive
// coverage is a terminal, non-retryable failure — re-checking will not make an
// inactive policy active — so the saga fails fast with nothing yet to
// compensate.
func (a *Activities) CheckEligibility(ctx context.Context, in EligibilityCheckInput) (EligibilityCheckResult, error) {
	if a.Eligibility == nil {
		return EligibilityCheckResult{}, fmt.Errorf("%w: eligibility gateway", errUnconfigured)
	}
	res, err := a.Eligibility.CheckEligibility(ctx, eligibility.Request{
		PatientID:       in.PatientID,
		PayerIdentifier: in.PayerIdentifier,
		MemberID:        in.MemberID,
		ServiceDate:     in.ServiceDate,
	})
	if err != nil {
		return EligibilityCheckResult{}, err
	}
	if !res.Active {
		return EligibilityCheckResult{}, sdktemporal.NewNonRetryableApplicationError(
			"payer reports coverage inactive", errNotEligible, nil)
	}
	return EligibilityCheckResult{
		Active:          true,
		PayerIdentifier: res.PayerIdentifier,
		ServiceDate:     res.ServiceDate,
	}, nil
}

// LineItem is a serializable invoice line the saga carries; it maps onto the
// billing model's InvoiceLineItem inside the activity.
type LineItem struct {
	Description string
	AmountCents int64
}

// GenerateInvoiceInput is the payload for GenerateInvoice.
type GenerateInvoiceInput struct {
	InvoiceID   string
	EncounterID string
	LineItems   []LineItem
	PolicyID    string
	// CoverageCents and CopayCents are the verified benefit amounts applied as
	// the insurance adjustment once eligibility has confirmed active coverage.
	CoverageCents int64
	CopayCents    int64
}

// GenerateInvoice creates the invoice from the encounter's charges and applies
// the verified insurance adjustment, persisting a consistent generated-then-
// adjusted invoice. It is the saga step whose failure the compensation voids.
func (a *Activities) GenerateInvoice(ctx context.Context, in GenerateInvoiceInput) (string, error) {
	if a.Invoices == nil {
		return "", fmt.Errorf("%w: invoice repository", errUnconfigured)
	}

	items := make([]billmodel.InvoiceLineItem, 0, len(in.LineItems))
	for _, li := range in.LineItems {
		items = append(items, billmodel.InvoiceLineItem{Description: li.Description, AmountCents: li.AmountCents})
	}

	inv := &billmodel.InvoiceAggregate{ID: in.InvoiceID}
	if _, err := inv.Execute(billmodel.GenerateInvoiceCmd{
		EncounterId: in.EncounterID,
		LineItems:   items,
		PolicyId:    in.PolicyID,
	}); err != nil {
		return "", fmt.Errorf("saga: generate invoice: %w", err)
	}
	if _, err := inv.Execute(billmodel.ApplyInsuranceAdjustmentCmd{
		InvoiceId: in.InvoiceID,
		Eligibility: billmodel.EligibilityResult{
			Verified:      true,
			CoverageCents: in.CoverageCents,
			CopayCents:    in.CopayCents,
		},
	}); err != nil {
		return "", fmt.Errorf("saga: apply adjustment: %w", err)
	}
	if err := a.Invoices.Save(ctx, inv); err != nil {
		return "", fmt.Errorf("saga: persist invoice: %w", err)
	}
	return in.InvoiceID, nil
}

// CapturePaymentInput is the payload for CapturePayment.
type CapturePaymentInput struct {
	PaymentID string
	InvoiceID string
	// PaymentToken is the gateway token standing in for the patient's card; raw
	// card data never reaches the saga.
	PaymentToken string
	AmountCents  int64
	Currency     string
	// IdempotencyKey is forwarded to the gateway so a retried charge is not
	// double-billed.
	IdempotencyKey string
}

// CapturePayment creates the tokenized charge against the gateway and records
// the initiated payment. A declined charge or unreachable gateway surfaces as an
// error the saga compensates on. The gateway idempotency key makes a retried
// attempt safe from double-billing.
func (a *Activities) CapturePayment(ctx context.Context, in CapturePaymentInput) (string, error) {
	if a.PaymentGateway == nil || a.Payments == nil {
		return "", fmt.Errorf("%w: payment gateway/repository", errUnconfigured)
	}

	idem := in.IdempotencyKey
	if idem == "" {
		idem = "pay:" + in.PaymentID
	}
	if _, err := a.PaymentGateway.CreateCharge(ctx, payment.ChargeRequest{
		IdempotencyKey: idem,
		InvoiceID:      in.InvoiceID,
		PaymentToken:   in.PaymentToken,
		AmountCents:    in.AmountCents,
		Currency:       in.Currency,
	}); err != nil {
		return "", fmt.Errorf("saga: create charge: %w", err)
	}

	pay := &billmodel.PaymentAggregate{ID: in.PaymentID}
	if _, err := pay.Execute(billmodel.InitiatePaymentCmd{
		InvoiceId:    in.InvoiceID,
		PaymentToken: in.PaymentToken,
		AmountCents:  in.AmountCents,
	}); err != nil {
		return "", fmt.Errorf("saga: initiate payment: %w", err)
	}
	if err := a.Payments.Save(ctx, pay); err != nil {
		return "", fmt.Errorf("saga: persist payment: %w", err)
	}
	return in.PaymentID, nil
}

// VoidInvoiceInput is the payload for the VoidInvoice compensation.
type VoidInvoiceInput struct {
	InvoiceID string
	Reason    string
}

// VoidInvoice is the saga's compensating action: it voids the invoice generated
// earlier in the saga so a failed payment leaves no dangling billable invoice.
// A missing invoice is treated as already-compensated (no-op), keeping the
// compensation idempotent.
func (a *Activities) VoidInvoice(ctx context.Context, in VoidInvoiceInput) error {
	if a.Invoices == nil {
		return fmt.Errorf("%w: invoice repository", errUnconfigured)
	}
	inv, err := a.Invoices.FindByID(ctx, in.InvoiceID)
	if err != nil {
		return fmt.Errorf("compensation: load invoice: %w", err)
	}
	if inv == nil {
		return nil
	}
	if inv.Voided {
		return nil
	}
	reason := in.Reason
	if reason == "" {
		reason = "saga compensation"
	}
	if _, err := inv.Execute(billmodel.VoidInvoiceCmd{InvoiceId: in.InvoiceID, Reason: reason}); err != nil {
		return fmt.Errorf("compensation: void invoice: %w", err)
	}
	if err := a.Invoices.Save(ctx, inv); err != nil {
		return fmt.Errorf("compensation: persist voided invoice: %w", err)
	}
	return nil
}

// ResultsReadyInput is the payload for NotifyResultsReady.
type ResultsReadyInput struct {
	LabOrderID string
	PatientID  string
	EncounterID string
}

// NotifyResultsReady delivers a results-ready notification to the patient,
// idempotently on the lab order so a retried delivery does not double-notify.
func (a *Activities) NotifyResultsReady(ctx context.Context, in ResultsReadyInput) error {
	if a.Notifier == nil {
		return fmt.Errorf("%w: notifier", errUnconfigured)
	}
	return a.Notifier.Notify(ctx, Notification{
		Kind:        "results_ready",
		RecipientID: in.PatientID,
		Subject:     "Your results are ready",
		Body:        "New results are available to review with your care team.",
		DedupeKey:   "results_ready:" + in.LabOrderID,
	})
}

// RollupActivityInput is the payload for ComputeRollup.
type RollupActivityInput struct {
	DashboardID string
	ClinicID    string
	// RangeStart and RangeEnd are inclusive RFC-3339 dates (2006-01-02).
	RangeStart string
	RangeEnd   string
	// MetricType labels the rollup on the dashboard aggregate's event.
	MetricType string
}

// ComputeRollup computes utilization, no-show rate, and revenue for a clinic
// over a reporting window from the scheduling/billing facts, enforces the
// dashboard's rollup invariants through the AnalyticsDashboard aggregate, and
// persists both the aggregate and the computed metrics for the dashboard.
func (a *Activities) ComputeRollup(ctx context.Context, in RollupActivityInput) (RollupMetrics, error) {
	if a.Facts == nil || a.Rollups == nil {
		return RollupMetrics{}, fmt.Errorf("%w: fact source/rollup store", errUnconfigured)
	}

	from, to, err := parseWindow(in.RangeStart, in.RangeEnd)
	if err != nil {
		return RollupMetrics{}, err
	}

	appts, err := a.Facts.AppointmentFacts(ctx, in.ClinicID, from, to)
	if err != nil {
		return RollupMetrics{}, fmt.Errorf("rollup: appointment facts: %w", err)
	}
	revenue, err := a.Facts.RevenueFacts(ctx, in.ClinicID, from, to)
	if err != nil {
		return RollupMetrics{}, fmt.Errorf("rollup: revenue facts: %w", err)
	}

	metrics := computeRollupMetrics(appts, revenue)
	metrics.DashboardID = in.DashboardID
	metrics.ClinicID = in.ClinicID
	metrics.RangeStart = in.RangeStart
	metrics.RangeEnd = in.RangeEnd
	metrics.ComputedAt = nowUTC()

	// Enforce the dashboard's rollup invariants (scope, no-PHI, reproducibility)
	// through the aggregate and persist the emitted event when a dashboard
	// repository is wired.
	if a.Dashboards != nil {
		dash := &adminmodel.AnalyticsDashboardAggregate{ID: in.DashboardID}
		if _, err := dash.Execute(adminmodel.ComputeRollupCmd{
			ClinicId:   in.ClinicID,
			DateRange:  adminmodel.DateRange{Start: in.RangeStart, End: in.RangeEnd},
			MetricType: in.MetricType,
		}); err != nil {
			return RollupMetrics{}, fmt.Errorf("rollup: compute command: %w", err)
		}
		if err := a.Dashboards.Save(ctx, dash); err != nil {
			return RollupMetrics{}, fmt.Errorf("rollup: persist dashboard: %w", err)
		}
	}

	if err := a.Rollups.SaveRollup(ctx, metrics); err != nil {
		return RollupMetrics{}, fmt.Errorf("rollup: persist metrics: %w", err)
	}
	return metrics, nil
}

// parseWindow turns the inclusive RFC-3339 date bounds into the [from, to)
// window the fact source expects, treating RangeEnd as inclusive by extending
// the exclusive upper bound to the end of that day.
func parseWindow(start, end string) (time.Time, time.Time, error) {
	const dateLayout = "2006-01-02"
	from, err := time.Parse(dateLayout, start)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("rollup: parse range start %q: %w", start, err)
	}
	endDay, err := time.Parse(dateLayout, end)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("rollup: parse range end %q: %w", end, err)
	}
	// [from, to): make the last day inclusive by moving the exclusive bound to
	// the following midnight.
	return from, endDay.AddDate(0, 0, 1), nil
}

// nowUTC is the wall-clock stamp for a computed rollup. It is only ever called
// from within an activity (never a workflow), so using the real clock does not
// break determinism.
func nowUTC() time.Time { return time.Now().UTC() }

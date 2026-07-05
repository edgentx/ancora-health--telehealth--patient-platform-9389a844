package temporal

import (
	"time"

	sdktemporal "go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// SagaInput drives the billing/eligibility saga end to end: the identities to
// mint, the eligibility check inputs, the invoice charges and benefit amounts,
// and the tokenized payment to capture.
type SagaInput struct {
	// InvoiceID and PaymentID are the aggregate identities the saga creates.
	InvoiceID string
	PaymentID string

	// Eligibility inputs.
	PatientID       string
	PayerIdentifier string
	MemberID        string
	ServiceDate     string

	// Invoice inputs.
	EncounterID   string
	LineItems     []LineItem
	PolicyID      string
	CoverageCents int64
	CopayCents    int64

	// Payment inputs.
	PaymentToken   string
	AmountCents    int64
	Currency       string
	IdempotencyKey string
}

// SagaResult is the consistent terminal state of a successful saga.
type SagaResult struct {
	InvoiceID       string
	PaymentID       string
	PayerIdentifier string
}

// sagaActivities is the nil-receiver name handle for the saga's activities (see
// reminderActivities).
var sagaActivities *Activities

// BillingEligibilitySagaWorkflow runs the billing/eligibility saga:
// eligibility → invoice → payment, with a compensation stack that unwinds
// completed steps in reverse on any later failure. If payment fails after the
// invoice was generated, the invoice is voided, leaving consistent
// Invoice/Payment state (a voided invoice and no recorded payment) rather than a
// dangling billable invoice.
//
// Compensations run on a disconnected context so an inbound workflow
// cancellation cannot abort the unwind — a half-applied saga must always finish
// compensating.
func BillingEligibilitySagaWorkflow(ctx workflow.Context, in SagaInput) (SagaResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: defaultActivityTimeout,
		RetryPolicy: &sdktemporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    defaultMaxAttempts,
			// An inactive-coverage result is terminal; do not burn retries on it.
			NonRetryableErrorTypes: []string{errNotEligible},
		},
	})

	// compensations is the unwind stack: each successful, reversible step pushes
	// its compensating action, run in reverse if a later step fails.
	var compensations []func(workflow.Context)
	compensate := func() {
		dctx, _ := workflow.NewDisconnectedContext(ctx)
		for i := len(compensations) - 1; i >= 0; i-- {
			compensations[i](dctx)
		}
	}

	// Step 1 — eligibility. Nothing is persisted yet, so a failure here needs no
	// compensation.
	var elig EligibilityCheckResult
	if err := workflow.ExecuteActivity(ctx, sagaActivities.CheckEligibility, EligibilityCheckInput{
		PatientID:       in.PatientID,
		PayerIdentifier: in.PayerIdentifier,
		MemberID:        in.MemberID,
		ServiceDate:     in.ServiceDate,
	}).Get(ctx, &elig); err != nil {
		return SagaResult{}, err
	}

	// Step 2 — invoice. On success, register the void compensation.
	var invoiceID string
	if err := workflow.ExecuteActivity(ctx, sagaActivities.GenerateInvoice, GenerateInvoiceInput{
		InvoiceID:     in.InvoiceID,
		EncounterID:   in.EncounterID,
		LineItems:     in.LineItems,
		PolicyID:      in.PolicyID,
		CoverageCents: in.CoverageCents,
		CopayCents:    in.CopayCents,
	}).Get(ctx, &invoiceID); err != nil {
		return SagaResult{}, err
	}
	compensations = append(compensations, func(c workflow.Context) {
		_ = workflow.ExecuteActivity(c, sagaActivities.VoidInvoice, VoidInvoiceInput{
			InvoiceID: invoiceID,
			Reason:    "billing saga compensation: payment step failed",
		}).Get(c, nil)
	})

	// Step 3 — payment. A declined charge or unreachable gateway triggers the
	// compensation stack, voiding the invoice generated in step 2.
	var paymentID string
	if err := workflow.ExecuteActivity(ctx, sagaActivities.CapturePayment, CapturePaymentInput{
		PaymentID:      in.PaymentID,
		InvoiceID:      invoiceID,
		PaymentToken:   in.PaymentToken,
		AmountCents:    in.AmountCents,
		Currency:       in.Currency,
		IdempotencyKey: in.IdempotencyKey,
	}).Get(ctx, &paymentID); err != nil {
		compensate()
		return SagaResult{}, err
	}

	return SagaResult{
		InvoiceID:       invoiceID,
		PaymentID:       paymentID,
		PayerIdentifier: elig.PayerIdentifier,
	}, nil
}

package rest

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	billingmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/billingandinsurance/model"
	billingrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/billingandinsurance/repository"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/integration/payment"
)

// billingFlowAPI adapts the invoice-and-payment lifecycle to HTTP: generating an
// invoice from a completed encounter, capturing a tokenized payment against it,
// and reconciling that payment from an HMAC-signed gateway webhook. Each
// mutation records a compliance entry, so the money movement leaves an audit
// trail; the webhook route is the inbound edge the gateway confirms a charge on.
type billingFlowAPI struct {
	invoices billingrepo.InvoiceRepository
	payments billingrepo.PaymentRepository
	audit    AuditSink

	// webhookSecret and webhookIdempot configure the inbound payment webhook.
	// When webhookSecret is empty (or payments is nil) the webhook is not
	// mounted, since a payment cannot be reconciled without both.
	webhookSecret  []byte
	webhookIdempot payment.IdempotencyStore
}

func (h billingFlowAPI) mount(r chi.Router) {
	if h.invoices != nil {
		r.Route("/invoices", func(r chi.Router) {
			r.Post("/", h.generateInvoice)
			r.Get("/{id}", h.getInvoice)
			r.Post("/{id}/adjustment", h.applyAdjustment)
		})
	}
	if h.payments != nil {
		r.Route("/payments", func(r chi.Router) {
			r.Post("/", h.initiatePayment)
			r.Get("/{id}", h.getPayment)
		})
	}
	// The webhook is the gateway's inbound confirmation edge. It is mounted only
	// when a signing secret and a payment repository are both configured.
	if len(h.webhookSecret) > 0 && h.payments != nil {
		idempot := h.webhookIdempot
		if idempot == nil {
			idempot = payment.NewMemoryIdempotencyStore()
		}
		handler := payment.NewWebhookHandler(
			payment.NewVerifier(h.webhookSecret),
			paymentReconciler{payments: h.payments, audit: h.audit},
			idempot,
		)
		r.Method(http.MethodPost, "/payment-webhooks", handler)
	}
}

// --- invoice DTOs + handlers ---

type lineItemDTO struct {
	Description string `json:"description"`
	AmountCents int64  `json:"amountCents"`
}

type generateInvoiceRequest struct {
	EncounterID string        `json:"encounterId"`
	PolicyID    string        `json:"policyId"`
	LineItems   []lineItemDTO `json:"lineItems"`
}

type applyAdjustmentRequest struct {
	Verified      bool  `json:"verified"`
	CoverageCents int64 `json:"coverageCents"`
	CopayCents    int64 `json:"copayCents"`
}

type invoiceResponse struct {
	ID            string        `json:"id"`
	Status        string        `json:"status"`
	EncounterID   string        `json:"encounterId,omitempty"`
	PolicyID      string        `json:"policyId,omitempty"`
	LineItems     []lineItemDTO `json:"lineItems,omitempty"`
	CoverageCents int64         `json:"coverageCents"`
	CopayCents    int64         `json:"copayCents"`
	Version       int           `json:"version"`
}

func toInvoiceResponse(i *billingmodel.InvoiceAggregate) invoiceResponse {
	items := make([]lineItemDTO, 0, len(i.LineItems))
	for _, li := range i.LineItems {
		items = append(items, lineItemDTO{Description: li.Description, AmountCents: li.AmountCents})
	}
	return invoiceResponse{
		ID:            i.ID,
		Status:        string(i.Status),
		EncounterID:   i.EncounterID,
		PolicyID:      i.PolicyID,
		LineItems:     items,
		CoverageCents: i.CoverageCents,
		CopayCents:    i.CopayCents,
		Version:       i.GetVersion(),
	}
}

func (h billingFlowAPI) generateInvoice(w http.ResponseWriter, r *http.Request) {
	var req generateInvoiceRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	for _, v := range [...]struct{ val, field string }{
		{req.EncounterID, "encounterId"},
		{req.PolicyID, "policyId"},
	} {
		if err := requireField(v.val, v.field); err != nil {
			writeError(w, err)
			return
		}
	}
	if len(req.LineItems) == 0 {
		writeError(w, badRequest("lineItems must contain at least one entry"))
		return
	}
	items := make([]billingmodel.InvoiceLineItem, 0, len(req.LineItems))
	for _, li := range req.LineItems {
		items = append(items, billingmodel.InvoiceLineItem{Description: li.Description, AmountCents: li.AmountCents})
	}

	agg := &billingmodel.InvoiceAggregate{ID: newID("inv")}
	cmd := billingmodel.GenerateInvoiceCmd{EncounterId: req.EncounterID, LineItems: items, PolicyId: req.PolicyID}
	if _, err := agg.Execute(cmd); err != nil {
		writeError(w, execErr(err))
		return
	}
	if err := h.invoices.Save(r.Context(), agg); err != nil {
		writeError(w, err)
		return
	}
	if err := recordAudit(r.Context(), h.audit, callerSubject(r), agg.ID, "invoice.generate"); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toInvoiceResponse(agg))
}

func (h billingFlowAPI) getInvoice(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	agg, err := h.invoices.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toInvoiceResponse(agg))
}

func (h billingFlowAPI) applyAdjustment(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var req applyAdjustmentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	agg, err := h.invoices.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	cmd := billingmodel.ApplyInsuranceAdjustmentCmd{
		InvoiceId: id,
		Eligibility: billingmodel.EligibilityResult{
			Verified:      req.Verified,
			CoverageCents: req.CoverageCents,
			CopayCents:    req.CopayCents,
		},
	}
	if _, err := agg.Execute(cmd); err != nil {
		writeError(w, execErr(err))
		return
	}
	if err := h.invoices.Save(r.Context(), agg); err != nil {
		writeError(w, err)
		return
	}
	if err := recordAudit(r.Context(), h.audit, callerSubject(r), agg.ID, "invoice.adjust"); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toInvoiceResponse(agg))
}

// --- payment DTOs + handlers ---

type initiatePaymentRequest struct {
	InvoiceID    string `json:"invoiceId"`
	PaymentToken string `json:"paymentToken"`
	AmountCents  int64  `json:"amountCents"`
}

type paymentResponse struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	InvoiceID   string `json:"invoiceId,omitempty"`
	AmountCents int64  `json:"amountCents"`
	Version     int    `json:"version"`
}

func toPaymentResponse(p *billingmodel.PaymentAggregate) paymentResponse {
	return paymentResponse{
		ID:          p.ID,
		Status:      string(p.Status),
		InvoiceID:   p.InvoiceID,
		AmountCents: p.AmountCents,
		Version:     p.GetVersion(),
	}
}

func (h billingFlowAPI) initiatePayment(w http.ResponseWriter, r *http.Request) {
	var req initiatePaymentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	for _, v := range [...]struct{ val, field string }{
		{req.InvoiceID, "invoiceId"},
		{req.PaymentToken, "paymentToken"},
	} {
		if err := requireField(v.val, v.field); err != nil {
			writeError(w, err)
			return
		}
	}
	if req.AmountCents <= 0 {
		writeError(w, badRequest("amountCents must be positive"))
		return
	}

	agg := &billingmodel.PaymentAggregate{ID: newID("pay")}
	cmd := billingmodel.InitiatePaymentCmd{InvoiceId: req.InvoiceID, PaymentToken: req.PaymentToken, AmountCents: req.AmountCents}
	if _, err := agg.Execute(cmd); err != nil {
		writeError(w, execErr(err))
		return
	}
	if err := h.payments.Save(r.Context(), agg); err != nil {
		writeError(w, err)
		return
	}
	if err := recordAudit(r.Context(), h.audit, callerSubject(r), agg.ID, "payment.initiate"); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toPaymentResponse(agg))
}

func (h billingFlowAPI) getPayment(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	agg, err := h.payments.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toPaymentResponse(agg))
}

// paymentReconciler applies a verified gateway webhook to the payment aggregate.
// It is the PaymentReconciler the webhook handler dispatches through once the
// HMAC signature has been verified: it loads the referenced payment, advances it
// on ReconcilePaymentCmd carrying the verified signature, persists, and audits
// the reconciliation. Keeping it here lets the webhook handler stay free of the
// aggregate and persistence machinery.
type paymentReconciler struct {
	payments billingrepo.PaymentRepository
	audit    AuditSink
}

// reconciledPayload is the placeholder message recorded as the webhook payload
// when the gateway envelope carries no explicit type; the command only requires
// a non-empty payload, and the meaningful provenance is the verified signature.
const reconciledPayload = "charge.reconciled"

func (rc paymentReconciler) Reconcile(ctx context.Context, event payment.PaymentEvent) error {
	agg, err := rc.payments.FindByID(ctx, event.PaymentID)
	if err != nil {
		return err
	}
	payload := event.Type
	if payload == "" {
		payload = reconciledPayload
	}
	cmd := billingmodel.ReconcilePaymentCmd{
		PaymentId:      event.PaymentID,
		WebhookPayload: payload,
		Signature:      event.Signature,
	}
	if _, err := agg.Execute(cmd); err != nil {
		return err
	}
	if err := rc.payments.Save(ctx, agg); err != nil {
		return err
	}
	return recordAudit(ctx, rc.audit, systemActor, agg.ID, "payment.reconcile")
}

// systemActor is the actor stamped on an audit entry produced by an inbound
// gateway webhook, where no interactive caller identity is present.
const systemActor = "payment-gateway"

// Compile-time assertion that the reconciler satisfies the webhook port.
var _ payment.PaymentReconciler = paymentReconciler{}

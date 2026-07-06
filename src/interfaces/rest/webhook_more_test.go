package rest

import (
	"net/http"
	"testing"

	billingmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/billingandinsurance/model"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/integration/payment"
)

var webhookSecret = []byte("whsec-test")

func webhookRouter(t *testing.T, f extFakes) http.Handler {
	t.Helper()
	d := f.deps()
	d.PaymentWebhookSecret = webhookSecret
	return NewRouter(d)
}

func TestWebhook_ReconcileSuccess(t *testing.T) {
	f := newExtFakes()
	f.pays.seed(&billingmodel.PaymentAggregate{ID: "pay-1", Status: billingmodel.PaymentStatusInitiated})
	router := webhookRouter(t, f)

	body := `{"id":"evt-1","type":"charge.succeeded","data":{"paymentId":"pay-1","status":"succeeded"}}`
	rec := doReq(t, router, http.MethodPost, "/api/v1/payment-webhooks", body, signedWebhookHeader(webhookSecret, body))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%q)", rec.Code, rec.Body.String())
	}
	got, _ := f.pays.FindByID(nil, "pay-1")
	if got.Status != billingmodel.PaymentStatusReconciled {
		t.Fatalf("payment status = %q, want reconciled", got.Status)
	}
	if f.audit.last.action != "payment.reconcile" || f.audit.last.actor != systemActor {
		t.Fatalf("audit not recorded from webhook: %+v", f.audit)
	}
}

func TestWebhook_EmptyTypeUsesPlaceholderPayload(t *testing.T) {
	f := newExtFakes()
	f.pays.seed(&billingmodel.PaymentAggregate{ID: "pay-2", Status: billingmodel.PaymentStatusInitiated})
	router := webhookRouter(t, f)

	// No "type" field: the reconciler substitutes the placeholder payload.
	body := `{"id":"evt-2","data":{"paymentId":"pay-2","status":"succeeded"}}`
	rec := doReq(t, router, http.MethodPost, "/api/v1/payment-webhooks", body, signedWebhookHeader(webhookSecret, body))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%q)", rec.Code, rec.Body.String())
	}
}

func TestWebhook_InvalidSignature(t *testing.T) {
	f := newExtFakes()
	f.pays.seed(&billingmodel.PaymentAggregate{ID: "pay-3", Status: billingmodel.PaymentStatusInitiated})
	router := webhookRouter(t, f)

	body := `{"id":"evt-3","type":"charge.succeeded","data":{"paymentId":"pay-3"}}`
	rec := doReq(t, router, http.MethodPost, "/api/v1/payment-webhooks", body, map[string]string{"X-Signature": "deadbeef"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestWebhook_MissingSignature(t *testing.T) {
	f := newExtFakes()
	router := webhookRouter(t, f)
	body := `{"id":"evt-4","data":{"paymentId":"pay-x"}}`
	rec := doReq(t, router, http.MethodPost, "/api/v1/payment-webhooks", body, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestWebhook_MalformedEvent(t *testing.T) {
	f := newExtFakes()
	router := webhookRouter(t, f)
	// A correctly signed body that carries no event id is malformed.
	body := `{"type":"charge.succeeded","data":{"paymentId":"pay-x"}}`
	rec := doReq(t, router, http.MethodPost, "/api/v1/payment-webhooks", body, signedWebhookHeader(webhookSecret, body))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestWebhook_IdempotentRedelivery(t *testing.T) {
	f := newExtFakes()
	f.pays.seed(&billingmodel.PaymentAggregate{ID: "pay-5", Status: billingmodel.PaymentStatusInitiated})
	router := webhookRouter(t, f)

	body := `{"id":"evt-5","type":"charge.succeeded","data":{"paymentId":"pay-5"}}`
	h := signedWebhookHeader(webhookSecret, body)
	if rec := doReq(t, router, http.MethodPost, "/api/v1/payment-webhooks", body, h); rec.Code != http.StatusOK {
		t.Fatalf("first delivery status = %d, want 200", rec.Code)
	}
	// Redelivery of the same event id is a no-op acknowledged with 200.
	if rec := doReq(t, router, http.MethodPost, "/api/v1/payment-webhooks", body, h); rec.Code != http.StatusOK {
		t.Fatalf("redelivery status = %d, want 200", rec.Code)
	}
	if f.audit.calls != 1 {
		t.Fatalf("reconcile ran %d times, want 1 (idempotent)", f.audit.calls)
	}
}

func TestWebhook_ReconcileFailureIs500(t *testing.T) {
	f := newExtFakes()
	// No payment seeded: the reconciler's FindByID fails -> 500.
	router := webhookRouter(t, f)
	body := `{"id":"evt-6","type":"charge.succeeded","data":{"paymentId":"missing"}}`
	rec := doReq(t, router, http.MethodPost, "/api/v1/payment-webhooks", body, signedWebhookHeader(webhookSecret, body))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 (body=%q)", rec.Code, rec.Body.String())
	}
}

func TestWebhook_NotMountedWithoutSecret(t *testing.T) {
	f := newExtFakes()
	// deps() leaves PaymentWebhookSecret empty, so the webhook route is absent.
	router := NewRouter(f.deps())
	body := `{"id":"evt-7","data":{"paymentId":"pay-1"}}`
	rec := doReq(t, router, http.MethodPost, "/api/v1/payment-webhooks", body, signedWebhookHeader(webhookSecret, body))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (webhook unmounted)", rec.Code)
	}
}

func TestWebhook_ExplicitIdempotencyStore(t *testing.T) {
	f := newExtFakes()
	f.pays.seed(&billingmodel.PaymentAggregate{ID: "pay-8", Status: billingmodel.PaymentStatusInitiated})
	d := f.deps()
	d.PaymentWebhookSecret = webhookSecret
	d.PaymentIdempotency = payment.NewMemoryIdempotencyStore()
	router := NewRouter(d)

	body := `{"id":"evt-8","type":"charge.succeeded","data":{"paymentId":"pay-8"}}`
	rec := doReq(t, router, http.MethodPost, "/api/v1/payment-webhooks", body, signedWebhookHeader(webhookSecret, body))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%q)", rec.Code, rec.Body.String())
	}
}

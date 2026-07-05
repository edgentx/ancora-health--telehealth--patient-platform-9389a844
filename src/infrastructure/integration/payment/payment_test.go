package payment

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/integration"
)

func newGateway(t *testing.T, h http.HandlerFunc) (*Adapter, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(h)
	client := integration.NewClient(srv.Client(), integration.Config{BaseURL: srv.URL, MaxAttempts: 2})
	return NewAdapter(client), srv
}

func TestCreateCharge_TokenizedSuccessNeverSendsCardData(t *testing.T) {
	a, srv := newGateway(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		// The wire body must carry the token as `source`, never a PAN field.
		if !strings.Contains(string(body), `"source":"tok_visa"`) {
			t.Errorf("charge body missing tokenized source: %s", body)
		}
		if strings.Contains(string(body), "4242") {
			t.Errorf("raw card data present in charge body")
		}
		if r.Header.Get("Idempotency-Key") != "idem-1" {
			t.Errorf("idempotency key not forwarded")
		}
		_ = json.NewEncoder(w).Encode(chargeResponseBody{ID: "ch_1", Status: "succeeded", Amount: 2500})
	})
	defer srv.Close()

	res, err := a.CreateCharge(context.Background(), ChargeRequest{
		IdempotencyKey: "idem-1",
		InvoiceID:      "inv-1",
		PaymentToken:   "tok_visa",
		AmountCents:    2500,
		Currency:       "usd",
	})
	if err != nil {
		t.Fatalf("CreateCharge: %v", err)
	}
	if res.GatewayChargeID != "ch_1" || res.Status != "succeeded" || res.AmountCents != 2500 {
		t.Fatalf("result = %+v", res)
	}
}

func TestCreateCharge_RejectsRawCardNumber(t *testing.T) {
	var called bool
	a, srv := newGateway(t, func(http.ResponseWriter, *http.Request) { called = true })
	defer srv.Close()

	_, err := a.CreateCharge(context.Background(), ChargeRequest{
		IdempotencyKey: "idem-1",
		InvoiceID:      "inv-1",
		PaymentToken:   "4242 4242 4242 4242", // a PAN, not a token
		AmountCents:    2500,
	})
	if !errors.Is(err, ErrRawCardData) {
		t.Fatalf("err = %v, want ErrRawCardData", err)
	}
	if called {
		t.Fatal("gateway called with raw card data")
	}
}

func TestCreateCharge_DeclineMapsError(t *testing.T) {
	a, srv := newGateway(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
	})
	defer srv.Close()

	_, err := a.CreateCharge(context.Background(), ChargeRequest{
		IdempotencyKey: "idem-1", InvoiceID: "inv-1", PaymentToken: "tok_visa", AmountCents: 100,
	})
	if !errors.Is(err, ErrChargeDeclined) {
		t.Fatalf("err = %v, want ErrChargeDeclined", err)
	}
}

// recordingReconciler captures the events the webhook applies.
type recordingReconciler struct {
	mu     sync.Mutex
	events []PaymentEvent
	fail   bool
}

func (r *recordingReconciler) Reconcile(_ context.Context, e PaymentEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.fail {
		return errors.New("boom")
	}
	r.events = append(r.events, e)
	return nil
}

func signedRequest(t *testing.T, v *Verifier, payload string, withSig bool, sig string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/webhooks/payment", strings.NewReader(payload))
	if withSig {
		if sig == "" {
			sig = v.Sign([]byte(payload))
		}
		req.Header.Set(signatureHeader, sig)
	}
	return req
}

const validEvent = `{"id":"evt-1","type":"charge.succeeded","data":{"paymentId":"pay-1","status":"succeeded"}}`

func TestWebhook_ValidSignatureAppliesEvent(t *testing.T) {
	v := NewVerifier([]byte("whsec"))
	rec := &recordingReconciler{}
	h := NewWebhookHandler(v, rec, NewMemoryIdempotencyStore())

	w := httptest.NewRecorder()
	h.ServeHTTP(w, signedRequest(t, v, validEvent, true, ""))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if len(rec.events) != 1 || rec.events[0].PaymentID != "pay-1" {
		t.Fatalf("events = %+v", rec.events)
	}
}

func TestWebhook_InvalidSignatureRejected(t *testing.T) {
	v := NewVerifier([]byte("whsec"))
	rec := &recordingReconciler{}
	h := NewWebhookHandler(v, rec, NewMemoryIdempotencyStore())

	w := httptest.NewRecorder()
	h.ServeHTTP(w, signedRequest(t, v, validEvent, true, "deadbeef")) // wrong signature

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	if len(rec.events) != 0 {
		t.Fatal("event applied despite invalid signature")
	}
}

func TestWebhook_MissingSignatureRejected(t *testing.T) {
	v := NewVerifier([]byte("whsec"))
	rec := &recordingReconciler{}
	h := NewWebhookHandler(v, rec, NewMemoryIdempotencyStore())

	w := httptest.NewRecorder()
	h.ServeHTTP(w, signedRequest(t, v, validEvent, false, ""))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestWebhook_RedeliveryIsIdempotent(t *testing.T) {
	v := NewVerifier([]byte("whsec"))
	rec := &recordingReconciler{}
	store := NewMemoryIdempotencyStore()
	h := NewWebhookHandler(v, rec, store)

	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, signedRequest(t, v, validEvent, true, ""))
		if w.Code != http.StatusOK {
			t.Fatalf("delivery %d: status %d", i, w.Code)
		}
	}
	if len(rec.events) != 1 {
		t.Fatalf("event applied %d times, want exactly 1 (idempotent)", len(rec.events))
	}
}

func TestWebhook_ReconcileFailureIsRetryable(t *testing.T) {
	v := NewVerifier([]byte("whsec"))
	rec := &recordingReconciler{fail: true}
	store := NewMemoryIdempotencyStore()
	h := NewWebhookHandler(v, rec, store)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, signedRequest(t, v, validEvent, true, ""))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 so gateway retries", w.Code)
	}
	// The event id must not have been remembered, so a redelivery re-applies.
	seen, _ := store.Seen(context.Background(), "evt-1")
	if seen {
		t.Fatal("failed event marked seen; redelivery would be dropped")
	}
}

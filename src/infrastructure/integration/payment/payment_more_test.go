package payment

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/integration"
)

func TestCreateCharge_InvalidRequest(t *testing.T) {
	var called bool
	a, srv := newGateway(t, func(http.ResponseWriter, *http.Request) { called = true })
	defer srv.Close()

	cases := []struct {
		name string
		req  ChargeRequest
	}{
		{"missing idempotency key", ChargeRequest{InvoiceID: "inv", PaymentToken: "tok", AmountCents: 100}},
		{"missing invoice", ChargeRequest{IdempotencyKey: "k", PaymentToken: "tok", AmountCents: 100}},
		{"missing token", ChargeRequest{IdempotencyKey: "k", InvoiceID: "inv", AmountCents: 100}},
		{"zero amount", ChargeRequest{IdempotencyKey: "k", InvoiceID: "inv", PaymentToken: "tok", AmountCents: 0}},
		{"negative amount", ChargeRequest{IdempotencyKey: "k", InvoiceID: "inv", PaymentToken: "tok", AmountCents: -5}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := a.CreateCharge(context.Background(), tc.req); !errors.Is(err, ErrInvalidCharge) {
				t.Fatalf("err = %v, want ErrInvalidCharge", err)
			}
		})
	}
	if called {
		t.Fatal("gateway called on invalid charge")
	}
}

func TestCreateCharge_DefaultsCurrencyToUSD(t *testing.T) {
	var gotCurrency string
	a, srv := newGateway(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var b chargeRequestBody
		_ = json.Unmarshal(body, &b)
		gotCurrency = b.Currency
		_ = json.NewEncoder(w).Encode(chargeResponseBody{ID: "ch", Status: "succeeded", Amount: 100})
	})
	defer srv.Close()

	if _, err := a.CreateCharge(context.Background(), ChargeRequest{
		IdempotencyKey: "k", InvoiceID: "inv", PaymentToken: "tok_x", AmountCents: 100,
	}); err != nil {
		t.Fatalf("CreateCharge: %v", err)
	}
	if gotCurrency != "usd" {
		t.Fatalf("currency = %q, want usd default", gotCurrency)
	}
}

func TestCreateCharge_ServerErrorMapsToUnavailable(t *testing.T) {
	a, srv := newGateway(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	})
	defer srv.Close()

	if _, err := a.CreateCharge(context.Background(), ChargeRequest{
		IdempotencyKey: "k", InvoiceID: "inv", PaymentToken: "tok_x", AmountCents: 100,
	}); !errors.Is(err, ErrGatewayUnavailable) {
		t.Fatalf("err = %v, want ErrGatewayUnavailable", err)
	}
}

func TestCreateCharge_TransportErrorMapsToUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	client := integration.NewClient(srv.Client(), integration.Config{BaseURL: srv.URL, MaxAttempts: 1})
	a := NewAdapter(client)
	srv.Close()

	if _, err := a.CreateCharge(context.Background(), ChargeRequest{
		IdempotencyKey: "k", InvoiceID: "inv", PaymentToken: "tok_x", AmountCents: 100,
	}); !errors.Is(err, ErrGatewayUnavailable) {
		t.Fatalf("err = %v, want ErrGatewayUnavailable", err)
	}
}

func TestCreateCharge_MalformedResponseIsDecodeError(t *testing.T) {
	a, srv := newGateway(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	})
	defer srv.Close()

	_, err := a.CreateCharge(context.Background(), ChargeRequest{
		IdempotencyKey: "k", InvoiceID: "inv", PaymentToken: "tok_x", AmountCents: 100,
	})
	if err == nil {
		t.Fatal("expected decode error")
	}
	if errors.Is(err, ErrChargeDeclined) || errors.Is(err, ErrGatewayUnavailable) {
		t.Fatalf("decode error mis-mapped: %v", err)
	}
}

func TestVerify_RejectsMalformedHexSignature(t *testing.T) {
	v := NewVerifier([]byte("whsec"))
	// An odd-length / non-hex string cannot be decoded, so Verify returns false
	// without a panic.
	if v.Verify([]byte("payload"), "zz") {
		t.Fatal("malformed hex signature must not verify")
	}
	if v.Verify([]byte("payload"), "abc") { // odd length
		t.Fatal("odd-length hex signature must not verify")
	}
}

// failingStore lets a test drive the idempotency-store error paths.
type failingStore struct {
	seenErr     error
	rememberErr error
}

func (s failingStore) Seen(context.Context, string) (bool, error) { return false, s.seenErr }
func (s failingStore) Remember(context.Context, string) error     { return s.rememberErr }

func TestWebhook_MalformedEventRejected(t *testing.T) {
	v := NewVerifier([]byte("whsec"))
	rec := &recordingReconciler{}
	h := NewWebhookHandler(v, rec, NewMemoryIdempotencyStore())

	cases := []struct {
		name    string
		payload string
	}{
		{"invalid json", "{not-json"},
		{"empty event id", `{"id":"","type":"charge.succeeded","data":{}}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			// Sign the (malformed) payload so it passes signature verification and
			// reaches the decode step.
			req := httptest.NewRequest(http.MethodPost, "/webhooks/payment", strings.NewReader(tc.payload))
			req.Header.Set(signatureHeader, v.Sign([]byte(tc.payload)))
			h.ServeHTTP(w, req)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", w.Code)
			}
			if len(rec.events) != 0 {
				t.Fatal("event applied despite malformed payload")
			}
		})
	}
}

// errBody is a request body whose Read always fails.
type errBody struct{}

func (errBody) Read([]byte) (int, error) { return 0, errors.New("read failed") }
func (errBody) Close() error             { return nil }

func TestWebhook_BodyReadErrorIs400(t *testing.T) {
	v := NewVerifier([]byte("whsec"))
	h := NewWebhookHandler(v, &recordingReconciler{}, NewMemoryIdempotencyStore())

	req := httptest.NewRequest(http.MethodPost, "/webhooks/payment", nil)
	req.Body = errBody{}
	req.Header.Set(signatureHeader, "deadbeef")

	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 on unreadable body", w.Code)
	}
}

func TestWebhook_IdempotencySeenErrorIs500(t *testing.T) {
	v := NewVerifier([]byte("whsec"))
	rec := &recordingReconciler{}
	h := NewWebhookHandler(v, rec, failingStore{seenErr: errors.New("store down")})

	w := httptest.NewRecorder()
	h.ServeHTTP(w, signedRequest(t, v, validEvent, true, ""))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	if len(rec.events) != 0 {
		t.Fatal("event applied despite idempotency check failure")
	}
}

func TestWebhook_RememberErrorIs500(t *testing.T) {
	v := NewVerifier([]byte("whsec"))
	rec := &recordingReconciler{}
	h := NewWebhookHandler(v, rec, failingStore{rememberErr: errors.New("persist down")})

	w := httptest.NewRecorder()
	h.ServeHTTP(w, signedRequest(t, v, validEvent, true, ""))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 so the gateway retries", w.Code)
	}
	// The event was reconciled before the persist failure.
	if len(rec.events) != 1 {
		t.Fatalf("events = %d, want 1 (reconcile ran before persist failed)", len(rec.events))
	}
}

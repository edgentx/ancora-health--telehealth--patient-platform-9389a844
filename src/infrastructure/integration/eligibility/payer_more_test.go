package eligibility

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/integration"
)

// failingAudit rejects every outbound-access record.
type failingAudit struct{ err error }

func (f failingAudit) RecordOutboundAccess(context.Context, integration.OutboundAccess) error {
	return f.err
}

func newAdapterWithAudit(t *testing.T, h http.HandlerFunc, audit integration.AuditRecorder) (*Adapter, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(h)
	client := integration.NewClient(srv.Client(), integration.Config{BaseURL: srv.URL, MaxAttempts: 1})
	return NewAdapter(client, audit), srv
}

func TestCheckEligibility_MissingFields(t *testing.T) {
	a, srv := newAdapter(t, func(http.ResponseWriter, *http.Request) {})
	defer srv.Close()

	cases := []Request{
		{PayerIdentifier: "p", MemberID: "m", ServiceDate: "2026-07-05"},
		{PatientID: "pat", MemberID: "m", ServiceDate: "2026-07-05"},
		{PatientID: "pat", PayerIdentifier: "p", ServiceDate: "2026-07-05"},
		{PatientID: "pat", PayerIdentifier: "p", MemberID: "m"},
	}
	for _, r := range cases {
		if _, err := a.CheckEligibility(context.Background(), r); !errors.Is(err, ErrInvalidRequest) {
			t.Fatalf("CheckEligibility(%+v) err = %v, want ErrInvalidRequest", r, err)
		}
	}
}

func TestCheckEligibility_AuditFailureAborts(t *testing.T) {
	var called bool
	a, srv := newAdapterWithAudit(t, func(http.ResponseWriter, *http.Request) { called = true },
		failingAudit{err: errors.New("audit down")})
	defer srv.Close()

	if _, err := a.CheckEligibility(context.Background(), sampleRequest()); err == nil {
		t.Fatal("expected audit failure to abort")
	}
	if called {
		t.Fatal("payer called despite audit failure")
	}
}

func TestCheckEligibility_EmptyPayerFallsBackToRequest(t *testing.T) {
	a, srv := newAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		// Payer echoes no identifier of its own, so the request's should win.
		_ = json.NewEncoder(w).Encode(eligibilityResponse{Active: true})
	})
	defer srv.Close()

	res, err := a.CheckEligibility(context.Background(), sampleRequest())
	if err != nil {
		t.Fatalf("CheckEligibility: %v", err)
	}
	if res.PayerIdentifier != "payer-1" {
		t.Fatalf("payer = %q, want fallback payer-1", res.PayerIdentifier)
	}
}

func TestCheckEligibility_MalformedResponseIsDecodeError(t *testing.T) {
	a, srv := newAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("{bad json"))
	})
	defer srv.Close()

	_, err := a.CheckEligibility(context.Background(), sampleRequest())
	if err == nil {
		t.Fatal("expected decode error")
	}
	if errors.Is(err, ErrPayerRejected) || errors.Is(err, ErrPayerUnavailable) {
		t.Fatalf("decode error mis-mapped: %v", err)
	}
}

func TestCheckEligibility_ServerErrorMapsToUnavailable(t *testing.T) {
	a, srv := newAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	})
	defer srv.Close()

	if _, err := a.CheckEligibility(context.Background(), sampleRequest()); !errors.Is(err, ErrPayerUnavailable) {
		t.Fatalf("err = %v, want ErrPayerUnavailable", err)
	}
}

func TestCheckEligibility_TransportErrorMapsToUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	client := integration.NewClient(srv.Client(), integration.Config{BaseURL: srv.URL, MaxAttempts: 1})
	a := NewAdapter(client, nil)
	srv.Close()

	if _, err := a.CheckEligibility(context.Background(), sampleRequest()); !errors.Is(err, ErrPayerUnavailable) {
		t.Fatalf("err = %v, want ErrPayerUnavailable", err)
	}
}

package pharmacy

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

// spyAudit records the outbound-access entries the adapter emits.
type spyAudit struct {
	entries []integration.OutboundAccess
}

func (s *spyAudit) RecordOutboundAccess(_ context.Context, a integration.OutboundAccess) error {
	s.entries = append(s.entries, a)
	return nil
}

func newAdapter(t *testing.T, h http.HandlerFunc, audit integration.AuditRecorder) (*Adapter, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(h)
	client := integration.NewClient(srv.Client(), integration.Config{BaseURL: srv.URL, MaxAttempts: 2})
	return NewAdapter(client, audit), srv
}

func authedProvider() ProviderContext {
	return ProviderContext{ProviderID: "prov-1", ProviderNPI: "1234567890", Authenticated: true}
}

func sampleOrder() PrescriptionOrder {
	return PrescriptionOrder{
		PrescriptionID: "rx-1",
		PatientID:      "pat-1",
		PharmacyID:     "pharm-1",
		Medication:     "amoxicillin",
		Dosage:         "500mg",
	}
}

func TestSubmit_RejectsUnauthenticatedProviderBeforeCall(t *testing.T) {
	var called bool
	a, srv := newAdapter(t, func(http.ResponseWriter, *http.Request) { called = true }, nil)
	defer srv.Close()

	unauth := authedProvider()
	unauth.Authenticated = false

	_, err := a.Submit(context.Background(), unauth, sampleOrder())
	if !errors.Is(err, ErrUnauthenticatedProvider) {
		t.Fatalf("err = %v, want ErrUnauthenticatedProvider", err)
	}
	if called {
		t.Fatal("gateway was called despite unauthenticated provider")
	}
}

func TestSubmit_SuccessMapsStatusAndAudits(t *testing.T) {
	audit := &spyAudit{}
	a, srv := newAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var msg scriptRequest
		if err := json.Unmarshal(body, &msg); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if msg.MessageType != messageTypeNewRx {
			t.Errorf("messageType = %q, want NewRx", msg.MessageType)
		}
		if r.Header.Get("X-Provider-Id") != "prov-1" {
			t.Errorf("provider header not forwarded")
		}
		_ = json.NewEncoder(w).Encode(scriptResponse{Status: "Accepted", MessageRef: "msg-99"})
	}, audit)
	defer srv.Close()

	res, err := a.Submit(context.Background(), authedProvider(), sampleOrder())
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if res.Status != StatusAccepted || res.GatewayReference != "msg-99" {
		t.Fatalf("result = %+v", res)
	}

	if len(audit.entries) != 1 {
		t.Fatalf("audit entries = %d, want 1", len(audit.entries))
	}
	e := audit.entries[0]
	if e.ResourceRef != "rx-1" || e.Action != "eprescribe.submit" || e.ActorContext != "prov-1" {
		t.Fatalf("audit entry = %+v", e)
	}
	// PHI must never leak into the audit record.
	if strings.Contains(e.ResourceRef, "amoxicillin") {
		t.Fatal("medication PHI leaked into audit resource ref")
	}
}

func TestSubmit_GatewayRejectionMapsError(t *testing.T) {
	a, srv := newAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
	}, nil)
	defer srv.Close()

	_, err := a.Submit(context.Background(), authedProvider(), sampleOrder())
	if !errors.Is(err, ErrGatewayRejected) {
		t.Fatalf("err = %v, want ErrGatewayRejected", err)
	}
}

func TestSubmit_ServerErrorMapsToUnavailable(t *testing.T) {
	a, srv := newAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}, nil)
	defer srv.Close()

	_, err := a.Submit(context.Background(), authedProvider(), sampleOrder())
	if !errors.Is(err, ErrGatewayUnavailable) {
		t.Fatalf("err = %v, want ErrGatewayUnavailable", err)
	}
}

func TestCancel_SendsCancelRx(t *testing.T) {
	a, srv := newAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var msg scriptRequest
		_ = json.Unmarshal(body, &msg)
		if msg.MessageType != messageTypeCancelRx {
			t.Errorf("messageType = %q, want CancelRx", msg.MessageType)
		}
		_ = json.NewEncoder(w).Encode(scriptResponse{Status: "Accepted", MessageRef: "cx-1"})
	}, nil)
	defer srv.Close()

	res, err := a.Cancel(context.Background(), authedProvider(), CancelOrder{PrescriptionID: "rx-1", PharmacyID: "pharm-1"})
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if res.Status != StatusAccepted {
		t.Fatalf("status = %v", res.Status)
	}
}

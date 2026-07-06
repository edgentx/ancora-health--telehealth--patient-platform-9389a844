package pharmacy

import (
	"context"
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

func TestSubmit_IncompleteOrderRejected(t *testing.T) {
	var called bool
	a, srv := newAdapter(t, func(http.ResponseWriter, *http.Request) { called = true }, nil)
	defer srv.Close()

	cases := []struct {
		name  string
		order PrescriptionOrder
	}{
		{"missing prescription id", PrescriptionOrder{PharmacyID: "p", Medication: "m", Dosage: "d"}},
		{"missing pharmacy id", PrescriptionOrder{PrescriptionID: "rx", Medication: "m", Dosage: "d"}},
		{"missing medication", PrescriptionOrder{PrescriptionID: "rx", PharmacyID: "p", Dosage: "d"}},
		{"missing dosage", PrescriptionOrder{PrescriptionID: "rx", PharmacyID: "p", Medication: "m"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := a.Submit(context.Background(), authedProvider(), tc.order)
			if !errors.Is(err, ErrIncompleteOrder) {
				t.Fatalf("err = %v, want ErrIncompleteOrder", err)
			}
		})
	}
	if called {
		t.Fatal("gateway called despite incomplete order")
	}
}

func TestCancel_RejectsUnauthenticatedAndIncomplete(t *testing.T) {
	var called bool
	a, srv := newAdapter(t, func(http.ResponseWriter, *http.Request) { called = true }, nil)
	defer srv.Close()

	unauth := authedProvider()
	unauth.Authenticated = false
	if _, err := a.Cancel(context.Background(), unauth, CancelOrder{PrescriptionID: "rx", PharmacyID: "p"}); !errors.Is(err, ErrUnauthenticatedProvider) {
		t.Fatalf("err = %v, want ErrUnauthenticatedProvider", err)
	}

	cases := []CancelOrder{
		{PharmacyID: "p"},
		{PrescriptionID: "rx"},
	}
	for _, o := range cases {
		if _, err := a.Cancel(context.Background(), authedProvider(), o); !errors.Is(err, ErrIncompleteOrder) {
			t.Fatalf("Cancel(%+v) err = %v, want ErrIncompleteOrder", o, err)
		}
	}
	if called {
		t.Fatal("gateway called despite invalid cancel order")
	}
}

func TestTransmit_AuditFailureAbortsBeforeCall(t *testing.T) {
	var called bool
	a, srv := newAdapter(t, func(http.ResponseWriter, *http.Request) { called = true },
		failingAudit{err: errors.New("audit sink down")})
	defer srv.Close()

	_, err := a.Submit(context.Background(), authedProvider(), sampleOrder())
	if err == nil {
		t.Fatal("expected audit failure to abort submission")
	}
	if called {
		t.Fatal("gateway called despite audit failure")
	}
}

func TestSubmit_MalformedResponseIsDecodeError(t *testing.T) {
	a, srv := newAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not json"))
	}, nil)
	defer srv.Close()

	_, err := a.Submit(context.Background(), authedProvider(), sampleOrder())
	if err == nil {
		t.Fatal("expected decode error on malformed response")
	}
	// A decode error is not one of the transport-mapped sentinels.
	if errors.Is(err, ErrGatewayRejected) || errors.Is(err, ErrGatewayUnavailable) {
		t.Fatalf("decode error mis-mapped: %v", err)
	}
}

func TestSubmit_TransportErrorMapsToUnavailable(t *testing.T) {
	// A closed server produces a connection error (not a status), exercising
	// mapTransmitError's non-StatusError branch.
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	client := integration.NewClient(srv.Client(), integration.Config{BaseURL: srv.URL, MaxAttempts: 1})
	a := NewAdapter(client, nil)
	srv.Close() // close before sending so the connection is refused

	_, err := a.Submit(context.Background(), authedProvider(), sampleOrder())
	if !errors.Is(err, ErrGatewayUnavailable) {
		t.Fatalf("err = %v, want ErrGatewayUnavailable", err)
	}
}

func TestNormalizeStatus(t *testing.T) {
	tests := []struct {
		gateway string
		want    TransmissionStatus
	}{
		{"Accepted", StatusAccepted},
		{"accepted", StatusAccepted},
		{"Success", StatusAccepted},
		{"success", StatusAccepted},
		{"Queued", StatusQueued},
		{"queued", StatusQueued},
		{"Pending", StatusQueued},
		{"pending", StatusQueued},
		{"Rejected", StatusRejected},
		{"", StatusRejected},
		{"anything-unknown", StatusRejected},
	}
	for _, tc := range tests {
		if got := normalizeStatus(tc.gateway); got != tc.want {
			t.Fatalf("normalizeStatus(%q) = %q, want %q", tc.gateway, got, tc.want)
		}
	}
}

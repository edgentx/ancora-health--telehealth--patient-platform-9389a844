package eligibility

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/billingandinsurance/model"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/integration"
)

func newAdapter(t *testing.T, h http.HandlerFunc) (*Adapter, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(h)
	client := integration.NewClient(srv.Client(), integration.Config{BaseURL: srv.URL, MaxAttempts: 2})
	return NewAdapter(client, nil), srv
}

func sampleRequest() Request {
	return Request{PatientID: "pat-1", PayerIdentifier: "payer-1", MemberID: "mem-9", ServiceDate: "2026-07-05"}
}

func TestCheckEligibility_MapsResponseToDomainModel(t *testing.T) {
	a, srv := newAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("member") != "mem-9" {
			t.Errorf("member query = %q", r.URL.Query().Get("member"))
		}
		_ = json.NewEncoder(w).Encode(eligibilityResponse{
			Active:            true,
			Payer:             "payer-1",
			CoverageStartDate: "2026-01-01",
			CoverageEndDate:   "2026-12-31",
		})
	})
	defer srv.Close()

	res, err := a.CheckEligibility(context.Background(), sampleRequest())
	if err != nil {
		t.Fatalf("CheckEligibility: %v", err)
	}
	if !res.Active {
		t.Fatal("expected active coverage")
	}
	want := model.EffectiveDates{Start: "2026-01-01", End: "2026-12-31"}
	if res.EffectiveDates != want {
		t.Fatalf("effective dates = %+v, want %+v", res.EffectiveDates, want)
	}

	// The mapping must feed the domain command cleanly.
	cmd := res.ToRegisterCommand()
	if cmd.PatientId != "pat-1" || cmd.PayerIdentifier != "payer-1" || cmd.EffectiveDates != want {
		t.Fatalf("register command = %+v", cmd)
	}
}

func TestCheckEligibility_ValidatesRequest(t *testing.T) {
	a, srv := newAdapter(t, func(http.ResponseWriter, *http.Request) {})
	defer srv.Close()

	_, err := a.CheckEligibility(context.Background(), Request{PatientID: "pat-1"})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("err = %v, want ErrInvalidRequest", err)
	}
}

func TestCheckEligibility_PayerRejectionMapsError(t *testing.T) {
	a, srv := newAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	defer srv.Close()

	_, err := a.CheckEligibility(context.Background(), sampleRequest())
	if !errors.Is(err, ErrPayerRejected) {
		t.Fatalf("err = %v, want ErrPayerRejected", err)
	}
}

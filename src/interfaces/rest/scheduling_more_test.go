package rest

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	schedmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/scheduling/model"
)

func TestScheduling_HoldSlotValidation(t *testing.T) {
	cases := []struct {
		name, body, wantField string
	}{
		{"missing provider", `{"timeSlot":"s","patientId":"p"}`, "providerId"},
		{"missing timeSlot", `{"providerId":"pr","patientId":"p"}`, "timeSlot"},
		{"missing patient", `{"providerId":"pr","timeSlot":"s"}`, "patientId"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakes()
			router := NewRouter(f.deps())
			rec := doRequest(t, router, http.MethodPost, "/api/v1/appointments", tc.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", rec.Code)
			}
			er := decodeErr(t, rec)
			if er.Code != codeValidation {
				t.Fatalf("code = %q", er.Code)
			}
		})
	}
}

func TestAppointment_BookSuccess(t *testing.T) {
	f := newFakes()
	f.appts.seed(&schedmodel.AppointmentAggregate{ID: "appt-b", Status: schedmodel.AppointmentStatusHeld})
	router := NewRouter(f.deps())

	rec := doRequest(t, router, http.MethodPost, "/api/v1/appointments/appt-b/booking",
		`{"holdToken":"tok","patientId":"pat-1","reason":"checkup"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%q)", rec.Code, rec.Body.String())
	}
	var resp appointmentResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != string(schedmodel.AppointmentStatusBooked) {
		t.Fatalf("status = %q, want booked", resp.Status)
	}
}

func TestAppointment_BookNotFound(t *testing.T) {
	f := newFakes()
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/appointments/missing/booking",
		`{"holdToken":"tok","patientId":"pat-1","reason":"checkup"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestAppointment_BookMissingHoldTokenUnprocessable(t *testing.T) {
	f := newFakes()
	f.appts.seed(&schedmodel.AppointmentAggregate{ID: "appt-h", Status: schedmodel.AppointmentStatusHeld})
	router := NewRouter(f.deps())
	// The handler forwards to the domain without pre-validating the token; the
	// domain rejects the missing token as a rule violation -> 422.
	rec := doRequest(t, router, http.MethodPost, "/api/v1/appointments/appt-h/booking",
		`{"patientId":"pat-1","reason":"checkup"}`)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 (body=%q)", rec.Code, rec.Body.String())
	}
}

func TestAppointment_CancelSuccess(t *testing.T) {
	f := newFakes()
	f.appts.seed(&schedmodel.AppointmentAggregate{ID: "appt-c", Status: schedmodel.AppointmentStatusHeld})
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/appointments/appt-c/cancellation",
		`{"reason":"patient no longer needs it"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%q)", rec.Code, rec.Body.String())
	}
	var resp appointmentResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Status != string(schedmodel.AppointmentStatusCancelled) {
		t.Fatalf("status = %q, want cancelled", resp.Status)
	}
}

func TestAppointment_CancelMissingReasonUnprocessable(t *testing.T) {
	f := newFakes()
	f.appts.seed(&schedmodel.AppointmentAggregate{ID: "appt-cr", Status: schedmodel.AppointmentStatusHeld})
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/appointments/appt-cr/cancellation", `{}`)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 (body=%q)", rec.Code, rec.Body.String())
	}
}

func TestAppointment_CancelMalformedJSON(t *testing.T) {
	f := newFakes()
	f.appts.seed(&schedmodel.AppointmentAggregate{ID: "appt-cm", Status: schedmodel.AppointmentStatusHeld})
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/appointments/appt-cm/cancellation", `{bad`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestAppointment_RescheduleSuccess(t *testing.T) {
	f := newFakes()
	f.appts.seed(&schedmodel.AppointmentAggregate{ID: "appt-r", Status: schedmodel.AppointmentStatusHeld, HeldTimeSlot: "old"})
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/appointments/appt-r/reschedule",
		`{"newTimeSlot":"2026-08-01T10:00Z"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%q)", rec.Code, rec.Body.String())
	}
	var resp appointmentResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.TimeSlot != "2026-08-01T10:00Z" {
		t.Fatalf("timeSlot = %q", resp.TimeSlot)
	}
}

func TestAppointment_RescheduleMissingSlotUnprocessable(t *testing.T) {
	f := newFakes()
	f.appts.seed(&schedmodel.AppointmentAggregate{ID: "appt-rm", Status: schedmodel.AppointmentStatusHeld})
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/appointments/appt-rm/reschedule", `{}`)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 (body=%q)", rec.Code, rec.Body.String())
	}
}

func TestAppointment_RescheduleNotFound(t *testing.T) {
	f := newFakes()
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/appointments/nope/reschedule",
		`{"newTimeSlot":"x"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestAppointment_GetSuccess(t *testing.T) {
	f := newFakes()
	f.appts.seed(&schedmodel.AppointmentAggregate{ID: "appt-g", Status: schedmodel.AppointmentStatusBooked, ScopedProviderID: "prov-1"})
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodGet, "/api/v1/appointments/appt-g", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestAppointment_GetBlankIDValidation(t *testing.T) {
	f := newFakes()
	router := NewRouter(f.deps())
	// A whitespace-only path segment trims to empty -> pathID rejects it as 400.
	rec := doRequest(t, router, http.MethodGet, "/api/v1/appointments/%20", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body=%q)", rec.Code, rec.Body.String())
	}
	er := decodeErr(t, rec)
	if er.Code != codeValidation {
		t.Fatalf("code = %q", er.Code)
	}
}

// --- provider schedule ---

func TestProviderSchedule_PublishSuccess(t *testing.T) {
	f := newFakes()
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/provider-schedules",
		`{"providerId":"prov-1","windows":["2026-07-10T09:00Z/2026-07-10T12:00Z"]}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body=%q)", rec.Code, rec.Body.String())
	}
	var resp providerScheduleResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ProviderID != "prov-1" || len(resp.PublishedWindows) != 1 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestProviderSchedule_PublishValidation(t *testing.T) {
	cases := []struct{ name, body string }{
		{"missing provider", `{"windows":["w"]}`},
		{"empty windows", `{"providerId":"prov-1","windows":[]}`},
		{"malformed", `{bad`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakes()
			router := NewRouter(f.deps())
			rec := doRequest(t, router, http.MethodPost, "/api/v1/provider-schedules", tc.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", rec.Code)
			}
		})
	}
}

func TestProviderSchedule_GetSuccessAndNotFound(t *testing.T) {
	f := newFakes()
	f.scheds.seed(&schedmodel.ProviderScheduleAggregate{ID: "sched-1", ScopedProviderID: "prov-1"})
	router := NewRouter(f.deps())

	if rec := doRequest(t, router, http.MethodGet, "/api/v1/provider-schedules/sched-1", ""); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if rec := doRequest(t, router, http.MethodGet, "/api/v1/provider-schedules/missing", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

// --- reserveSlot booker branch (slotBooker capability) ---

func TestHoldSlot_BookerBranch(t *testing.T) {
	t.Run("success books under lock", func(t *testing.T) {
		f := newFakes()
		d := f.deps()
		d.Appointments = bookingRepo{fakeRepo: f.appts}
		router := NewRouter(d)
		rec := doRequest(t, router, http.MethodPost, "/api/v1/appointments",
			`{"providerId":"prov-1","timeSlot":"slot","patientId":"pat-1"}`)
		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201 (body=%q)", rec.Code, rec.Body.String())
		}
	})

	t.Run("double booked -> 409", func(t *testing.T) {
		f := newFakes()
		d := f.deps()
		d.Appointments = bookingRepo{fakeRepo: f.appts, bookErr: schedmodel.ErrSlotDoubleBooked}
		router := NewRouter(d)
		rec := doRequest(t, router, http.MethodPost, "/api/v1/appointments",
			`{"providerId":"prov-1","timeSlot":"slot","patientId":"pat-1"}`)
		if rec.Code != http.StatusConflict {
			t.Fatalf("status = %d, want 409 (body=%q)", rec.Code, rec.Body.String())
		}
	})

	t.Run("infra error -> 500", func(t *testing.T) {
		f := newFakes()
		d := f.deps()
		d.Appointments = bookingRepo{fakeRepo: f.appts, bookErr: errors.New("redis down")}
		router := NewRouter(d)
		rec := doRequest(t, router, http.MethodPost, "/api/v1/appointments",
			`{"providerId":"prov-1","timeSlot":"slot","patientId":"pat-1"}`)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500 (body=%q)", rec.Code, rec.Body.String())
		}
	})
}

package rest

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	schedmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/scheduling/model"
	schedulingrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/scheduling/repository"
)

// schedulingAPI adapts the scheduling bounded context (appointments and provider
// schedules) to HTTP. It holds only the domain repository ports, so it can be
// exercised end-to-end with in-memory fakes.
type schedulingAPI struct {
	appointments schedulingrepo.AppointmentRepository
	schedules    schedulingrepo.ProviderScheduleRepository
}

// mount registers the scheduling routes. State transitions are modeled as
// sub-resources acted on with POST (…/booking, …/cancellation, …/reschedule)
// rather than verbs on the collection, so each command has a stable, versioned
// address.
func (h schedulingAPI) mount(r chi.Router) {
	r.Route("/appointments", func(r chi.Router) {
		r.Post("/", h.holdSlot)
		r.Get("/{id}", h.getAppointment)
		r.Post("/{id}/booking", h.book)
		r.Post("/{id}/cancellation", h.cancel)
		r.Post("/{id}/reschedule", h.reschedule)
	})
	r.Route("/provider-schedules", func(r chi.Router) {
		r.Post("/", h.publishAvailability)
		r.Get("/{id}", h.getSchedule)
	})
}

// --- request DTOs ---

type holdSlotRequest struct {
	ProviderID string `json:"providerId"`
	TimeSlot   string `json:"timeSlot"`
	PatientID  string `json:"patientId"`
}

type bookRequest struct {
	HoldToken string `json:"holdToken"`
	PatientID string `json:"patientId"`
	Reason    string `json:"reason"`
}

type cancelRequest struct {
	Reason string `json:"reason"`
}

type rescheduleRequest struct {
	NewTimeSlot string `json:"newTimeSlot"`
}

type publishAvailabilityRequest struct {
	ProviderID string   `json:"providerId"`
	Windows    []string `json:"windows"`
}

// --- response DTOs ---

// appointmentResponse is the safe outward projection of an appointment. It
// exposes lifecycle and participant identifiers plus the concurrency version,
// and nothing the persistence layer encrypts.
type appointmentResponse struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	ProviderID string `json:"providerId,omitempty"`
	PatientID  string `json:"patientId,omitempty"`
	TimeSlot   string `json:"timeSlot,omitempty"`
	Version    int    `json:"version"`
}

func toAppointmentResponse(a *schedmodel.AppointmentAggregate) appointmentResponse {
	return appointmentResponse{
		ID:         a.ID,
		Status:     string(a.Status),
		ProviderID: a.ScopedProviderID,
		PatientID:  a.ScopedPatientID,
		TimeSlot:   a.HeldTimeSlot,
		Version:    a.GetVersion(),
	}
}

type providerScheduleResponse struct {
	ID               string   `json:"id"`
	ProviderID       string   `json:"providerId,omitempty"`
	PublishedWindows []string `json:"publishedWindows,omitempty"`
	BlockedIntervals []string `json:"blockedIntervals,omitempty"`
	Version          int      `json:"version"`
}

func toProviderScheduleResponse(s *schedmodel.ProviderScheduleAggregate) providerScheduleResponse {
	return providerScheduleResponse{
		ID:               s.ID,
		ProviderID:       s.ScopedProviderID,
		PublishedWindows: s.PublishedWindows,
		BlockedIntervals: s.BlockedIntervals,
		Version:          s.GetVersion(),
	}
}

// --- appointment handlers ---

// holdSlot reserves a provider slot for a patient, bringing a new appointment
// into being. The identity of the caller is available from the trusted headers;
// authorization for the action was already decided at the edge.
func (h schedulingAPI) holdSlot(w http.ResponseWriter, r *http.Request) {
	var req holdSlotRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	for _, v := range [...]struct{ val, field string }{
		{req.ProviderID, "providerId"},
		{req.TimeSlot, "timeSlot"},
		{req.PatientID, "patientId"},
	} {
		if err := requireField(v.val, v.field); err != nil {
			writeError(w, err)
			return
		}
	}

	agg := &schedmodel.AppointmentAggregate{ID: newID("appt")}
	cmd := schedmodel.HoldSlotCmd{ProviderId: req.ProviderID, TimeSlot: req.TimeSlot, PatientId: req.PatientID}
	if _, err := agg.Execute(cmd); err != nil {
		writeError(w, execErr(err, schedmodel.ErrSlotDoubleBooked))
		return
	}
	if err := h.appointments.Save(r.Context(), agg); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toAppointmentResponse(agg))
}

func (h schedulingAPI) getAppointment(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	agg, err := h.appointments.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAppointmentResponse(agg))
}

func (h schedulingAPI) book(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var req bookRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	agg, err := h.appointments.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	cmd := schedmodel.BookAppointmentCmd{HoldToken: req.HoldToken, PatientId: req.PatientID, Reason: req.Reason}
	if _, err := agg.Execute(cmd); err != nil {
		writeError(w, execErr(err, schedmodel.ErrSlotDoubleBooked))
		return
	}
	if err := h.appointments.Save(r.Context(), agg); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAppointmentResponse(agg))
}

func (h schedulingAPI) cancel(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var req cancelRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	agg, err := h.appointments.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	cmd := schedmodel.CancelAppointmentCmd{AppointmentId: id, Reason: req.Reason}
	if _, err := agg.Execute(cmd); err != nil {
		writeError(w, execErr(err, schedmodel.ErrSlotDoubleBooked))
		return
	}
	if err := h.appointments.Save(r.Context(), agg); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAppointmentResponse(agg))
}

func (h schedulingAPI) reschedule(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var req rescheduleRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	agg, err := h.appointments.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	cmd := schedmodel.RescheduleAppointmentCmd{AppointmentId: id, NewTimeSlot: req.NewTimeSlot}
	if _, err := agg.Execute(cmd); err != nil {
		writeError(w, execErr(err, schedmodel.ErrSlotDoubleBooked))
		return
	}
	if err := h.appointments.Save(r.Context(), agg); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAppointmentResponse(agg))
}

// --- provider-schedule handlers ---

func (h schedulingAPI) publishAvailability(w http.ResponseWriter, r *http.Request) {
	var req publishAvailabilityRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	if err := requireField(req.ProviderID, "providerId"); err != nil {
		writeError(w, err)
		return
	}
	if len(req.Windows) == 0 {
		writeError(w, badRequest("windows must contain at least one entry"))
		return
	}

	agg := &schedmodel.ProviderScheduleAggregate{ID: newID("sched")}
	cmd := schedmodel.PublishAvailabilityCmd{ProviderId: req.ProviderID, Windows: req.Windows}
	if _, err := agg.Execute(cmd); err != nil {
		writeError(w, execErr(err))
		return
	}
	if err := h.schedules.Save(r.Context(), agg); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toProviderScheduleResponse(agg))
}

func (h schedulingAPI) getSchedule(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	agg, err := h.schedules.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toProviderScheduleResponse(agg))
}

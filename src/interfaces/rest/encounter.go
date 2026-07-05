package rest

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	clinicalmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/clinicalrecords/model"
	clinicalrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/clinicalrecords/repository"
)

// encounterAPI adapts the clinical encounter lifecycle to HTTP: opening a
// telehealth encounter, signing its SOAP note, and completing it. Every
// mutation records a compliance entry, so a documented encounter leaves an
// audit trail. Responses carry only coded, non-PHI fields — the signed note and
// diagnoses the persistence layer seals never cross this boundary.
type encounterAPI struct {
	encounters clinicalrepo.EncounterRepository
	audit      AuditSink
}

func (h encounterAPI) mount(r chi.Router) {
	r.Route("/encounters", func(r chi.Router) {
		r.Post("/", h.open)
		r.Get("/{id}", h.get)
		r.Post("/{id}/soap-note", h.signNote)
		r.Post("/{id}/completion", h.complete)
	})
}

// --- request DTOs ---

type openEncounterRequest struct {
	AppointmentID string `json:"appointmentId"`
	ProviderID    string `json:"providerId"`
	PatientID     string `json:"patientId"`
}

type diagnosisDTO struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

type signNoteRequest struct {
	ProviderID string         `json:"providerId"`
	SoapNote   string         `json:"soapNote"`
	Diagnoses  []diagnosisDTO `json:"diagnoses"`
}

type completeEncounterRequest struct {
	ProviderID string `json:"providerId"`
}

// --- response DTO ---

type encounterResponse struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	ProviderID  string `json:"providerId,omitempty"`
	PatientID   string `json:"patientId,omitempty"`
	VideoRoomID string `json:"videoRoomId,omitempty"`
	NoteSigned  bool   `json:"noteSigned"`
	Version     int    `json:"version"`
}

func toEncounterResponse(e *clinicalmodel.EncounterAggregate) encounterResponse {
	return encounterResponse{
		ID:          e.ID,
		Status:      string(e.Status),
		ProviderID:  e.ScopedProviderID,
		PatientID:   e.ScopedPatientID,
		VideoRoomID: e.VideoRoomID,
		NoteSigned:  e.Note != nil && e.Note.Signed,
		Version:     e.GetVersion(),
	}
}

// --- handlers ---

// open brings a new encounter into being for a booked appointment, provisioning
// its video room. The action is audited against the encounter id.
func (h encounterAPI) open(w http.ResponseWriter, r *http.Request) {
	var req openEncounterRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	for _, v := range [...]struct{ val, field string }{
		{req.AppointmentID, "appointmentId"},
		{req.ProviderID, "providerId"},
		{req.PatientID, "patientId"},
	} {
		if err := requireField(v.val, v.field); err != nil {
			writeError(w, err)
			return
		}
	}

	agg := &clinicalmodel.EncounterAggregate{ID: newID("enc")}
	cmd := clinicalmodel.OpenEncounterCmd{
		AppointmentId: req.AppointmentID,
		ProviderId:    req.ProviderID,
		PatientId:     req.PatientID,
	}
	if _, err := agg.Execute(cmd); err != nil {
		writeError(w, execErr(err))
		return
	}
	if err := h.encounters.Save(r.Context(), agg); err != nil {
		writeError(w, err)
		return
	}
	if err := recordAudit(r.Context(), h.audit, callerSubject(r), agg.ID, "encounter.open"); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toEncounterResponse(agg))
}

func (h encounterAPI) get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	agg, err := h.encounters.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toEncounterResponse(agg))
}

// signNote seals the SOAP note and coded diagnoses onto an open encounter.
func (h encounterAPI) signNote(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var req signNoteRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	if err := requireField(req.SoapNote, "soapNote"); err != nil {
		writeError(w, err)
		return
	}
	if len(req.Diagnoses) == 0 {
		writeError(w, badRequest("diagnoses must contain at least one entry"))
		return
	}
	agg, err := h.encounters.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	diagnoses := make([]clinicalmodel.Diagnosis, 0, len(req.Diagnoses))
	for _, d := range req.Diagnoses {
		diagnoses = append(diagnoses, clinicalmodel.Diagnosis{Code: d.Code, Description: d.Description})
	}
	cmd := clinicalmodel.SignSoapNoteCmd{
		EncounterId: id,
		ProviderId:  req.ProviderID,
		SoapNote:    req.SoapNote,
		Diagnoses:   diagnoses,
	}
	if _, err := agg.Execute(cmd); err != nil {
		writeError(w, execErr(err))
		return
	}
	if err := h.encounters.Save(r.Context(), agg); err != nil {
		writeError(w, err)
		return
	}
	if err := recordAudit(r.Context(), h.audit, callerSubject(r), agg.ID, "encounter.sign-note"); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toEncounterResponse(agg))
}

// complete closes a documented encounter.
func (h encounterAPI) complete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var req completeEncounterRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	agg, err := h.encounters.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	cmd := clinicalmodel.CompleteEncounterCmd{EncounterId: id, ProviderId: req.ProviderID}
	if _, err := agg.Execute(cmd); err != nil {
		writeError(w, execErr(err))
		return
	}
	if err := h.encounters.Save(r.Context(), agg); err != nil {
		writeError(w, err)
		return
	}
	if err := recordAudit(r.Context(), h.audit, callerSubject(r), agg.ID, "encounter.complete"); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toEncounterResponse(agg))
}

// callerSubject returns the trusted subject the edge stamped on the request, so
// an audit entry names the actor that drove the flow.
func callerSubject(r *http.Request) string {
	return CallerFrom(r.Context()).Subject
}

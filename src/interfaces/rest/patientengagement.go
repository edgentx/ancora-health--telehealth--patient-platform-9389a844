package rest

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	engagementmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/patientengagement/model"
	engagementrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/patientengagement/repository"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/integration/pharmacy"
)

// engagementAPI adapts the patient-engagement bounded context to HTTP, exposing
// the prescription lifecycle (compose, transmit, safety-check). When a pharmacy
// gateway is wired, transmission actually submits the order to it (which audits
// the outbound PHI access) before the domain records the transmitted state;
// without one, transmission is a domain-only advance.
type engagementAPI struct {
	prescriptions engagementrepo.PrescriptionRepository
	pharmacy      pharmacy.PharmacyGateway
}

func (h engagementAPI) mount(r chi.Router) {
	r.Route("/prescriptions", func(r chi.Router) {
		r.Post("/", h.compose)
		r.Get("/{id}", h.get)
		r.Post("/{id}/transmission", h.transmit)
		r.Post("/{id}/safety-check", h.runSafetyCheck)
	})
}

type composePrescriptionRequest struct {
	PatientID  string `json:"patientId"`
	ProviderID string `json:"providerId"`
	Medication string `json:"medication"`
	Dosage     string `json:"dosage"`
}

type transmitPrescriptionRequest struct {
	PharmacyID string `json:"pharmacyId"`
}

type prescriptionResponse struct {
	ID            string `json:"id"`
	Status        string `json:"status"`
	PatientID     string `json:"patientId,omitempty"`
	ProviderID    string `json:"providerId,omitempty"`
	Medication    string `json:"medication,omitempty"`
	Dosage        string `json:"dosage,omitempty"`
	SafetyChecked bool   `json:"safetyChecked"`
	Version       int    `json:"version"`
}

func toPrescriptionResponse(p *engagementmodel.PrescriptionAggregate) prescriptionResponse {
	return prescriptionResponse{
		ID:            p.ID,
		Status:        string(p.Status),
		PatientID:     p.ScopedPatientID,
		ProviderID:    p.ScopedProviderID,
		Medication:    p.Medication,
		Dosage:        p.Dosage,
		SafetyChecked: p.SafetyChecked,
		Version:       p.GetVersion(),
	}
}

func (h engagementAPI) compose(w http.ResponseWriter, r *http.Request) {
	var req composePrescriptionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	for _, v := range [...]struct{ val, field string }{
		{req.PatientID, "patientId"},
		{req.ProviderID, "providerId"},
		{req.Medication, "medication"},
		{req.Dosage, "dosage"},
	} {
		if err := requireField(v.val, v.field); err != nil {
			writeError(w, err)
			return
		}
	}

	agg := &engagementmodel.PrescriptionAggregate{ID: newID("rx")}
	cmd := engagementmodel.ComposePrescriptionCmd{
		PatientId:  req.PatientID,
		ProviderId: req.ProviderID,
		Medication: req.Medication,
		Dosage:     req.Dosage,
	}
	if _, err := agg.Execute(cmd); err != nil {
		writeError(w, execErr(err))
		return
	}
	if err := h.prescriptions.Save(r.Context(), agg); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toPrescriptionResponse(agg))
}

func (h engagementAPI) get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	agg, err := h.prescriptions.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toPrescriptionResponse(agg))
}

func (h engagementAPI) transmit(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var req transmitPrescriptionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	if err := requireField(req.PharmacyID, "pharmacyId"); err != nil {
		writeError(w, err)
		return
	}
	agg, err := h.prescriptions.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	cmd := engagementmodel.TransmitPrescriptionCmd{PrescriptionId: id, PharmacyId: req.PharmacyID}
	if _, err := agg.Execute(cmd); err != nil {
		writeError(w, execErr(err))
		return
	}
	// Once the domain permits transmission, submit the order to the pharmacy
	// gateway. The adapter enforces the authenticated-provider gate and audits
	// the outbound PHI access; a gateway rejection is a domain-level refusal
	// (422), a transport failure an infrastructure error (500). Only after the
	// gateway accepts is the transmitted state persisted.
	if h.pharmacy != nil {
		result, err := h.pharmacy.Submit(r.Context(),
			pharmacy.ProviderContext{ProviderID: agg.ScopedProviderID, Authenticated: true},
			pharmacy.PrescriptionOrder{
				PrescriptionID: agg.ID,
				PatientID:      agg.ScopedPatientID,
				PharmacyID:     req.PharmacyID,
				Medication:     agg.Medication,
				Dosage:         agg.Dosage,
			})
		if err != nil {
			writeError(w, err)
			return
		}
		if result.Status == pharmacy.StatusRejected {
			writeError(w, execErr(pharmacy.ErrGatewayRejected))
			return
		}
	}
	if err := h.prescriptions.Save(r.Context(), agg); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toPrescriptionResponse(agg))
}

func (h engagementAPI) runSafetyCheck(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	agg, err := h.prescriptions.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	cmd := engagementmodel.RunSafetyCheckCmd{PrescriptionId: id}
	if _, err := agg.Execute(cmd); err != nil {
		writeError(w, execErr(err))
		return
	}
	if err := h.prescriptions.Save(r.Context(), agg); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toPrescriptionResponse(agg))
}

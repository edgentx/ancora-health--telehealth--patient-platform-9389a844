package rest

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	clinicalmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/clinicalrecords/model"
	clinicalrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/clinicalrecords/repository"
)

// clinicalAPI adapts the clinical-records bounded context to HTTP. It exposes
// lab orders; the responses carry only coded, non-PHI fields — the encrypted
// clinical payload the persistence layer seals never crosses this boundary.
type clinicalAPI struct {
	labOrders clinicalrepo.LabOrderRepository
}

func (h clinicalAPI) mount(r chi.Router) {
	r.Route("/lab-orders", func(r chi.Router) {
		r.Post("/", h.place)
		r.Get("/{id}", h.get)
	})
}

type placeLabOrderRequest struct {
	PatientID  string `json:"patientId"`
	ProviderID string `json:"providerId"`
	TestCode   string `json:"testCode"`
}

type labOrderResponse struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	PatientID  string `json:"patientId,omitempty"`
	ProviderID string `json:"providerId,omitempty"`
	TestCode   string `json:"testCode,omitempty"`
	Version    int    `json:"version"`
}

func toLabOrderResponse(o *clinicalmodel.LabOrderAggregate) labOrderResponse {
	return labOrderResponse{
		ID:         o.ID,
		Status:     string(o.Status),
		PatientID:  o.ScopedPatientID,
		ProviderID: o.ScopedProviderID,
		TestCode:   o.TestCode,
		Version:    o.GetVersion(),
	}
}

func (h clinicalAPI) place(w http.ResponseWriter, r *http.Request) {
	var req placeLabOrderRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	for _, v := range [...]struct{ val, field string }{
		{req.PatientID, "patientId"},
		{req.ProviderID, "providerId"},
		{req.TestCode, "testCode"},
	} {
		if err := requireField(v.val, v.field); err != nil {
			writeError(w, err)
			return
		}
	}

	agg := &clinicalmodel.LabOrderAggregate{ID: newID("lab")}
	cmd := clinicalmodel.PlaceLabOrderCmd{PatientId: req.PatientID, ProviderId: req.ProviderID, TestCode: req.TestCode}
	if _, err := agg.Execute(cmd); err != nil {
		writeError(w, execErr(err))
		return
	}
	if err := h.labOrders.Save(r.Context(), agg); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toLabOrderResponse(agg))
}

func (h clinicalAPI) get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	agg, err := h.labOrders.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toLabOrderResponse(agg))
}

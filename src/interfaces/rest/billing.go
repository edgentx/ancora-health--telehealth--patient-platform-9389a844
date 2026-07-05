package rest

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	billingmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/billingandinsurance/model"
	billingrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/billingandinsurance/repository"
)

// billingAPI adapts the billing-and-insurance bounded context to HTTP, exposing
// the insurance-policy lifecycle (register, verify eligibility).
type billingAPI struct {
	policies billingrepo.InsurancePolicyRepository
}

func (h billingAPI) mount(r chi.Router) {
	r.Route("/insurance-policies", func(r chi.Router) {
		r.Post("/", h.register)
		r.Get("/{id}", h.get)
		r.Post("/{id}/eligibility", h.verifyEligibility)
	})
}

type effectiveDatesDTO struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type registerPolicyRequest struct {
	PatientID       string            `json:"patientId"`
	PayerIdentifier string            `json:"payerIdentifier"`
	EffectiveDates  effectiveDatesDTO `json:"effectiveDates"`
}

type verifyEligibilityRequest struct {
	ServiceDate string `json:"serviceDate"`
}

type insurancePolicyResponse struct {
	ID                  string            `json:"id"`
	Status              string            `json:"status"`
	PatientID           string            `json:"patientId,omitempty"`
	PayerIdentifier     string            `json:"payerIdentifier,omitempty"`
	EffectiveDates      effectiveDatesDTO `json:"effectiveDates"`
	VerifiedServiceDate string            `json:"verifiedServiceDate,omitempty"`
	Version             int               `json:"version"`
}

func toInsurancePolicyResponse(p *billingmodel.InsurancePolicyAggregate) insurancePolicyResponse {
	return insurancePolicyResponse{
		ID:                  p.ID,
		Status:              string(p.Status),
		PatientID:           p.PatientID,
		PayerIdentifier:     p.PayerIdentifier,
		EffectiveDates:      effectiveDatesDTO{Start: p.EffectiveDates.Start, End: p.EffectiveDates.End},
		VerifiedServiceDate: p.VerifiedServiceDate,
		Version:             p.GetVersion(),
	}
}

func (h billingAPI) register(w http.ResponseWriter, r *http.Request) {
	var req registerPolicyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	for _, v := range [...]struct{ val, field string }{
		{req.PatientID, "patientId"},
		{req.PayerIdentifier, "payerIdentifier"},
		{req.EffectiveDates.Start, "effectiveDates.start"},
		{req.EffectiveDates.End, "effectiveDates.end"},
	} {
		if err := requireField(v.val, v.field); err != nil {
			writeError(w, err)
			return
		}
	}

	agg := &billingmodel.InsurancePolicyAggregate{ID: newID("pol")}
	cmd := billingmodel.RegisterInsurancePolicyCmd{
		PatientId:       req.PatientID,
		PayerIdentifier: req.PayerIdentifier,
		EffectiveDates:  billingmodel.EffectiveDates{Start: req.EffectiveDates.Start, End: req.EffectiveDates.End},
	}
	if _, err := agg.Execute(cmd); err != nil {
		writeError(w, execErr(err, billingmodel.ErrActivePrimaryPolicyExists))
		return
	}
	if err := h.policies.Save(r.Context(), agg); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toInsurancePolicyResponse(agg))
}

func (h billingAPI) get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	agg, err := h.policies.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toInsurancePolicyResponse(agg))
}

func (h billingAPI) verifyEligibility(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var req verifyEligibilityRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	if err := requireField(req.ServiceDate, "serviceDate"); err != nil {
		writeError(w, err)
		return
	}
	agg, err := h.policies.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	cmd := billingmodel.VerifyEligibilityCmd{PolicyId: id, ServiceDate: req.ServiceDate}
	if _, err := agg.Execute(cmd); err != nil {
		writeError(w, execErr(err))
		return
	}
	if err := h.policies.Save(r.Context(), agg); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toInsurancePolicyResponse(agg))
}

package rest

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	adminmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/administrationandanalytics/model"
	adminrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/administrationandanalytics/repository"
)

// adminAPI adapts the administration-and-analytics bounded context to HTTP,
// exposing the clinic directory: provider registration, specialty management and
// clinic configuration.
type adminAPI struct {
	directories adminrepo.ClinicDirectoryRepository
}

func (h adminAPI) mount(r chi.Router) {
	r.Route("/clinic-directories", func(r chi.Router) {
		r.Post("/", h.create)
		r.Get("/{id}", h.get)
		r.Post("/{id}/providers", h.registerProvider)
		r.Post("/{id}/specialties", h.manageSpecialty)
		r.Post("/{id}/clinics", h.configureClinic)
	})
}

type registerProviderRequest struct {
	ProviderID  string   `json:"providerId"`
	Specialties []string `json:"specialties"`
	ClinicIDs   []string `json:"clinicIds"`
}

type manageSpecialtyRequest struct {
	SpecialtyCode string `json:"specialtyCode"`
	DisplayName   string `json:"displayName"`
}

type configureClinicRequest struct {
	ClinicIdentity string `json:"clinicIdentity"`
	OperatingHours string `json:"operatingHours"`
}

type clinicDirectoryResponse struct {
	ID             string   `json:"id"`
	ProviderIDs    []string `json:"providerIds,omitempty"`
	SpecialtyCodes []string `json:"specialtyCodes,omitempty"`
	ClinicIDs      []string `json:"clinicIds,omitempty"`
	Version        int      `json:"version"`
}

func toClinicDirectoryResponse(d *adminmodel.ClinicDirectoryAggregate) clinicDirectoryResponse {
	return clinicDirectoryResponse{
		ID:             d.ID,
		ProviderIDs:    d.ProviderIDs,
		SpecialtyCodes: d.SpecialtyCodes,
		ClinicIDs:      d.ClinicIDs,
		Version:        d.GetVersion(),
	}
}

// create provisions a new, empty clinic directory. The directory's zero value is
// a valid empty directory, so creation is a pure Save with no command — the
// provider/specialty/clinic commands then build it up.
func (h adminAPI) create(w http.ResponseWriter, r *http.Request) {
	agg := &adminmodel.ClinicDirectoryAggregate{ID: newID("dir")}
	if err := h.directories.Save(r.Context(), agg); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toClinicDirectoryResponse(agg))
}

func (h adminAPI) get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	agg, err := h.directories.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toClinicDirectoryResponse(agg))
}

func (h adminAPI) registerProvider(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var req registerProviderRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	if err := requireField(req.ProviderID, "providerId"); err != nil {
		writeError(w, err)
		return
	}
	if len(req.Specialties) == 0 {
		writeError(w, badRequest("specialties must contain at least one entry"))
		return
	}
	if len(req.ClinicIDs) == 0 {
		writeError(w, badRequest("clinicIds must contain at least one entry"))
		return
	}
	agg, err := h.directories.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	cmd := adminmodel.RegisterProviderCmd{ProviderId: req.ProviderID, Specialties: req.Specialties, ClinicIds: req.ClinicIDs}
	if _, err := agg.Execute(cmd); err != nil {
		writeError(w, execErr(err, adminmodel.ErrDuplicateSpecialtyCode, adminmodel.ErrClinicDeactivationBlocked))
		return
	}
	if err := h.directories.Save(r.Context(), agg); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toClinicDirectoryResponse(agg))
}

func (h adminAPI) manageSpecialty(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var req manageSpecialtyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	for _, v := range [...]struct{ val, field string }{
		{req.SpecialtyCode, "specialtyCode"},
		{req.DisplayName, "displayName"},
	} {
		if err := requireField(v.val, v.field); err != nil {
			writeError(w, err)
			return
		}
	}
	agg, err := h.directories.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	cmd := adminmodel.ManageSpecialtyCmd{SpecialtyCode: req.SpecialtyCode, DisplayName: req.DisplayName}
	if _, err := agg.Execute(cmd); err != nil {
		writeError(w, execErr(err, adminmodel.ErrDuplicateSpecialtyCode))
		return
	}
	if err := h.directories.Save(r.Context(), agg); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toClinicDirectoryResponse(agg))
}

func (h adminAPI) configureClinic(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var req configureClinicRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	for _, v := range [...]struct{ val, field string }{
		{req.ClinicIdentity, "clinicIdentity"},
		{req.OperatingHours, "operatingHours"},
	} {
		if err := requireField(v.val, v.field); err != nil {
			writeError(w, err)
			return
		}
	}
	agg, err := h.directories.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	cmd := adminmodel.ConfigureClinicCmd{ClinicIdentity: req.ClinicIdentity, OperatingHours: req.OperatingHours}
	if _, err := agg.Execute(cmd); err != nil {
		writeError(w, execErr(err, adminmodel.ErrClinicDeactivationBlocked))
		return
	}
	if err := h.directories.Save(r.Context(), agg); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toClinicDirectoryResponse(agg))
}

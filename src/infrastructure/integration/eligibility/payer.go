// Package eligibility is the outbound payer-eligibility adapter. It queries a
// payer's eligibility service and maps the external response onto the domain
// InsurancePolicy model, so the billing context can register a policy or verify
// eligibility from a normalised result rather than the payer's wire format.
package eligibility

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/billingandinsurance/model"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/integration"
)

// Request identifies the coverage to check: the patient, the payer, the member
// id the payer knows the patient by, and the service date coverage is verified
// for.
type Request struct {
	PatientID       string
	PayerIdentifier string
	MemberID        string
	// ServiceDate is the date coverage is verified for (RFC 3339 date).
	ServiceDate string
}

// Result is the eligibility outcome mapped onto the domain vocabulary. It
// carries enough to drive both RegisterInsurancePolicyCmd (via the payer,
// patient and effective dates) and the verified service date, plus the active
// flag the caller gates registration on.
type Result struct {
	// Active reports whether the payer confirmed active coverage as of the
	// service date.
	Active bool
	// PatientID is the patient the coverage is for, echoed from the request.
	PatientID string
	// PayerIdentifier is the payer that underwrites the coverage.
	PayerIdentifier string
	// EffectiveDates is the coverage window mapped onto the domain's value type.
	EffectiveDates model.EffectiveDates
	// ServiceDate is the date coverage was verified for.
	ServiceDate string
}

// ToRegisterCommand projects the mapped result onto the domain command that
// admits the policy into billing. It is the seam that turns an external
// eligibility response into a domain InsurancePolicy registration without the
// caller re-deriving the mapping.
func (r Result) ToRegisterCommand() model.RegisterInsurancePolicyCmd {
	return model.RegisterInsurancePolicyCmd{
		PatientId:       r.PatientID,
		PayerIdentifier: r.PayerIdentifier,
		EffectiveDates:  r.EffectiveDates,
	}
}

// Gateway is the outbound port for payer eligibility.
type Gateway interface {
	CheckEligibility(ctx context.Context, req Request) (Result, error)
}

// Sentinel errors surfaced to the domain.
var (
	// ErrInvalidRequest is returned when an eligibility request omits a required
	// field.
	ErrInvalidRequest = errors.New("eligibility: invalid request")

	// ErrPayerRejected is returned when the payer answers with a client (4xx)
	// status — e.g. an unknown member — and the check will not be retried.
	ErrPayerRejected = errors.New("eligibility: payer rejected the request")

	// ErrPayerUnavailable is returned when the payer service could not be reached
	// or answered with a server error after retries.
	ErrPayerUnavailable = errors.New("eligibility: payer service unavailable")
)

// Adapter is the payer-eligibility Gateway. It queries the payer over the shared
// transport, records the outbound PHI access, and maps the response onto the
// domain InsurancePolicy model.
type Adapter struct {
	client *integration.Client
	audit  integration.AuditRecorder
	now    func() time.Time
}

// NewAdapter builds the eligibility adapter over an integration transport. audit
// may be nil, in which case outbound-access recording is skipped.
func NewAdapter(client *integration.Client, audit integration.AuditRecorder) *Adapter {
	return &Adapter{client: client, audit: audit, now: time.Now}
}

// eligibilityResponse is the payer's wire format (an X12 270/271-style response
// normalised to JSON at the payer edge).
type eligibilityResponse struct {
	Active            bool   `json:"active"`
	Payer             string `json:"payer"`
	CoverageStartDate string `json:"coverageStartDate"`
	CoverageEndDate   string `json:"coverageEndDate"`
}

// CheckEligibility queries the payer and maps the response onto the domain
// model. It records the outbound PHI access (referencing the patient and payer,
// never the member data) before the call, and translates transport failures
// into the payer-facing error vocabulary.
func (a *Adapter) CheckEligibility(ctx context.Context, req Request) (Result, error) {
	if req.PatientID == "" || req.PayerIdentifier == "" || req.MemberID == "" || req.ServiceDate == "" {
		return Result{}, ErrInvalidRequest
	}

	if err := integration.RecordIfSet(ctx, a.audit, integration.OutboundAccess{
		ActorContext: req.PatientID,
		ResourceRef:  req.PayerIdentifier + "/" + req.PatientID,
		Action:       "eligibility.check",
		Destination:  "payer-eligibility",
		OccurredAt:   a.now(),
	}); err != nil {
		return Result{}, fmt.Errorf("eligibility: audit outbound access: %w", err)
	}

	query := url.Values{
		"payer":       []string{req.PayerIdentifier},
		"member":      []string{req.MemberID},
		"serviceDate": []string{req.ServiceDate},
	}
	resp, err := a.client.Send(ctx, &integration.Request{
		Method: http.MethodGet,
		URL:    "/eligibility?" + query.Encode(),
		Header: http.Header{"Accept": []string{"application/json"}},
	})
	if err != nil {
		return Result{}, mapPayerError(err)
	}

	var payer eligibilityResponse
	if err := json.Unmarshal(resp.Body, &payer); err != nil {
		return Result{}, fmt.Errorf("eligibility: decode payer response: %w", err)
	}

	// Map the payer response onto the domain InsurancePolicy value types.
	return Result{
		Active:          payer.Active,
		PatientID:       req.PatientID,
		PayerIdentifier: firstNonEmpty(payer.Payer, req.PayerIdentifier),
		EffectiveDates: model.EffectiveDates{
			Start: payer.CoverageStartDate,
			End:   payer.CoverageEndDate,
		},
		ServiceDate: req.ServiceDate,
	}, nil
}

// mapPayerError translates a transport error into the adapter's payer-facing
// vocabulary: a 4xx is a terminal rejection, anything else is unavailable.
func mapPayerError(err error) error {
	var statusErr *integration.StatusError
	if errors.As(err, &statusErr) {
		if statusErr.StatusCode >= 400 && statusErr.StatusCode < 500 {
			return fmt.Errorf("%w: status %d", ErrPayerRejected, statusErr.StatusCode)
		}
		return fmt.Errorf("%w: status %d", ErrPayerUnavailable, statusErr.StatusCode)
	}
	return fmt.Errorf("%w: %v", ErrPayerUnavailable, err)
}

// firstNonEmpty returns a if it is non-empty, otherwise b. It lets the payer's
// echoed identifier win while falling back to the requested one.
func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

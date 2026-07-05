// Package pharmacy is the outbound e-prescribing adapter. It submits and cancels
// prescriptions against a Surescripts-compatible pharmacy gateway over the
// shared integration transport, enforces that only an authenticated provider may
// transmit, maps the gateway's transmission outcome back to the domain, and
// audits every outbound access to prescription PHI.
package pharmacy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/integration"
)

// TransmissionStatus is the domain-facing outcome of a pharmacy transmission,
// normalised from the gateway's own vocabulary so the prescription aggregate
// never has to know the Surescripts wire codes.
type TransmissionStatus string

const (
	// StatusAccepted means the pharmacy gateway accepted the message for
	// delivery to the destination pharmacy.
	StatusAccepted TransmissionStatus = "accepted"
	// StatusQueued means the message was received and queued but not yet
	// confirmed accepted by the destination.
	StatusQueued TransmissionStatus = "queued"
	// StatusRejected means the gateway rejected the message (e.g. a validation or
	// destination error).
	StatusRejected TransmissionStatus = "rejected"
)

// ProviderContext carries the authenticated-provider assertion passed through
// from the trusted edge (headers verified upstream). The adapter refuses to
// transmit unless Authenticated is true — enforcing, at the integration
// boundary, the same invariant the prescription aggregate enforces on the domain
// side: a prescription may only be issued by an authenticated provider.
type ProviderContext struct {
	// ProviderID is the internal identity of the issuing provider.
	ProviderID string
	// ProviderNPI is the National Provider Identifier the gateway attributes the
	// prescription to.
	ProviderNPI string
	// Authenticated reports whether the edge verified the provider's identity. It
	// is the gate on every outbound transmission.
	Authenticated bool
}

// PrescriptionOrder is the drafted order handed to the gateway. Medication and
// Dosage are PHI: they travel to the gateway in the request body but are never
// written to logs or the audit trail.
type PrescriptionOrder struct {
	PrescriptionID string
	PatientID      string
	PharmacyID     string
	Medication     string
	Dosage         string
}

// CancelOrder identifies a previously transmitted prescription to cancel.
type CancelOrder struct {
	PrescriptionID string
	PharmacyID     string
	// Reason is an optional cancellation reason code passed to the gateway.
	Reason string
}

// TransmissionResult is what the adapter surfaces to the domain after a
// submit/cancel: the normalised status plus the gateway's own reference so a
// transmission can be reconciled later.
type TransmissionResult struct {
	PrescriptionID string
	PharmacyID     string
	Status         TransmissionStatus
	// GatewayReference is the message id the gateway assigned, used to correlate
	// asynchronous status updates.
	GatewayReference string
}

// PharmacyGateway is the outbound port for e-prescribing. Submit transmits a new
// prescription; Cancel supersedes a transmitted one. Both require an
// authenticated provider context.
type PharmacyGateway interface {
	Submit(ctx context.Context, provider ProviderContext, order PrescriptionOrder) (TransmissionResult, error)
	Cancel(ctx context.Context, provider ProviderContext, order CancelOrder) (TransmissionResult, error)
}

// Sentinel errors the adapter surfaces to the domain.
var (
	// ErrUnauthenticatedProvider is returned when a transmission is attempted
	// with a provider context that the edge did not authenticate. It is checked
	// before any request leaves the process.
	ErrUnauthenticatedProvider = errors.New("pharmacy: unauthenticated provider context rejected before submission")

	// ErrIncompleteOrder is returned when a submit/cancel omits a required field.
	ErrIncompleteOrder = errors.New("pharmacy: order is missing a required field")

	// ErrGatewayRejected is returned when the gateway rejects the message with a
	// client (4xx) status; the transmission will not be retried.
	ErrGatewayRejected = errors.New("pharmacy: gateway rejected the prescription")

	// ErrGatewayUnavailable is returned when the gateway could not be reached or
	// answered with a server error after retries.
	ErrGatewayUnavailable = errors.New("pharmacy: gateway unavailable")
)

// Message types on the Surescripts-compatible NCPDP SCRIPT contract.
const (
	messageTypeNewRx    = "NewRx"
	messageTypeCancelRx = "CancelRx"
)

// Adapter is the Surescripts-compatible PharmacyGateway. It sends NCPDP
// SCRIPT-style messages through the shared transport and records every outbound
// access to prescription PHI.
type Adapter struct {
	client *integration.Client
	audit  integration.AuditRecorder
	now    func() time.Time
}

// NewAdapter builds the pharmacy adapter over an integration transport. audit
// may be nil, in which case outbound-access recording is skipped.
func NewAdapter(client *integration.Client, audit integration.AuditRecorder) *Adapter {
	return &Adapter{client: client, audit: audit, now: time.Now}
}

// scriptRequest is the Surescripts-compatible message envelope. The provider
// assertion and the drafted order ride in the body; the gateway attributes the
// message to the NPI.
type scriptRequest struct {
	MessageType    string `json:"messageType"`
	PrescriptionID string `json:"prescriptionId"`
	PharmacyID     string `json:"pharmacyId"`
	ProviderNPI    string `json:"providerNpi,omitempty"`
	PatientID      string `json:"patientId,omitempty"`
	Medication     string `json:"medication,omitempty"`
	Dosage         string `json:"dosage,omitempty"`
	Reason         string `json:"reason,omitempty"`
}

// scriptResponse is the gateway's acknowledgement.
type scriptResponse struct {
	Status     string `json:"status"`
	MessageRef string `json:"messageRef"`
}

// Submit transmits a NewRx to the pharmacy gateway. It rejects an
// unauthenticated provider before the request leaves the process, validates the
// order, records the outbound PHI access, and maps the gateway acknowledgement
// to a TransmissionResult.
func (a *Adapter) Submit(ctx context.Context, provider ProviderContext, order PrescriptionOrder) (TransmissionResult, error) {
	if !provider.Authenticated {
		return TransmissionResult{}, ErrUnauthenticatedProvider
	}
	if order.PrescriptionID == "" || order.PharmacyID == "" || order.Medication == "" || order.Dosage == "" {
		return TransmissionResult{}, ErrIncompleteOrder
	}

	req := scriptRequest{
		MessageType:    messageTypeNewRx,
		PrescriptionID: order.PrescriptionID,
		PharmacyID:     order.PharmacyID,
		ProviderNPI:    provider.ProviderNPI,
		PatientID:      order.PatientID,
		Medication:     order.Medication,
		Dosage:         order.Dosage,
	}
	return a.transmit(ctx, provider, "eprescribe.submit", order.PrescriptionID, order.PharmacyID, req)
}

// Cancel supersedes a transmitted prescription by sending a CancelRx. Like
// Submit it enforces the authenticated-provider gate and audits the access; a
// cancellation carries no medication/dosage PHI in the body.
func (a *Adapter) Cancel(ctx context.Context, provider ProviderContext, order CancelOrder) (TransmissionResult, error) {
	if !provider.Authenticated {
		return TransmissionResult{}, ErrUnauthenticatedProvider
	}
	if order.PrescriptionID == "" || order.PharmacyID == "" {
		return TransmissionResult{}, ErrIncompleteOrder
	}

	req := scriptRequest{
		MessageType:    messageTypeCancelRx,
		PrescriptionID: order.PrescriptionID,
		PharmacyID:     order.PharmacyID,
		ProviderNPI:    provider.ProviderNPI,
		Reason:         order.Reason,
	}
	return a.transmit(ctx, provider, "eprescribe.cancel", order.PrescriptionID, order.PharmacyID, req)
}

// transmit encodes and sends a SCRIPT message, records the outbound access, and
// maps the response. It is shared by Submit and Cancel so the auth gate, audit,
// and error mapping live in one place.
func (a *Adapter) transmit(ctx context.Context, provider ProviderContext, action, prescriptionID, pharmacyID string, msg scriptRequest) (TransmissionResult, error) {
	// Audit the outbound PHI access before the call. The record references the
	// prescription and provider but never the medication or dosage.
	if err := integration.RecordIfSet(ctx, a.audit, integration.OutboundAccess{
		ActorContext: provider.ProviderID,
		ResourceRef:  prescriptionID,
		Action:       action,
		Destination:  "pharmacy-gateway",
		OccurredAt:   a.now(),
	}); err != nil {
		return TransmissionResult{}, fmt.Errorf("pharmacy: audit outbound access: %w", err)
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return TransmissionResult{}, fmt.Errorf("pharmacy: encode request: %w", err)
	}

	resp, err := a.client.Send(ctx, &integration.Request{
		Method: http.MethodPost,
		URL:    a.endpoint(),
		Header: http.Header{
			"Content-Type": []string{"application/json"},
			// The trusted provider assertion is forwarded to the gateway; the edge
			// has already authenticated it, and the guard above refuses to send it
			// unless it was.
			"X-Provider-Id":  []string{provider.ProviderID},
			"X-Provider-NPI": []string{provider.ProviderNPI},
		},
		Body: body,
	})
	if err != nil {
		return TransmissionResult{}, mapTransmitError(err)
	}

	var ack scriptResponse
	if err := json.Unmarshal(resp.Body, &ack); err != nil {
		return TransmissionResult{}, fmt.Errorf("pharmacy: decode gateway response: %w", err)
	}
	return TransmissionResult{
		PrescriptionID:   prescriptionID,
		PharmacyID:       pharmacyID,
		Status:           normalizeStatus(ack.Status),
		GatewayReference: ack.MessageRef,
	}, nil
}

// endpoint is the SCRIPT message endpoint on the gateway base URL.
func (a *Adapter) endpoint() string {
	return "/script/messages"
}

// mapTransmitError translates a transport error into the adapter's domain-facing
// vocabulary: a client rejection (4xx) is a terminal ErrGatewayRejected, while a
// server error or unreachable gateway is ErrGatewayUnavailable.
func mapTransmitError(err error) error {
	var statusErr *integration.StatusError
	if errors.As(err, &statusErr) {
		if statusErr.StatusCode >= 400 && statusErr.StatusCode < 500 {
			return fmt.Errorf("%w: status %d", ErrGatewayRejected, statusErr.StatusCode)
		}
		return fmt.Errorf("%w: status %d", ErrGatewayUnavailable, statusErr.StatusCode)
	}
	return fmt.Errorf("%w: %v", ErrGatewayUnavailable, err)
}

// normalizeStatus maps the gateway's acknowledgement vocabulary to the
// domain-facing TransmissionStatus. Unknown codes are treated as rejected so an
// unexpected response never reads as success.
func normalizeStatus(gateway string) TransmissionStatus {
	switch gateway {
	case "Accepted", "accepted", "Success", "success":
		return StatusAccepted
	case "Queued", "queued", "Pending", "pending":
		return StatusQueued
	default:
		return StatusRejected
	}
}

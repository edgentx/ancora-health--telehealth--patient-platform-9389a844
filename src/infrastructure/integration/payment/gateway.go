// Package payment is the outbound payment adapter and inbound payment webhook
// handler. The adapter creates tokenized charges against a PCI-compliant gateway
// (Stripe-compatible) without ever handling or storing raw card data, and the
// webhook handler verifies the gateway's HMAC signature before idempotently
// applying a payment event to the domain.
package payment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/integration"
)

// ChargeRequest is a tokenized charge. By construction it carries only a gateway
// token standing in for the patient's card — there is no field for a card
// number, CVC, or expiry, so the type itself makes it impossible for raw card
// data to reach this adapter or be persisted downstream.
type ChargeRequest struct {
	// IdempotencyKey lets the gateway (and this adapter) deduplicate a retried
	// charge so a network retry never double-charges. It is required.
	IdempotencyKey string
	// InvoiceID is the invoice the charge is captured against.
	InvoiceID string
	// PaymentToken is the gateway token representing the patient's card. Raw card
	// data is never accepted.
	PaymentToken string
	// AmountCents is the amount to capture, in whole cents. It must be positive.
	AmountCents int64
	// Currency is the ISO-4217 currency code, e.g. "usd".
	Currency string
}

// ChargeResult is the gateway's confirmation of a created charge.
type ChargeResult struct {
	// GatewayChargeID is the charge identity the gateway assigned.
	GatewayChargeID string
	// Status is the gateway charge status, e.g. "succeeded" or "pending".
	Status string
	// AmountCents echoes the captured amount.
	AmountCents int64
}

// PaymentGateway is the outbound port for creating charges.
type PaymentGateway interface {
	CreateCharge(ctx context.Context, req ChargeRequest) (ChargeResult, error)
}

// Sentinel errors surfaced to the domain.
var (
	// ErrInvalidCharge is returned when a charge request is missing a required
	// field or carries a non-positive amount.
	ErrInvalidCharge = errors.New("payment: invalid charge request")

	// ErrRawCardData is returned when a payment token looks like a raw primary
	// account number (PAN) rather than a tokenized reference — a defensive guard
	// so raw card data can never be forwarded even if a caller misuses the token
	// field.
	ErrRawCardData = errors.New("payment: raw card data must never be submitted; use a gateway token")

	// ErrChargeDeclined is returned when the gateway declines the charge with a
	// client (4xx) status; it will not be retried.
	ErrChargeDeclined = errors.New("payment: charge declined by gateway")

	// ErrGatewayUnavailable is returned when the gateway could not be reached or
	// answered with a server error after retries.
	ErrGatewayUnavailable = errors.New("payment: gateway unavailable")
)

// Adapter is the Stripe-compatible PaymentGateway. It posts tokenized charges
// through the shared transport. It deliberately performs no logging of the
// request body, so a payment token or amount never lands in a log line.
type Adapter struct {
	client *integration.Client
}

// NewAdapter builds the payment adapter over an integration transport.
func NewAdapter(client *integration.Client) *Adapter {
	return &Adapter{client: client}
}

// chargeRequestBody is the form the gateway expects: it references the card only
// by token (`source`), never by PAN.
type chargeRequestBody struct {
	Source   string `json:"source"`
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
	Invoice  string `json:"invoice"`
}

type chargeResponseBody struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Amount int64  `json:"amount"`
}

// CreateCharge submits a tokenized charge to the gateway. It validates the
// request, refuses anything resembling raw card data, forwards the idempotency
// key so a retry is safe, and maps the gateway response or error to the domain
// vocabulary.
func (a *Adapter) CreateCharge(ctx context.Context, req ChargeRequest) (ChargeResult, error) {
	if req.IdempotencyKey == "" || req.InvoiceID == "" || req.PaymentToken == "" || req.AmountCents <= 0 {
		return ChargeResult{}, ErrInvalidCharge
	}
	if looksLikeCardNumber(req.PaymentToken) {
		return ChargeResult{}, ErrRawCardData
	}
	currency := req.Currency
	if currency == "" {
		currency = "usd"
	}

	body, err := json.Marshal(chargeRequestBody{
		Source:   req.PaymentToken,
		Amount:   req.AmountCents,
		Currency: currency,
		Invoice:  req.InvoiceID,
	})
	if err != nil {
		return ChargeResult{}, fmt.Errorf("payment: encode request: %w", err)
	}

	resp, err := a.client.Send(ctx, &integration.Request{
		Method: http.MethodPost,
		URL:    "/v1/charges",
		Header: http.Header{
			"Content-Type":    []string{"application/json"},
			"Idempotency-Key": []string{req.IdempotencyKey},
		},
		Body: body,
	})
	if err != nil {
		return ChargeResult{}, mapChargeError(err)
	}

	var out chargeResponseBody
	if err := json.Unmarshal(resp.Body, &out); err != nil {
		return ChargeResult{}, fmt.Errorf("payment: decode gateway response: %w", err)
	}
	return ChargeResult{
		GatewayChargeID: out.ID,
		Status:          out.Status,
		AmountCents:     out.Amount,
	}, nil
}

// mapChargeError translates a transport error into the adapter's domain-facing
// vocabulary: a 4xx is a terminal decline, anything else is an unavailable
// gateway.
func mapChargeError(err error) error {
	var statusErr *integration.StatusError
	if errors.As(err, &statusErr) {
		if statusErr.StatusCode >= 400 && statusErr.StatusCode < 500 {
			return fmt.Errorf("%w: status %d", ErrChargeDeclined, statusErr.StatusCode)
		}
		return fmt.Errorf("%w: status %d", ErrGatewayUnavailable, statusErr.StatusCode)
	}
	return fmt.Errorf("%w: %v", ErrGatewayUnavailable, err)
}

// looksLikeCardNumber reports whether s resembles a raw PAN: 13–19 digits after
// stripping spaces and dashes. Gateway tokens carry non-digit prefixes (e.g.
// "tok_", "pm_"), so this catches a caller accidentally passing a card number
// without rejecting legitimate tokens.
func looksLikeCardNumber(s string) bool {
	digits := 0
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			digits++
		case r == ' ' || r == '-':
			// separators are ignored
		default:
			return false
		}
	}
	return digits >= 13 && digits <= 19
}

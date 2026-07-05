package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync"
)

// maxWebhookBytes bounds the webhook body read into memory. Signature
// verification and idempotency both need the full body, but a hostile caller
// must not be able to exhaust memory.
const maxWebhookBytes = 1 << 20 // 1 MiB

// signatureHeader is the header the gateway delivers its HMAC signature in.
const signatureHeader = "X-Signature"

// PaymentEvent is the domain-facing view of a verified gateway webhook. It
// carries no card data — only the payment reference, the event identity used for
// idempotency, and the outcome the domain applies.
type PaymentEvent struct {
	// EventID is the gateway's unique event identity, the key idempotency is keyed
	// on so a redelivered webhook is applied at most once.
	EventID string
	// Type is the event kind, e.g. "charge.succeeded" or "charge.failed".
	Type string
	// PaymentID is the payment the event pertains to.
	PaymentID string
	// Status is the resulting gateway status.
	Status string
	// Signature is the verified HMAC of the raw payload, preserved so the domain
	// can record the verified-webhook-to-reconciled provenance.
	Signature string
}

// PaymentReconciler is the port the webhook handler applies a verified event
// through. A concrete implementation loads the Payment/Invoice aggregate and
// dispatches ReconcilePaymentCmd, translating the event into a domain state
// advance. Keeping it a port lets the handler stay free of the persistence and
// aggregate machinery.
type PaymentReconciler interface {
	Reconcile(ctx context.Context, event PaymentEvent) error
}

// IdempotencyStore records which gateway events have already been applied so a
// redelivery is a no-op. Seen reports whether an event id was already processed;
// Remember marks it processed. Implementations must make the check-and-set safe
// under concurrent delivery.
type IdempotencyStore interface {
	Seen(ctx context.Context, eventID string) (bool, error)
	Remember(ctx context.Context, eventID string) error
}

// Errors surfaced by signature verification and handling.
var (
	// ErrInvalidSignature is returned when the delivered signature does not match
	// the HMAC computed over the payload with the shared secret.
	ErrInvalidSignature = errors.New("payment: webhook signature verification failed")

	// ErrMalformedEvent is returned when a verified payload cannot be decoded into
	// a PaymentEvent.
	ErrMalformedEvent = errors.New("payment: malformed webhook event")
)

// Verifier computes and checks the HMAC-SHA256 signature the gateway attaches to
// each webhook. Signatures are compared in constant time so a caller cannot
// probe the secret via timing.
type Verifier struct {
	secret []byte
}

// NewVerifier builds a signature verifier over the gateway's shared webhook
// secret.
func NewVerifier(secret []byte) *Verifier {
	return &Verifier{secret: secret}
}

// Verify reports whether signatureHex is the lowercase-hex HMAC-SHA256 of
// payload under the secret. The comparison is constant time.
func (v *Verifier) Verify(payload []byte, signatureHex string) bool {
	mac := hmac.New(sha256.New, v.secret)
	mac.Write(payload)
	expected := mac.Sum(nil)

	got, err := hex.DecodeString(signatureHex)
	if err != nil {
		return false
	}
	return hmac.Equal(expected, got)
}

// Sign computes the lowercase-hex HMAC-SHA256 of payload under the secret. It is
// the inverse of Verify, used by tests (and any internal signing) to produce a
// signature the handler will accept.
func (v *Verifier) Sign(payload []byte) string {
	mac := hmac.New(sha256.New, v.secret)
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// wireEvent is the gateway's webhook envelope on the wire.
type wireEvent struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Data struct {
		PaymentID string `json:"paymentId"`
		Status    string `json:"status"`
	} `json:"data"`
}

// WebhookHandler is the inbound HTTP handler for payment webhooks. It verifies
// the HMAC signature, rejects anything unsigned or mis-signed, and idempotently
// applies a verified event to the domain via the reconciler.
type WebhookHandler struct {
	verifier    *Verifier
	reconciler  PaymentReconciler
	idempotency IdempotencyStore
}

// NewWebhookHandler wires the verifier, the reconciler the verified event is
// applied through, and the idempotency store that makes redelivery safe.
func NewWebhookHandler(verifier *Verifier, reconciler PaymentReconciler, idempotency IdempotencyStore) *WebhookHandler {
	return &WebhookHandler{verifier: verifier, reconciler: reconciler, idempotency: idempotency}
}

// ServeHTTP implements http.Handler. The flow is: read the bounded body, verify
// the signature (400 on failure), decode the event (400 on malformed),
// short-circuit on a already-seen event (200, no-op), then reconcile and record
// the event id. A reconciler failure is a 500 so the gateway retries; an
// idempotency-store failure before applying is likewise a 500.
func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBytes))
	if err != nil {
		http.Error(w, "cannot read body", http.StatusBadRequest)
		return
	}

	signature := r.Header.Get(signatureHeader)
	if signature == "" || !h.verifier.Verify(body, signature) {
		// Reject invalid signatures before the payload is trusted or applied.
		http.Error(w, ErrInvalidSignature.Error(), http.StatusBadRequest)
		return
	}

	var evt wireEvent
	if err := json.Unmarshal(body, &evt); err != nil || evt.ID == "" {
		http.Error(w, ErrMalformedEvent.Error(), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	seen, err := h.idempotency.Seen(ctx, evt.ID)
	if err != nil {
		http.Error(w, "idempotency check failed", http.StatusInternalServerError)
		return
	}
	if seen {
		// A redelivered event has already advanced the domain; acknowledge without
		// re-applying so the update stays idempotent.
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := h.reconciler.Reconcile(ctx, PaymentEvent{
		EventID:   evt.ID,
		Type:      evt.Type,
		PaymentID: evt.Data.PaymentID,
		Status:    evt.Data.Status,
		Signature: signature,
	}); err != nil {
		// Do not mark the event seen: a failed apply must be retried on
		// redelivery, so leave the idempotency key unset.
		http.Error(w, "reconcile failed", http.StatusInternalServerError)
		return
	}

	if err := h.idempotency.Remember(ctx, evt.ID); err != nil {
		http.Error(w, "idempotency persist failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// MemoryIdempotencyStore is an in-process IdempotencyStore for local development
// and tests. Production deployments back the store with a shared, durable store
// so idempotency holds across processes; the semantics are identical.
type MemoryIdempotencyStore struct {
	mu   sync.Mutex
	seen map[string]struct{}
}

// NewMemoryIdempotencyStore builds an empty in-memory idempotency store.
func NewMemoryIdempotencyStore() *MemoryIdempotencyStore {
	return &MemoryIdempotencyStore{seen: make(map[string]struct{})}
}

// Seen reports whether eventID has been remembered.
func (s *MemoryIdempotencyStore) Seen(_ context.Context, eventID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.seen[eventID]
	return ok, nil
}

// Remember marks eventID as processed.
func (s *MemoryIdempotencyStore) Remember(_ context.Context, eventID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seen[eventID] = struct{}{}
	return nil
}

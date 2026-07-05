// Package realtime holds the transport-facing gateways for the platform's
// real-time surfaces served at ws.{project}.vforce360.ai:
//
//   - SignalingGateway coordinates WebRTC video visits — it relays SDP
//     offers/answers and ICE candidates between peers over a WebSocket and issues
//     ephemeral coturn (TURN/STUN) credentials for NAT traversal. A session is
//     scoped to a valid appointment/care relationship supplied at handshake, and
//     unscoped sessions are refused.
//   - MessagingGateway is the secure-messaging WebSocket gateway — it
//     authenticates via trusted headers, persists messages through the
//     MessageThread repository, and fans out via a pub/sub broker so every
//     replica delivers consistently.
//
// Both gateways record connection lifecycle and PHI message access to the audit
// trail.
package realtime

// Conn is the minimal WebSocket connection the gateways read from and write to.
// It is deliberately the subset of *gorilla/websocket.Conn the gateways use
// (ReadJSON/WriteJSON/Close), which that type satisfies directly, so the gateway
// logic is testable against an in-memory fake without a live socket — the same
// port-and-adapter approach the persistence and locking layers take.
//
// Implementations need not be safe for concurrent writers; the gateways
// serialize all writes to a connection through a single goroutine.
type Conn interface {
	// ReadJSON decodes the next JSON frame from the peer into v, blocking until a
	// frame arrives or the connection fails.
	ReadJSON(v any) error
	// WriteJSON encodes v as a JSON frame to the peer.
	WriteJSON(v any) error
	// Close tears down the connection.
	Close() error
}

// Handshake carries the trusted context a gateway is handed when a connection is
// established. The fronting ingress authenticates the caller and injects the
// identity headers; the gateway trusts them (it never re-authenticates) and uses
// them to scope the session. The values originate from request headers/query on
// the upgrade request.
type Handshake struct {
	// UserID is the authenticated principal, injected by the trusted ingress
	// (e.g. the X-User-Id header). Empty means the ingress did not authenticate
	// the caller, which the gateways reject.
	UserID string
	// Role is the caller's role (e.g. the X-User-Role header): "patient",
	// "provider", or a care-team role. It is recorded in the audit actor context.
	Role string
	// AppointmentID scopes a signaling session to a booked video visit. Required
	// by the signaling gateway.
	AppointmentID string
	// CareRelationshipID optionally scopes a session to a specific care
	// relationship; when present it must be active.
	CareRelationshipID string
	// ThreadID scopes a messaging connection to a secure message thread. Required
	// by the messaging gateway.
	ThreadID string
}

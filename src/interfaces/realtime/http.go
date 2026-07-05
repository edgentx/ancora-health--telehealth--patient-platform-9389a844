package realtime

import (
	"net/http"

	"github.com/gorilla/websocket"
)

// Trusted-identity headers injected by the fronting ingress. The gateways trust
// these verbatim: TLS-terminating ingress authenticates the caller and stamps
// the principal and role, so the gateway never re-authenticates.
const (
	HeaderUserID = "X-User-Id"
	HeaderRole   = "X-User-Role"
)

// upgrader promotes an HTTP request to a WebSocket. Origin checking is delegated
// to the ingress/CORS layer in front of the gateway, so any origin that reaches
// this handler has already been vetted.
var upgrader = websocket.Upgrader{
	CheckOrigin: func(*http.Request) bool { return true },
}

// HandshakeFromRequest builds the trusted handshake from the upgrade request:
// identity from the trusted headers, session scope from the query string. The
// *gorilla/websocket.Conn returned by the upgrader satisfies Conn directly.
func HandshakeFromRequest(r *http.Request) Handshake {
	q := r.URL.Query()
	return Handshake{
		UserID:             r.Header.Get(HeaderUserID),
		Role:               r.Header.Get(HeaderRole),
		AppointmentID:      q.Get("appointmentId"),
		CareRelationshipID: q.Get("careRelationshipId"),
		ThreadID:           q.Get("threadId"),
	}
}

// SignalingHTTPHandler adapts the signaling gateway to an http.Handler that
// upgrades the connection and drives the session.
func (g *SignalingGateway) SignalingHTTPHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		_ = g.Handle(r.Context(), conn, HandshakeFromRequest(r))
	})
}

// MessagingHTTPHandler adapts the messaging gateway to an http.Handler that
// upgrades the connection and drives the session.
func (g *MessagingGateway) MessagingHTTPHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		_ = g.Handle(r.Context(), conn, HandshakeFromRequest(r))
	})
}

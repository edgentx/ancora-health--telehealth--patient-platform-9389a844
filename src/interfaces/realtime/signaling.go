package realtime

import (
	"context"
	"sync"
	"time"

	"github.com/pion/webrtc/v4"
)

// Signal message types exchanged over the signaling WebSocket. Offers, answers
// and ICE candidates are relayed opaquely between the two peers; the server
// never terminates the media, it only brokers the handshake and NAT-traversal
// credentials.
const (
	// SignalOffer carries an SDP offer from the initiating peer.
	SignalOffer = "offer"
	// SignalAnswer carries an SDP answer from the responding peer.
	SignalAnswer = "answer"
	// SignalICECandidate carries a trickled ICE candidate.
	SignalICECandidate = "ice-candidate"
	// SignalSessionReady is the server's opening frame: it confirms the scoped
	// session and delivers the coturn credentials the client feeds to its
	// RTCPeerConnection.
	SignalSessionReady = "session-ready"
	// SignalError reports a refused or failed session to the client.
	SignalError = "error"
)

// SignalMessage is one frame on the signaling channel. Exactly one of SDP or
// Candidate is set depending on Type; the server stamps From so the receiving
// peer knows which participant produced the frame. The SDP/ICE payloads use the
// Pion (WebRTC) wire types so they round-trip a browser RTCPeerConnection
// verbatim.
type SignalMessage struct {
	Type      string                     `json:"type"`
	SDP       *webrtc.SessionDescription `json:"sdp,omitempty"`
	Candidate *webrtc.ICECandidateInit   `json:"candidate,omitempty"`
	From      string                     `json:"from,omitempty"`
	Error     string                     `json:"error,omitempty"`
}

// SessionReady is the server's opening frame on a scoped session: it echoes the
// visit scope and hands the peer its ephemeral coturn credentials.
type SessionReady struct {
	Type          string          `json:"type"`
	AppointmentID string          `json:"appointmentId"`
	Role          string          `json:"role"`
	ICEServers    TurnCredentials `json:"iceServers"`
}

// SignalingGateway is the WebRTC signaling service. It scopes each connection to
// a booked appointment via the SessionAuthorizer, relays SDP/ICE frames between
// the peers sharing that appointment, issues coturn credentials for NAT
// traversal, and records the session lifecycle to the audit trail.
type SignalingGateway struct {
	authorizer SessionAuthorizer
	coturn     *CoturnIssuer
	audit      AuditRecorder
	hub        *signalingHub
	now        func() time.Time
}

// NewSignalingGateway wires a signaling gateway. now defaults to time.Now when
// nil, and is injectable so audit timestamps are deterministic under test.
func NewSignalingGateway(
	authorizer SessionAuthorizer,
	coturn *CoturnIssuer,
	audit AuditRecorder,
	now func() time.Time,
) *SignalingGateway {
	if now == nil {
		now = time.Now
	}
	return &SignalingGateway{
		authorizer: authorizer,
		coturn:     coturn,
		audit:      audit,
		hub:        newSignalingHub(),
		now:        now,
	}
}

// Handle drives one signaling connection end to end: it authorizes the
// handshake, refuses unscoped sessions, sends the session-ready frame with
// coturn credentials, then relays signaling frames to the peer until the
// connection closes. It blocks until the connection ends and always closes conn.
func (g *SignalingGateway) Handle(ctx context.Context, conn Conn, h Handshake) error {
	defer conn.Close()

	scope, err := g.authorizer.Authorize(ctx, h)
	if err != nil {
		// A refused session is a security-relevant event; record the denial and
		// tell the client before dropping the connection.
		_ = g.audit.Record(ctx, auditActor(h.UserID, h.Role), "appointment:"+h.AppointmentID, "signaling.session.denied", g.now())
		_ = conn.WriteJSON(SignalMessage{Type: SignalError, Error: err.Error()})
		return err
	}

	creds := g.coturn.Issue(h.UserID)
	if err := conn.WriteJSON(SessionReady{
		Type:          SignalSessionReady,
		AppointmentID: scope.AppointmentID,
		Role:          scope.CallerRole,
		ICEServers:    creds,
	}); err != nil {
		return err
	}

	p := &signalingPeer{conn: conn, userID: h.UserID, role: scope.CallerRole}
	g.hub.join(scope.AppointmentID, p)
	_ = g.audit.Record(ctx, auditActor(h.UserID, scope.CallerRole), "appointment:"+scope.AppointmentID, "signaling.session.opened", g.now())

	defer func() {
		g.hub.leave(scope.AppointmentID, p)
		_ = g.audit.Record(context.WithoutCancel(ctx), auditActor(h.UserID, scope.CallerRole), "appointment:"+scope.AppointmentID, "signaling.session.closed", g.now())
	}()

	for {
		var msg SignalMessage
		if err := conn.ReadJSON(&msg); err != nil {
			return err
		}
		// Only relayable signaling frames are forwarded; unknown types are ignored
		// so a peer cannot inject arbitrary control frames.
		switch msg.Type {
		case SignalOffer, SignalAnswer, SignalICECandidate:
			msg.From = h.UserID
			g.hub.relay(scope.AppointmentID, p, msg)
		default:
			// ignore
		}
	}
}

// signalingHub groups connected peers by appointment so a frame from one peer is
// relayed to the others in the same video visit.
type signalingHub struct {
	mu       sync.Mutex
	sessions map[string]map[*signalingPeer]struct{}
}

func newSignalingHub() *signalingHub {
	return &signalingHub{sessions: make(map[string]map[*signalingPeer]struct{})}
}

func (h *signalingHub) join(appointmentID string, p *signalingPeer) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.sessions[appointmentID] == nil {
		h.sessions[appointmentID] = make(map[*signalingPeer]struct{})
	}
	h.sessions[appointmentID][p] = struct{}{}
}

func (h *signalingHub) leave(appointmentID string, p *signalingPeer) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set := h.sessions[appointmentID]; set != nil {
		delete(set, p)
		if len(set) == 0 {
			delete(h.sessions, appointmentID)
		}
	}
}

// relay forwards msg to every peer in the session except the sender.
func (h *signalingHub) relay(appointmentID string, from *signalingPeer, msg SignalMessage) {
	h.mu.Lock()
	targets := make([]*signalingPeer, 0, len(h.sessions[appointmentID]))
	for p := range h.sessions[appointmentID] {
		if p != from {
			targets = append(targets, p)
		}
	}
	h.mu.Unlock()

	for _, p := range targets {
		p.write(msg)
	}
}

// signalingPeer is one connected participant. Its write mutex serializes frames
// written by the peer's own read loop (the session-ready frame) and by other
// peers' relay goroutines.
type signalingPeer struct {
	conn   Conn
	userID string
	role   string

	writeMu sync.Mutex
}

func (p *signalingPeer) write(msg SignalMessage) {
	p.writeMu.Lock()
	defer p.writeMu.Unlock()
	_ = p.conn.WriteJSON(msg)
}

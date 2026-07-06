package realtime

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	engagemodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/patientengagement/model"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/pubsub"
)

func TestHandshakeFromRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet,
		"/ws?appointmentId=appt-9&careRelationshipId=rel-9&threadId=thread-9", nil)
	req.Header.Set(HeaderUserID, "user-9")
	req.Header.Set(HeaderRole, "provider")

	h := HandshakeFromRequest(req)
	want := Handshake{
		UserID:             "user-9",
		Role:               "provider",
		AppointmentID:      "appt-9",
		CareRelationshipID: "rel-9",
		ThreadID:           "thread-9",
	}
	if h != want {
		t.Fatalf("handshake = %+v, want %+v", h, want)
	}
}

// dialWS upgrades a websocket connection to the handler server with the given
// identity header and query string.
func dialWS(t *testing.T, srvURL, query string, header http.Header) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(srvURL, "http") + query
	return websocket.DefaultDialer.Dial(wsURL, header)
}

func TestSignalingHTTPHandler_UpgradesAndRefusesUnscoped(t *testing.T) {
	store := NewMemoryAuditTrailStore()
	audit := NewTrailAuditRecorder(store, "sig")
	coturn := NewCoturnIssuer("secret", nil, time.Hour)
	gw := NewSignalingGateway(fakeAuthorizer{err: ErrSessionNotScoped}, coturn, audit, nil)

	srv := httptest.NewServer(gw.SignalingHTTPHandler())
	defer srv.Close()

	header := http.Header{}
	header.Set(HeaderUserID, "intruder")
	conn, _, err := dialWS(t, srv.URL, "?appointmentId=", header)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var frame SignalMessage
	if err := conn.ReadJSON(&frame); err != nil {
		t.Fatalf("read error frame: %v", err)
	}
	if frame.Type != SignalError {
		t.Fatalf("frame type = %q, want error", frame.Type)
	}
	waitForAuditAction(t, store, "sig", "signaling.session.denied", 1)
}

func TestMessagingHTTPHandler_UpgradesAndRefusesNonParticipant(t *testing.T) {
	thread := &engagemodel.MessageThreadAggregate{
		ID:              "thread-1",
		Status:          engagemodel.MessageThreadStatusOpen,
		ScopedPatientID: "patient-1",
	}
	threads := &fakeThreadRepo{m: map[string]*engagemodel.MessageThreadAggregate{thread.ID: thread}}
	store := NewMemoryAuditTrailStore()
	audit := NewTrailAuditRecorder(store, "msg")
	gw := NewMessagingGateway(threads, pubsub.NewMemoryBroker(4), audit, nil)

	srv := httptest.NewServer(gw.MessagingHTTPHandler())
	defer srv.Close()

	header := http.Header{}
	header.Set(HeaderUserID, "stranger")
	header.Set(HeaderRole, "patient")
	conn, _, err := dialWS(t, srv.URL, "?threadId=thread-1", header)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var frame MessageFrame
	if err := conn.ReadJSON(&frame); err != nil {
		t.Fatalf("read error frame: %v", err)
	}
	if frame.Type != SignalError {
		t.Fatalf("frame type = %q, want error", frame.Type)
	}
	assertAuditHas(t, store, "msg", "message.thread.access.denied")
}

func TestHTTPHandlers_NonWebSocketRequestDropped(t *testing.T) {
	// A plain HTTP request cannot be upgraded; the handler must return cleanly
	// (no panic) rather than driving a session.
	sigGW := NewSignalingGateway(fakeAuthorizer{}, NewCoturnIssuer("s", nil, time.Hour), NewTrailAuditRecorder(NewMemoryAuditTrailStore(), "sig"), nil)
	msgGW := NewMessagingGateway(&fakeThreadRepo{m: map[string]*engagemodel.MessageThreadAggregate{}}, pubsub.NewMemoryBroker(4), NewTrailAuditRecorder(NewMemoryAuditTrailStore(), "msg"), nil)

	for _, h := range []http.Handler{sigGW.SignalingHTTPHandler(), msgGW.MessagingHTTPHandler()} {
		srv := httptest.NewServer(h)
		resp, err := http.Get(srv.URL)
		if err != nil {
			srv.Close()
			t.Fatalf("GET: %v", err)
		}
		_ = resp.Body.Close()
		// gorilla responds 400 to a non-upgrade request; the key assertion is the
		// handler returned without hijacking.
		if resp.StatusCode == http.StatusSwitchingProtocols {
			srv.Close()
			t.Fatal("plain request unexpectedly upgraded")
		}
		srv.Close()
	}
}

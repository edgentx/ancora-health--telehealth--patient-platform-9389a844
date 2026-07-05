package realtime

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pion/webrtc/v4"

	authzmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/authorization/model"
	schedmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/scheduling/model"
)

// waitForAuditAction blocks until the trail records at least n entries with the
// given action, or fails after a timeout. Because the signaling gateway records
// "session.opened" immediately after a peer joins the relay hub, the test uses
// it as a deterministic barrier that both peers are registered before frames are
// relayed.
func waitForAuditAction(t *testing.T, store *MemoryAuditTrailStore, trailID, action string, n int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		trail, err := store.Load(context.Background(), trailID)
		if err != nil {
			t.Fatalf("load audit trail: %v", err)
		}
		if trail != nil {
			count := 0
			for _, e := range trail.Entries() {
				if e.Action == action {
					count++
				}
			}
			if count >= n {
				return
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %d %q audit entries", n, action)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func TestSignalingHandshakeIssuesCoturnAndRelaysFrames(t *testing.T) {
	store := NewMemoryAuditTrailStore()
	audit := NewTrailAuditRecorder(store, "sig")
	coturn := NewCoturnIssuer("shared-secret", []string{"turn:turn.example.ai:3478"}, time.Hour)
	gw := NewSignalingGateway(fakeAuthorizer{}, coturn, audit, nil)

	connA := newFakeConn()
	connB := newFakeConn()
	ctx := context.Background()
	go func() {
		_ = gw.Handle(ctx, connA, Handshake{UserID: "patient-1", Role: "patient", AppointmentID: "appt-1"})
	}()
	go func() {
		_ = gw.Handle(ctx, connB, Handshake{UserID: "provider-1", Role: "provider", AppointmentID: "appt-1"})
	}()

	// Each peer receives a session-ready frame carrying coturn credentials.
	var readyA SessionReady
	if err := connA.next(&readyA); err != nil {
		t.Fatalf("read ready A: %v", err)
	}
	if readyA.Type != SignalSessionReady {
		t.Fatalf("got type %q, want %q", readyA.Type, SignalSessionReady)
	}
	if readyA.ICEServers.Username == "" || readyA.ICEServers.Password == "" {
		t.Fatalf("expected coturn credentials, got %+v", readyA.ICEServers)
	}
	if len(readyA.ICEServers.URIs) == 0 {
		t.Fatal("expected TURN URIs advertised to the client")
	}
	var readyB SessionReady
	if err := connB.next(&readyB); err != nil {
		t.Fatalf("read ready B: %v", err)
	}

	// Both peers must be registered before a relayed frame can reach the other.
	waitForAuditAction(t, store, "sig", "signaling.session.opened", 2)

	// Peer A sends an SDP offer; the gateway relays it verbatim to peer B, stamped
	// with the sender's identity.
	offer := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: "v=0\r\no=- 1 1 IN IP4 0.0.0.0\r\n"}
	connA.push(SignalMessage{Type: SignalOffer, SDP: &offer})

	var relayed SignalMessage
	if err := connB.next(&relayed); err != nil {
		t.Fatalf("read relayed offer: %v", err)
	}
	if relayed.Type != SignalOffer {
		t.Fatalf("got type %q, want %q", relayed.Type, SignalOffer)
	}
	if relayed.From != "patient-1" {
		t.Fatalf("got From %q, want %q", relayed.From, "patient-1")
	}
	if relayed.SDP == nil || relayed.SDP.SDP != offer.SDP {
		t.Fatalf("relayed SDP mismatch: %+v", relayed.SDP)
	}

	// An ICE candidate from B is relayed back to A.
	frag := "candidate:1 1 udp 1 10.0.0.1 5000 typ host"
	connB.push(SignalMessage{Type: SignalICECandidate, Candidate: &webrtc.ICECandidateInit{Candidate: frag}})
	var relayedICE SignalMessage
	if err := connA.next(&relayedICE); err != nil {
		t.Fatalf("read relayed ICE: %v", err)
	}
	if relayedICE.Type != SignalICECandidate || relayedICE.Candidate == nil || relayedICE.Candidate.Candidate != frag {
		t.Fatalf("relayed ICE mismatch: %+v", relayedICE)
	}

	connA.Close()
	connB.Close()
}

func TestSignalingRefusesUnscopedSession(t *testing.T) {
	store := NewMemoryAuditTrailStore()
	audit := NewTrailAuditRecorder(store, "sig")
	coturn := NewCoturnIssuer("shared-secret", nil, time.Hour)
	gw := NewSignalingGateway(fakeAuthorizer{err: ErrSessionNotScoped}, coturn, audit, nil)

	conn := newFakeConn()
	err := gw.Handle(context.Background(), conn, Handshake{UserID: "intruder", AppointmentID: ""})
	if !errors.Is(err, ErrSessionNotScoped) {
		t.Fatalf("got err %v, want ErrSessionNotScoped", err)
	}

	var frame SignalMessage
	if err := conn.next(&frame); err != nil {
		t.Fatalf("read error frame: %v", err)
	}
	if frame.Type != SignalError {
		t.Fatalf("got type %q, want %q", frame.Type, SignalError)
	}

	// The denial is recorded to the audit trail.
	trail, err := store.Load(context.Background(), "sig")
	if err != nil || trail == nil {
		t.Fatalf("load trail: %v", err)
	}
	if got := trail.Entries(); len(got) != 1 || got[0].Action != "signaling.session.denied" {
		t.Fatalf("expected a single session.denied entry, got %+v", got)
	}
}

func TestAppointmentSessionAuthorizerScoping(t *testing.T) {
	booked := &schedmodel.AppointmentAggregate{
		ID:               "appt-1",
		Status:           schedmodel.AppointmentStatusBooked,
		ScopedProviderID: "provider-1",
		ScopedPatientID:  "patient-1",
	}
	open := &schedmodel.AppointmentAggregate{
		ID:               "appt-open",
		Status:           schedmodel.AppointmentStatusOpen,
		ScopedProviderID: "provider-1",
		ScopedPatientID:  "patient-1",
	}
	appts := &fakeAppointmentRepo{m: map[string]*schedmodel.AppointmentAggregate{
		"appt-1":    booked,
		"appt-open": open,
	}}
	rels := &fakeCareRelRepo{m: map[string]*authzmodel.CareRelationshipAggregate{
		"rel-1": {ID: "rel-1", Status: authzmodel.RelationshipStatusActive, ProviderID: "provider-1", PatientID: "patient-1"},
		"rel-x": {ID: "rel-x", Status: authzmodel.RelationshipStatusRevoked, ProviderID: "provider-1", PatientID: "patient-1"},
	}}
	authorizer := NewAppointmentSessionAuthorizer(appts, rels)
	ctx := context.Background()

	t.Run("scoped participant is authorized", func(t *testing.T) {
		scope, err := authorizer.Authorize(ctx, Handshake{UserID: "patient-1", AppointmentID: "appt-1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if scope.AppointmentID != "appt-1" || scope.CallerRole != "patient" {
			t.Fatalf("unexpected scope: %+v", scope)
		}
	})

	t.Run("active care relationship is honored", func(t *testing.T) {
		scope, err := authorizer.Authorize(ctx, Handshake{UserID: "provider-1", AppointmentID: "appt-1", CareRelationshipID: "rel-1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if scope.CareRelationshipID != "rel-1" || scope.CallerRole != "provider" {
			t.Fatalf("unexpected scope: %+v", scope)
		}
	})

	cases := []struct {
		name string
		h    Handshake
	}{
		{"missing appointment", Handshake{UserID: "patient-1"}},
		{"unknown appointment", Handshake{UserID: "patient-1", AppointmentID: "nope"}},
		{"appointment not booked", Handshake{UserID: "patient-1", AppointmentID: "appt-open"}},
		{"non-participant", Handshake{UserID: "stranger", AppointmentID: "appt-1"}},
		{"revoked care relationship", Handshake{UserID: "patient-1", AppointmentID: "appt-1", CareRelationshipID: "rel-x"}},
		{"unknown care relationship", Handshake{UserID: "patient-1", AppointmentID: "appt-1", CareRelationshipID: "ghost"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := authorizer.Authorize(ctx, tc.h); !errors.Is(err, ErrSessionNotScoped) {
				t.Fatalf("got err %v, want ErrSessionNotScoped", err)
			}
		})
	}
}

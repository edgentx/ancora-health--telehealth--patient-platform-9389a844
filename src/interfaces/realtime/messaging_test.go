package realtime

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	engagemodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/patientengagement/model"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/pubsub"
)

// countingBroker wraps a MemoryBroker and reports each Subscribe so a test can
// wait until both replicas have subscribed before publishing, making the
// cross-instance fan-out deterministic.
type countingBroker struct {
	*pubsub.MemoryBroker
	subscribed chan string
}

func (b *countingBroker) Subscribe(ctx context.Context, channel string) (pubsub.Subscription, error) {
	sub, err := b.MemoryBroker.Subscribe(ctx, channel)
	if err == nil {
		b.subscribed <- channel
	}
	return sub, err
}

func newOpenThread() *engagemodel.MessageThreadAggregate {
	return &engagemodel.MessageThreadAggregate{
		ID:                      "thread-1",
		Status:                  engagemodel.MessageThreadStatusOpen,
		ScopedPatientID:         "patient-1",
		ScopedCareTeamMemberIDs: []string{"clinician-1"},
		Subject:                 "post-visit follow-up",
	}
}

// TestMessagingFansOutAcrossTwoInstances proves a message posted on one gateway
// replica is delivered to a participant connected to a second replica, via the
// shared pub/sub broker — the multi-replica consistency guarantee.
func TestMessagingFansOutAcrossTwoInstances(t *testing.T) {
	thread := newOpenThread()
	threads := &fakeThreadRepo{m: map[string]*engagemodel.MessageThreadAggregate{thread.ID: thread}}
	broker := &countingBroker{MemoryBroker: pubsub.NewMemoryBroker(16), subscribed: make(chan string, 4)}
	store := NewMemoryAuditTrailStore()
	audit := NewTrailAuditRecorder(store, "msg")

	// Two gateway instances sharing the same broker and repository model two
	// replicas behind the same Redis and database.
	instance1 := NewMessagingGateway(threads, broker, audit, nil)
	instance2 := NewMessagingGateway(threads, broker, audit, nil)

	connPatient := newFakeConn()   // connected to instance 1
	connClinician := newFakeConn() // connected to instance 2
	ctx := context.Background()
	go func() {
		_ = instance1.Handle(ctx, connPatient, Handshake{UserID: "patient-1", Role: "patient", ThreadID: "thread-1"})
	}()
	go func() {
		_ = instance2.Handle(ctx, connClinician, Handshake{UserID: "clinician-1", Role: "provider", ThreadID: "thread-1"})
	}()

	// Wait until both replicas have subscribed before publishing.
	for i := 0; i < 2; i++ {
		select {
		case <-broker.subscribed:
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for both replicas to subscribe")
		}
	}

	// The patient posts on instance 1.
	connPatient.push(MessageFrame{Body: "labs look good — talk tomorrow"})

	// The clinician on instance 2 receives it, fanned out through the broker.
	select {
	case b := <-connClinician.out:
		var frame MessageFrame
		if err := json.Unmarshal(b, &frame); err != nil {
			t.Fatalf("decode delivered frame: %v", err)
		}
		if frame.Type != MessageType {
			t.Fatalf("got type %q, want %q", frame.Type, MessageType)
		}
		if frame.Body != "labs look good — talk tomorrow" {
			t.Fatalf("got body %q", frame.Body)
		}
		if frame.AuthorID != "patient-1" {
			t.Fatalf("got author %q, want patient-1", frame.AuthorID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("clinician on the second replica never received the message")
	}

	// The message was persisted through the MessageThread repository.
	stored, _ := threads.FindByID(ctx, "thread-1")
	if stored.PostedMessageCount != 1 {
		t.Fatalf("got posted count %d, want 1", stored.PostedMessageCount)
	}

	// PHI access and the secure post are recorded to the audit trail.
	assertAuditHas(t, store, "msg", "message.secure.posted")
	assertAuditHas(t, store, "msg", "message.phi.accessed")

	connPatient.Close()
	connClinician.Close()
}

func TestMessagingRefusesNonParticipant(t *testing.T) {
	thread := newOpenThread()
	threads := &fakeThreadRepo{m: map[string]*engagemodel.MessageThreadAggregate{thread.ID: thread}}
	broker := pubsub.NewMemoryBroker(16)
	store := NewMemoryAuditTrailStore()
	audit := NewTrailAuditRecorder(store, "msg")
	gw := NewMessagingGateway(threads, broker, audit, nil)

	conn := newFakeConn()
	err := gw.Handle(context.Background(), conn, Handshake{UserID: "stranger", Role: "patient", ThreadID: "thread-1"})
	if err == nil {
		t.Fatal("expected access to be denied")
	}

	var frame MessageFrame
	if err := conn.next(&frame); err != nil {
		t.Fatalf("read error frame: %v", err)
	}
	if frame.Type != SignalError {
		t.Fatalf("got type %q, want error frame", frame.Type)
	}
	assertAuditHas(t, store, "msg", "message.thread.access.denied")
}

// assertAuditHas fails unless the trail contains an entry with the given action.
func assertAuditHas(t *testing.T, store *MemoryAuditTrailStore, trailID, action string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		trail, err := store.Load(context.Background(), trailID)
		if err != nil {
			t.Fatalf("load trail: %v", err)
		}
		if trail != nil {
			for _, e := range trail.Entries() {
				if e.Action == action {
					return
				}
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("audit trail never recorded action %q", action)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

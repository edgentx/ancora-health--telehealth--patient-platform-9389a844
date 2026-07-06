package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"

	auditmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/auditandcompliance/model"
	engagemodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/patientengagement/model"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/pubsub"
)

func TestNewCoturnIssuer_NonPositiveTTLDefaultsToOneHour(t *testing.T) {
	iss := NewCoturnIssuer("secret", []string{"turn:x:3478"}, 0)
	creds := iss.Issue("user-1")
	if creds.TTLSeconds != 3600 {
		t.Fatalf("ttl = %d, want 3600 (one hour default)", creds.TTLSeconds)
	}
	if creds.Username == "" || creds.Password == "" {
		t.Fatalf("expected populated credentials, got %+v", creds)
	}
}

func TestNewTrailAuditRecorder_EmptyTrailIDDefaults(t *testing.T) {
	store := NewMemoryAuditTrailStore()
	rec := NewTrailAuditRecorder(store, "")

	if err := rec.Record(context.Background(), "actor", "res", "act", time.Now()); err != nil {
		t.Fatalf("Record: %v", err)
	}
	trail, err := store.Load(context.Background(), "realtime-access")
	if err != nil || trail == nil {
		t.Fatalf("expected default trail 'realtime-access', got trail=%v err=%v", trail, err)
	}
	if len(trail.Entries()) != 1 {
		t.Fatalf("entries = %d, want 1", len(trail.Entries()))
	}
}

// loadErrStore is an AuditTrailStore whose Load always fails.
type loadErrStore struct{}

func (loadErrStore) Load(context.Context, string) (*auditmodel.AuditTrailAggregate, error) {
	return nil, errors.New("store unreachable")
}
func (loadErrStore) Save(context.Context, *auditmodel.AuditTrailAggregate) error { return nil }

func TestTrailAuditRecorder_LoadErrorPropagates(t *testing.T) {
	rec := NewTrailAuditRecorder(loadErrStore{}, "t")
	if err := rec.Record(context.Background(), "a", "r", "act", time.Now()); err == nil {
		t.Fatal("expected Record to surface the store load error")
	}
}

func TestTrailAuditRecorder_ExecuteErrorPropagates(t *testing.T) {
	rec := NewTrailAuditRecorder(NewMemoryAuditTrailStore(), "t")
	// A zero timestamp breaks the append invariant, so Execute rejects the entry.
	if err := rec.Record(context.Background(), "actor", "res", "act", time.Time{}); err == nil {
		t.Fatal("expected Execute to reject an incomplete entry")
	}
}

func openThread(id, patient string) *engagemodel.MessageThreadAggregate {
	return &engagemodel.MessageThreadAggregate{
		ID:              id,
		Status:          engagemodel.MessageThreadStatusOpen,
		ScopedPatientID: patient,
	}
}

func TestPost_UnknownThreadDenied(t *testing.T) {
	threads := &fakeThreadRepo{m: map[string]*engagemodel.MessageThreadAggregate{}}
	gw := NewMessagingGateway(threads, pubsub.NewMemoryBroker(4), NewTrailAuditRecorder(NewMemoryAuditTrailStore(), "m"), nil)

	err := gw.post(context.Background(), Handshake{UserID: "patient-1", ThreadID: "ghost"}, "hello")
	if !errors.Is(err, ErrThreadAccessDenied) {
		t.Fatalf("err = %v, want ErrThreadAccessDenied", err)
	}
}

func TestPost_InvariantViolationPropagates(t *testing.T) {
	bad := openThread("bad", "patient-1")
	bad.NoActiveCareRelationship = true // breaks the post invariant
	threads := &fakeThreadRepo{m: map[string]*engagemodel.MessageThreadAggregate{bad.ID: bad}}
	gw := NewMessagingGateway(threads, pubsub.NewMemoryBroker(4), NewTrailAuditRecorder(NewMemoryAuditTrailStore(), "m"), nil)

	err := gw.post(context.Background(), Handshake{UserID: "patient-1", ThreadID: "bad"}, "hello")
	if err == nil || errors.Is(err, ErrThreadAccessDenied) {
		t.Fatalf("err = %v, want the domain invariant error", err)
	}
}

// saveErrThreadRepo returns a valid thread but fails to persist it.
type saveErrThreadRepo struct {
	thread *engagemodel.MessageThreadAggregate
}

func (r saveErrThreadRepo) FindByID(context.Context, string) (*engagemodel.MessageThreadAggregate, error) {
	return r.thread, nil
}
func (r saveErrThreadRepo) Save(context.Context, *engagemodel.MessageThreadAggregate) error {
	return errors.New("save failed")
}

func TestPost_SaveErrorPropagates(t *testing.T) {
	gw := NewMessagingGateway(saveErrThreadRepo{thread: openThread("t", "patient-1")},
		pubsub.NewMemoryBroker(4), NewTrailAuditRecorder(NewMemoryAuditTrailStore(), "m"), nil)

	err := gw.post(context.Background(), Handshake{UserID: "patient-1", ThreadID: "t"}, "hello")
	if err == nil || errors.Is(err, ErrThreadAccessDenied) {
		t.Fatalf("err = %v, want the save failure", err)
	}
}

// errBroker fails to subscribe, modelling an unreachable Redis.
type errBroker struct{}

func (errBroker) Publish(context.Context, string, []byte) error { return nil }
func (errBroker) Subscribe(context.Context, string) (pubsub.Subscription, error) {
	return nil, errors.New("broker unreachable")
}

func TestHandle_SubscribeErrorPropagates(t *testing.T) {
	thread := openThread("thread-1", "patient-1")
	threads := &fakeThreadRepo{m: map[string]*engagemodel.MessageThreadAggregate{thread.ID: thread}}
	gw := NewMessagingGateway(threads, errBroker{}, NewTrailAuditRecorder(NewMemoryAuditTrailStore(), "m"), nil)

	conn := newFakeConn()
	err := gw.Handle(context.Background(), conn, Handshake{UserID: "patient-1", ThreadID: "thread-1"})
	if err == nil || errors.Is(err, ErrThreadAccessDenied) {
		t.Fatalf("err = %v, want the subscribe failure", err)
	}
}

func TestDeliver_SkipsMalformedThenDelivers(t *testing.T) {
	broker := pubsub.NewMemoryBroker(8)
	gw := NewMessagingGateway(&fakeThreadRepo{m: map[string]*engagemodel.MessageThreadAggregate{}},
		broker, NewTrailAuditRecorder(NewMemoryAuditTrailStore(), "m"), nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sub, err := broker.Subscribe(ctx, threadChannel("thread-1"))
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	conn := newFakeConn()
	go gw.deliver(ctx, conn, sub, Handshake{UserID: "patient-1", Role: "patient", ThreadID: "thread-1"})

	// A malformed payload is skipped; a well-formed one is delivered.
	_ = broker.Publish(ctx, threadChannel("thread-1"), []byte("not json"))
	good, _ := json.Marshal(MessageFrame{Type: MessageType, ThreadID: "thread-1", AuthorID: "patient-1", Body: "hi"})
	_ = broker.Publish(ctx, threadChannel("thread-1"), good)

	select {
	case b := <-conn.out:
		var frame MessageFrame
		if err := json.Unmarshal(b, &frame); err != nil {
			t.Fatalf("decode delivered frame: %v", err)
		}
		if frame.Body != "hi" {
			t.Fatalf("body = %q, want hi", frame.Body)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("well-formed frame was never delivered")
	}
}

// writeErrConn always fails WriteJSON, exercising deliver's write-error exit.
type writeErrConn struct{}

func (writeErrConn) ReadJSON(any) error  { return io.EOF }
func (writeErrConn) WriteJSON(any) error { return errors.New("write failed") }
func (writeErrConn) Close() error        { return nil }

func TestDeliver_WriteErrorStops(t *testing.T) {
	broker := pubsub.NewMemoryBroker(4)
	gw := NewMessagingGateway(&fakeThreadRepo{m: map[string]*engagemodel.MessageThreadAggregate{}},
		broker, NewTrailAuditRecorder(NewMemoryAuditTrailStore(), "m"), nil)

	ctx := context.Background()
	sub, _ := broker.Subscribe(ctx, threadChannel("t"))

	done := make(chan struct{})
	go func() {
		gw.deliver(ctx, writeErrConn{}, sub, Handshake{UserID: "u", ThreadID: "t"})
		close(done)
	}()
	good, _ := json.Marshal(MessageFrame{Type: MessageType, ThreadID: "t", Body: "hi"})
	_ = broker.Publish(ctx, threadChannel("t"), good)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("deliver did not return after a write failure")
	}
}

func TestHandle_EmptyBodyFrameIsSkipped(t *testing.T) {
	thread := &engagemodel.MessageThreadAggregate{
		ID:                      "thread-1",
		Status:                  engagemodel.MessageThreadStatusOpen,
		ScopedPatientID:         "patient-1",
		ScopedCareTeamMemberIDs: []string{"clinician-1"},
	}
	threads := &fakeThreadRepo{m: map[string]*engagemodel.MessageThreadAggregate{thread.ID: thread}}
	broker := &countingBroker{MemoryBroker: pubsub.NewMemoryBroker(8), subscribed: make(chan string, 2)}
	store := NewMemoryAuditTrailStore()
	gw := NewMessagingGateway(threads, broker, NewTrailAuditRecorder(store, "m"), nil)

	conn := newFakeConn()
	ctx := context.Background()
	go func() {
		_ = gw.Handle(ctx, conn, Handshake{UserID: "patient-1", Role: "patient", ThreadID: "thread-1"})
	}()

	select {
	case <-broker.subscribed:
	case <-time.After(2 * time.Second):
		t.Fatal("gateway never subscribed")
	}

	conn.push(MessageFrame{Body: ""})   // skipped by the read loop
	conn.push(MessageFrame{Body: "hi"}) // posted

	stored, _ := threads.FindByID(ctx, "thread-1")
	assertAuditHas(t, store, "m", "message.secure.posted")
	if stored.PostedMessageCount < 1 {
		t.Fatalf("posted count = %d, want at least 1", stored.PostedMessageCount)
	}
	conn.Close()
}

func TestDeliver_ContextCancellationStops(t *testing.T) {
	broker := pubsub.NewMemoryBroker(4)
	gw := NewMessagingGateway(&fakeThreadRepo{m: map[string]*engagemodel.MessageThreadAggregate{}},
		broker, NewTrailAuditRecorder(NewMemoryAuditTrailStore(), "m"), nil)

	ctx, cancel := context.WithCancel(context.Background())
	sub, _ := broker.Subscribe(ctx, threadChannel("t"))
	conn := newFakeConn()

	done := make(chan struct{})
	go func() {
		gw.deliver(ctx, conn, sub, Handshake{UserID: "u", ThreadID: "t"})
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("deliver did not return after context cancellation")
	}
}

package audit

import (
	"context"
	"errors"
	"testing"
	"time"

	auditmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/auditandcompliance/model"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/integration"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/persistence/mongodb"
)

// stubTrailRepo is an in-memory AuditTrailRepository that lets each test control
// the FindByID/Save outcomes and inspect what was persisted.
type stubTrailRepo struct {
	store map[string]*auditmodel.AuditTrailAggregate

	findErr error
	saveErr error

	saved     *auditmodel.AuditTrailAggregate
	saveCalls int
	findCalls int
}

func newStubTrailRepo() *stubTrailRepo {
	return &stubTrailRepo{store: map[string]*auditmodel.AuditTrailAggregate{}}
}

func (s *stubTrailRepo) FindByID(_ context.Context, id string) (*auditmodel.AuditTrailAggregate, error) {
	s.findCalls++
	if s.findErr != nil {
		return nil, s.findErr
	}
	if a, ok := s.store[id]; ok {
		return a, nil
	}
	return nil, mongodb.ErrDocumentNotFound
}

func (s *stubTrailRepo) Save(_ context.Context, a *auditmodel.AuditTrailAggregate) error {
	s.saveCalls++
	if s.saveErr != nil {
		return s.saveErr
	}
	s.saved = a
	s.store[a.ID] = a
	return nil
}

func fixedClock(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

func TestRecord_FirstEntryStartsFreshTrail(t *testing.T) {
	repo := newStubTrailRepo()
	rec := NewTrailRecorder(repo)
	rec.now = fixedClock(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))

	if err := rec.Record(context.Background(), "alice", "rx-1", "record.read"); err != nil {
		t.Fatalf("Record() error = %v, want nil", err)
	}
	if repo.saveCalls != 1 {
		t.Fatalf("Save calls = %d, want 1", repo.saveCalls)
	}
	if repo.saved == nil {
		t.Fatal("nothing saved")
	}
	if repo.saved.ID != trailPrefix+"rx-1" {
		t.Fatalf("saved trail id = %q, want %q", repo.saved.ID, trailPrefix+"rx-1")
	}
	if got := len(repo.saved.Entries()); got != 1 {
		t.Fatalf("saved entries = %d, want 1", got)
	}
	if repo.saved.Entries()[0].ActorContext != "alice" {
		t.Fatalf("actor = %q, want alice", repo.saved.Entries()[0].ActorContext)
	}
}

func TestRecord_AppendsToExistingTrail(t *testing.T) {
	repo := newStubTrailRepo()
	rec := NewTrailRecorder(repo)
	rec.now = fixedClock(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))

	// Seed an existing trail with one entry.
	if err := rec.Record(context.Background(), "alice", "rx-1", "record.read"); err != nil {
		t.Fatalf("seed Record() error = %v", err)
	}
	firstHead := repo.saved.HeadHash()

	if err := rec.Record(context.Background(), "bob", "rx-1", "record.update"); err != nil {
		t.Fatalf("Record() error = %v, want nil", err)
	}
	if got := len(repo.saved.Entries()); got != 2 {
		t.Fatalf("entries = %d, want 2", got)
	}
	if repo.saved.HeadHash() == firstHead {
		t.Fatal("head hash did not advance after second append")
	}
}

func TestRecord_BlankActorNormalisedToSystem(t *testing.T) {
	repo := newStubTrailRepo()
	rec := NewTrailRecorder(repo)
	rec.now = fixedClock(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))

	if err := rec.Record(context.Background(), "   ", "rx-1", "record.read"); err != nil {
		t.Fatalf("Record() error = %v, want nil", err)
	}
	if got := repo.saved.Entries()[0].ActorContext; got != systemActor {
		t.Fatalf("actor = %q, want %q", got, systemActor)
	}
}

func TestRecord_NoopOnBlankResourceOrAction(t *testing.T) {
	cases := []struct {
		name, resource, action string
	}{
		{"blank resource", "  ", "record.read"},
		{"blank action", "rx-1", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newStubTrailRepo()
			rec := NewTrailRecorder(repo)

			if err := rec.Record(context.Background(), "alice", tc.resource, tc.action); err != nil {
				t.Fatalf("Record() error = %v, want nil", err)
			}
			if repo.findCalls != 0 || repo.saveCalls != 0 {
				t.Fatalf("no-op should not touch repo: find=%d save=%d", repo.findCalls, repo.saveCalls)
			}
		})
	}
}

func TestRecord_FindErrorPropagates(t *testing.T) {
	repo := newStubTrailRepo()
	repo.findErr = errors.New("boom")
	rec := NewTrailRecorder(repo)

	err := rec.Record(context.Background(), "alice", "rx-1", "record.read")
	if err == nil || err.Error() != "boom" {
		t.Fatalf("Record() error = %v, want boom", err)
	}
	if repo.saveCalls != 0 {
		t.Fatalf("Save should not run after find error, calls = %d", repo.saveCalls)
	}
}

func TestRecord_SaveErrorPropagates(t *testing.T) {
	repo := newStubTrailRepo()
	repo.saveErr = errors.New("save failed")
	rec := NewTrailRecorder(repo)
	rec.now = fixedClock(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))

	err := rec.Record(context.Background(), "alice", "rx-1", "record.read")
	if err == nil || err.Error() != "save failed" {
		t.Fatalf("Record() error = %v, want save failed", err)
	}
}

func TestRecord_ExecuteErrorPropagates(t *testing.T) {
	repo := newStubTrailRepo()
	rec := NewTrailRecorder(repo)
	// A zero OccurredAt makes the aggregate reject the append command as an
	// incomplete entry, driving Record's Execute error branch.
	rec.now = fixedClock(time.Time{})

	err := rec.Record(context.Background(), "alice", "rx-1", "record.read")
	if err == nil {
		t.Fatal("Record() = nil, want aggregate Execute error")
	}
	if repo.saveCalls != 0 {
		t.Fatalf("Save should not run after Execute error, calls = %d", repo.saveCalls)
	}
}

func TestNewTrailRecorder_DefaultClock(t *testing.T) {
	rec := NewTrailRecorder(newStubTrailRepo())
	if rec.now == nil {
		t.Fatal("NewTrailRecorder did not set a default clock")
	}
	if rec.now().IsZero() {
		t.Fatal("default clock returned zero time")
	}
}

func TestRecordOutboundAccess_DelegatesToRecord(t *testing.T) {
	repo := newStubTrailRepo()
	rec := NewTrailRecorder(repo)
	rec.now = fixedClock(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))

	access := integration.OutboundAccess{
		ActorContext: "svc-eprescribe",
		ResourceRef:  "rx-99",
		Action:       "eprescribe.submit",
		Destination:  "pharmacy-gateway",
		OccurredAt:   time.Now(),
	}
	if err := rec.RecordOutboundAccess(context.Background(), access); err != nil {
		t.Fatalf("RecordOutboundAccess() error = %v, want nil", err)
	}
	if repo.saved == nil {
		t.Fatal("outbound access was not recorded")
	}
	entry := repo.saved.Entries()[0]
	if entry.ActorContext != "svc-eprescribe" || entry.ResourceRef != "rx-99" || entry.Action != "eprescribe.submit" {
		t.Fatalf("recorded entry = %+v, want outbound access fields", entry)
	}
}

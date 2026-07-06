package temporal

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/persistence/mongodb"
)

// errFactSource fails one or both of its fact lookups; the successful side
// returns no facts.
type errFactSource struct {
	apptErr    error
	revenueErr error
}

func (e errFactSource) AppointmentFacts(_ context.Context, _ string, _, _ time.Time) ([]mongodb.AppointmentFact, error) {
	return nil, e.apptErr
}
func (e errFactSource) RevenueFacts(_ context.Context, _ string, _, _ time.Time) ([]mongodb.RevenueFact, error) {
	return nil, e.revenueErr
}

func rollupInput() RollupActivityInput {
	return RollupActivityInput{
		DashboardID: "dash-1",
		ClinicID:    "clinic-1",
		RangeStart:  "2026-07-01",
		RangeEnd:    "2026-07-07",
		MetricType:  "operational_rollup",
	}
}

func TestComputeRollup_Direct(t *testing.T) {
	ctx := context.Background()

	t.Run("unconfigured", func(t *testing.T) {
		a := &Activities{}
		if _, err := a.ComputeRollup(ctx, rollupInput()); !errors.Is(err, errUnconfigured) {
			t.Fatalf("want errUnconfigured, got %v", err)
		}
	})

	t.Run("bad range start", func(t *testing.T) {
		a := &Activities{Facts: &mongodb.MemFactSource{}, Rollups: &MemRollupStore{}}
		in := rollupInput()
		in.RangeStart = "not-a-date"
		if _, err := a.ComputeRollup(ctx, in); err == nil {
			t.Fatal("expected parse error for bad range start")
		}
	})

	t.Run("bad range end", func(t *testing.T) {
		a := &Activities{Facts: &mongodb.MemFactSource{}, Rollups: &MemRollupStore{}}
		in := rollupInput()
		in.RangeEnd = "not-a-date"
		if _, err := a.ComputeRollup(ctx, in); err == nil {
			t.Fatal("expected parse error for bad range end")
		}
	})

	t.Run("appointment facts error", func(t *testing.T) {
		a := &Activities{Facts: errFactSource{apptErr: errBoom}, Rollups: &MemRollupStore{}}
		if _, err := a.ComputeRollup(ctx, rollupInput()); !errors.Is(err, errBoom) {
			t.Fatalf("want errBoom, got %v", err)
		}
	})

	t.Run("revenue facts error", func(t *testing.T) {
		a := &Activities{Facts: errFactSource{revenueErr: errBoom}, Rollups: &MemRollupStore{}}
		if _, err := a.ComputeRollup(ctx, rollupInput()); !errors.Is(err, errBoom) {
			t.Fatalf("want errBoom, got %v", err)
		}
	})

	t.Run("rollup store error", func(t *testing.T) {
		a := &Activities{Facts: &mongodb.MemFactSource{}, Rollups: errRollupStore{}}
		if _, err := a.ComputeRollup(ctx, rollupInput()); !errors.Is(err, errBoom) {
			t.Fatalf("want errBoom, got %v", err)
		}
	})

	t.Run("dashboard command error", func(t *testing.T) {
		// Empty clinic id makes the dashboard aggregate reject the rollup command.
		a := &Activities{
			Facts:      &mongodb.MemFactSource{},
			Rollups:    &MemRollupStore{},
			Dashboards: &stubDashboardRepo{},
		}
		in := rollupInput()
		in.ClinicID = ""
		if _, err := a.ComputeRollup(ctx, in); err == nil {
			t.Fatal("expected dashboard command error for missing clinic")
		}
	})

	t.Run("dashboard persist error", func(t *testing.T) {
		a := &Activities{
			Facts:      &mongodb.MemFactSource{},
			Rollups:    &MemRollupStore{},
			Dashboards: &stubDashboardRepo{saveErr: errBoom},
		}
		if _, err := a.ComputeRollup(ctx, rollupInput()); !errors.Is(err, errBoom) {
			t.Fatalf("want errBoom, got %v", err)
		}
	})

	t.Run("success with dashboard wired", func(t *testing.T) {
		dash := &stubDashboardRepo{}
		rollups := &MemRollupStore{}
		facts := &mongodb.MemFactSource{
			Appointments: []mongodb.AppointmentFact{
				{ClinicID: "clinic-1", Status: mongodb.FactStatusCompleted, SlotStart: time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)},
			},
			Revenue: []mongodb.RevenueFact{
				{ClinicID: "clinic-1", AmountCents: 5000, CapturedAt: time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)},
			},
		}
		a := &Activities{Facts: facts, Rollups: rollups, Dashboards: dash}
		m, err := a.ComputeRollup(ctx, rollupInput())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.DashboardID != "dash-1" || m.ClinicID != "clinic-1" {
			t.Fatalf("unexpected metrics scope: %+v", m)
		}
		if m.RevenueCents != 5000 || m.TotalSlots != 1 {
			t.Fatalf("unexpected metrics: %+v", m)
		}
		if m.ComputedAt.IsZero() {
			t.Fatal("ComputedAt should be stamped")
		}
		if dash.saved == nil {
			t.Fatal("dashboard aggregate not persisted")
		}
		if len(rollups.All()) != 1 {
			t.Fatalf("want 1 persisted rollup, got %d", len(rollups.All()))
		}
	})
}

// errRollupStore always fails SaveRollup.
type errRollupStore struct{}

func (errRollupStore) SaveRollup(_ context.Context, _ RollupMetrics) error { return errBoom }

func TestParseWindow(t *testing.T) {
	from, to, err := parseWindow("2026-07-01", "2026-07-07")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// End is made inclusive by extending the exclusive upper bound one day.
	wantTo := time.Date(2026, 7, 8, 0, 0, 0, 0, time.UTC)
	if !to.Equal(wantTo) {
		t.Fatalf("to = %v, want %v", to, wantTo)
	}
	if !from.Equal(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("from = %v", from)
	}

	if _, _, err := parseWindow("bad", "2026-07-07"); err == nil {
		t.Fatal("expected start parse error")
	}
	if _, _, err := parseWindow("2026-07-01", "bad"); err == nil {
		t.Fatal("expected end parse error")
	}
}

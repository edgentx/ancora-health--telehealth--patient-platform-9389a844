package temporal

import (
	"math"
	"testing"
	"time"

	"go.temporal.io/sdk/testsuite"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/persistence/mongodb"
)

// TestComputeRollupMetrics exercises the pure rollup arithmetic: utilization,
// no-show rate, and revenue, including the empty-window divide-by-zero guard and
// the exclusion of cancelled slots from the denominators.
func TestComputeRollupMetrics(t *testing.T) {
	tests := []struct {
		name        string
		appts       []mongodb.AppointmentFact
		revenue     []mongodb.RevenueFact
		wantTotal   int
		wantUtil    float64
		wantNoShow  float64
		wantRevenue int64
	}{
		{
			name:      "empty window yields zero rates",
			wantTotal: 0,
		},
		{
			name: "cancelled excluded from denominators",
			appts: []mongodb.AppointmentFact{
				{Status: mongodb.FactStatusOpen},
				{Status: mongodb.FactStatusBooked},
				{Status: mongodb.FactStatusCompleted},
				{Status: mongodb.FactStatusNoShow},
				{Status: mongodb.FactStatusCancelled},
			},
			revenue: []mongodb.RevenueFact{
				{AmountCents: 5000},
				{AmountCents: 2500},
			},
			// total (offered) = open+booked+completed+noShow = 4
			// utilization = scheduled(3)/total(4) = 0.75
			// noShowRate = noShow(1)/scheduled(3) = 0.3333...
			wantTotal:   4,
			wantUtil:    0.75,
			wantNoShow:  1.0 / 3.0,
			wantRevenue: 7500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeRollupMetrics(tt.appts, tt.revenue)
			if got.TotalSlots != tt.wantTotal {
				t.Errorf("TotalSlots = %d, want %d", got.TotalSlots, tt.wantTotal)
			}
			if math.Abs(got.UtilizationRate-tt.wantUtil) > 1e-9 {
				t.Errorf("UtilizationRate = %v, want %v", got.UtilizationRate, tt.wantUtil)
			}
			if math.Abs(got.NoShowRate-tt.wantNoShow) > 1e-9 {
				t.Errorf("NoShowRate = %v, want %v", got.NoShowRate, tt.wantNoShow)
			}
			if got.RevenueCents != tt.wantRevenue {
				t.Errorf("RevenueCents = %d, want %d", got.RevenueCents, tt.wantRevenue)
			}
		})
	}
}

// TestAnalyticsRollupWorkflow verifies the scheduled rollup workflow computes
// metrics from the fact source and persists them for the dashboard.
func TestAnalyticsRollupWorkflow(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()

	day := func(d int) time.Time { return time.Date(2026, 7, d, 12, 0, 0, 0, time.UTC) }
	facts := &mongodb.MemFactSource{
		Appointments: []mongodb.AppointmentFact{
			{ClinicID: "clinic-1", Status: mongodb.FactStatusCompleted, SlotStart: day(2)},
			{ClinicID: "clinic-1", Status: mongodb.FactStatusNoShow, SlotStart: day(3)},
			{ClinicID: "clinic-1", Status: mongodb.FactStatusBooked, SlotStart: day(4)},
			{ClinicID: "clinic-1", Status: mongodb.FactStatusOpen, SlotStart: day(5)},
			// Out of window / other clinic — must be excluded.
			{ClinicID: "clinic-2", Status: mongodb.FactStatusCompleted, SlotStart: day(3)},
			{ClinicID: "clinic-1", Status: mongodb.FactStatusCompleted, SlotStart: day(20)},
		},
		Revenue: []mongodb.RevenueFact{
			{ClinicID: "clinic-1", AmountCents: 4000, CapturedAt: day(2)},
			{ClinicID: "clinic-1", AmountCents: 6000, CapturedAt: day(4)},
		},
	}
	rollups := &MemRollupStore{}
	env.RegisterActivity(&Activities{Facts: facts, Rollups: rollups})

	env.ExecuteWorkflow(AnalyticsRollupWorkflow, RollupWorkflowInput{
		DashboardID: "dash-1",
		ClinicID:    "clinic-1",
		RangeStart:  "2026-07-01",
		RangeEnd:    "2026-07-07",
	})

	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow did not complete")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("workflow error: %v", err)
	}

	var metrics RollupMetrics
	if err := env.GetWorkflowResult(&metrics); err != nil {
		t.Fatalf("decode metrics: %v", err)
	}
	// In-window clinic-1: open+booked+completed+noShow = 4 offered slots.
	if metrics.TotalSlots != 4 {
		t.Fatalf("TotalSlots = %d, want 4", metrics.TotalSlots)
	}
	if metrics.RevenueCents != 10000 {
		t.Fatalf("RevenueCents = %d, want 10000", metrics.RevenueCents)
	}
	if got := rollups.All(); len(got) != 1 {
		t.Fatalf("expected 1 persisted rollup, got %d", len(got))
	}
}

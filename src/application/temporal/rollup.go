package temporal

import (
	"time"

	sdktemporal "go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/persistence/mongodb"
)

// RollupWorkflowInput drives the scheduled analytics rollup workflow: the
// dashboard and clinic to compute for, the inclusive reporting window, and the
// metric label recorded on the dashboard event.
type RollupWorkflowInput struct {
	DashboardID string
	ClinicID    string
	RangeStart  string
	RangeEnd    string
	MetricType  string
}

// rollupActivities is the nil-receiver name handle for the rollup activity (see
// reminderActivities).
var rollupActivities *Activities

// AnalyticsRollupWorkflow computes and persists a clinic's utilization,
// no-show rate, and revenue for a reporting window, feeding the
// AnalyticsDashboard. It is intended to run on a Temporal Schedule (a recurring
// cron trigger configured at deploy time), one execution per reporting window;
// the workflow itself simply drives the compute-and-persist activity so each run
// is a single durable unit.
func AnalyticsRollupWorkflow(ctx workflow.Context, in RollupWorkflowInput) (RollupMetrics, error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: defaultActivityTimeout,
		RetryPolicy: &sdktemporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    defaultMaxAttempts,
		},
	})

	metricType := in.MetricType
	if metricType == "" {
		metricType = "operational_rollup"
	}

	var metrics RollupMetrics
	if err := workflow.ExecuteActivity(ctx, rollupActivities.ComputeRollup, RollupActivityInput{
		DashboardID: in.DashboardID,
		ClinicID:    in.ClinicID,
		RangeStart:  in.RangeStart,
		RangeEnd:    in.RangeEnd,
		MetricType:  metricType,
	}).Get(ctx, &metrics); err != nil {
		return RollupMetrics{}, err
	}
	return metrics, nil
}

// computeRollupMetrics reduces a window of scheduling and billing facts into the
// dashboard's operational metrics. It is a pure function so the arithmetic can
// be exercised hermetically, independent of the fact source or Temporal.
//
// Definitions:
//
//   - TotalSlots counts offered capacity: every slot except cancelled ones,
//     since a cancelled slot freed the provider's time rather than consuming it.
//   - UtilizationRate is claimed capacity over offered capacity —
//     (booked + completed + no-show) / TotalSlots — the share of offered time
//     that was taken.
//   - NoShowRate is no-shows over scheduled slots —
//     no-show / (booked + completed + no-show) — how often a claimed slot was
//     not honored.
//   - RevenueCents sums the captured payment amounts in the window.
//
// A window with no offered slots yields zero rates rather than a divide-by-zero.
func computeRollupMetrics(appts []mongodb.AppointmentFact, revenue []mongodb.RevenueFact) RollupMetrics {
	var open, booked, completed, noShow int
	for _, f := range appts {
		switch f.Status {
		case mongodb.FactStatusOpen:
			open++
		case mongodb.FactStatusBooked:
			booked++
		case mongodb.FactStatusCompleted:
			completed++
		case mongodb.FactStatusNoShow:
			noShow++
		case mongodb.FactStatusCancelled:
			// Cancelled slots freed capacity; excluded from every denominator.
		}
	}

	totalSlots := open + booked + completed + noShow
	scheduled := booked + completed + noShow

	var utilization float64
	if totalSlots > 0 {
		utilization = float64(scheduled) / float64(totalSlots)
	}
	var noShowRate float64
	if scheduled > 0 {
		noShowRate = float64(noShow) / float64(scheduled)
	}

	var revenueCents int64
	for _, r := range revenue {
		revenueCents += r.AmountCents
	}

	return RollupMetrics{
		TotalSlots:      totalSlots,
		UtilizationRate: utilization,
		NoShowRate:      noShowRate,
		RevenueCents:    revenueCents,
	}
}

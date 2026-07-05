package temporal

import (
	"time"

	sdktemporal "go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ResultsReadyWorkflowInput drives the results-ready notification workflow: the
// lab order whose results are ready and the patient to notify.
type ResultsReadyWorkflowInput struct {
	LabOrderID  string
	PatientID   string
	EncounterID string
}

// resultsActivities is the nil-receiver name handle for the results-ready
// activity (see reminderActivities).
var resultsActivities *Activities

// ResultsReadyWorkflow delivers a results-ready event to the patient when lab
// results become available, routing through the notifier. It is a thin durable
// wrapper so the notification survives worker restarts and is retried until it
// is delivered; the send is idempotent on the lab order, so retries never
// double-notify.
func ResultsReadyWorkflow(ctx workflow.Context, in ResultsReadyWorkflowInput) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: defaultActivityTimeout,
		RetryPolicy: &sdktemporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    defaultMaxAttempts,
		},
	})

	return workflow.ExecuteActivity(ctx, resultsActivities.NotifyResultsReady, ResultsReadyInput{
		LabOrderID:  in.LabOrderID,
		PatientID:   in.PatientID,
		EncounterID: in.EncounterID,
	}).Get(ctx, nil)
}

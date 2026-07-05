package temporal

import (
	"time"

	sdktemporal "go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ReminderWorkflowInput drives the appointment reminder workflow: which
// appointment to remind about, its patient and slot, the absolute appointment
// time, and the configured lead times reminders fire at before it (for example
// 24h and 1h).
type ReminderWorkflowInput struct {
	AppointmentID string
	PatientID     string
	TimeSlot      string
	// AppointmentTime is the absolute start time reminders are scheduled
	// relative to.
	AppointmentTime time.Time
	// LeadTimes are the durations before AppointmentTime at which to send a
	// reminder. Each yields one idempotent delivery.
	LeadTimes []time.Duration
}

// reminderActivities references the activity methods by their registered names
// through a nil receiver. Temporal reflects the method to resolve the activity;
// it never invokes it on this nil pointer, so the value is only a name handle.
var reminderActivities *Activities

// ReminderWorkflow schedules and sends appointment reminders at each configured
// lead time. For every lead time it sleeps (durably) until the reminder is due,
// then invokes the send activity. The workflow is idempotent on retry: the send
// activity deduplicates on (appointment, lead time), so a replayed or retried
// reminder is delivered at most once per lead time. A lead time already in the
// past when the workflow runs fires immediately.
func ReminderWorkflow(ctx workflow.Context, in ReminderWorkflowInput) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: defaultActivityTimeout,
		RetryPolicy: &sdktemporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    defaultMaxAttempts,
		},
	})

	for _, lead := range in.LeadTimes {
		dueAt := in.AppointmentTime.Add(-lead)

		// Sleep until the reminder is due. workflow.Now is the deterministic
		// workflow clock; a non-positive delay means the lead time has already
		// passed, so the reminder fires without waiting.
		if delay := dueAt.Sub(workflow.Now(ctx)); delay > 0 {
			if err := workflow.Sleep(ctx, delay); err != nil {
				return err
			}
		}

		leadMinutes := int(lead / time.Minute)
		if err := workflow.ExecuteActivity(ctx, reminderActivities.SendAppointmentReminder, ReminderActivityInput{
			AppointmentID: in.AppointmentID,
			PatientID:     in.PatientID,
			LeadMinutes:   leadMinutes,
			TimeSlot:      in.TimeSlot,
		}).Get(ctx, nil); err != nil {
			return err
		}
	}
	return nil
}

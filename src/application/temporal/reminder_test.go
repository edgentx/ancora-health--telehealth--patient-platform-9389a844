package temporal

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.temporal.io/sdk/testsuite"
)

// flakyNotifier wraps an idempotent Notifier and fails its first failUntil calls
// after delegating the (deduplicated) delivery. It simulates a downstream that
// performs the send but fails to acknowledge, forcing Temporal to retry the
// activity — the exact condition idempotency-on-retry must survive.
type flakyNotifier struct {
	inner     *MemNotifier
	failUntil int
	calls     int
}

func (f *flakyNotifier) Notify(ctx context.Context, n Notification) error {
	f.calls++
	// Delegate first so the idempotent inner notifier records the delivery, then
	// simulate a failed acknowledgement on the early attempts.
	_ = f.inner.Notify(ctx, n)
	if f.calls <= f.failUntil {
		return errors.New("notifier: transient acknowledgement failure")
	}
	return nil
}

// TestReminderWorkflow_HappyPath verifies a reminder is delivered once per
// configured lead time when everything succeeds.
func TestReminderWorkflow_HappyPath(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()

	notifier := &MemNotifier{}
	env.RegisterActivity(&Activities{Notifier: notifier})

	env.ExecuteWorkflow(ReminderWorkflow, ReminderWorkflowInput{
		AppointmentID:   "appt-1",
		PatientID:       "pat-1",
		TimeSlot:        "day-2 09:00",
		AppointmentTime: env.Now().Add(48 * time.Hour),
		LeadTimes:       []time.Duration{24 * time.Hour, time.Hour},
	})

	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow did not complete")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("workflow error: %v", err)
	}
	if got := len(notifier.Delivered()); got != 2 {
		t.Fatalf("expected 2 reminders (one per lead time), got %d", got)
	}
}

// TestReminderWorkflow_RetryIdempotent verifies that when the send activity is
// retried, the reminder is still delivered exactly once — the idempotency the
// acceptance criteria requires on retry.
func TestReminderWorkflow_RetryIdempotent(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()

	notifier := &flakyNotifier{inner: &MemNotifier{}, failUntil: 1}
	env.RegisterActivity(&Activities{Notifier: notifier})

	env.ExecuteWorkflow(ReminderWorkflow, ReminderWorkflowInput{
		AppointmentID:   "appt-2",
		PatientID:       "pat-2",
		TimeSlot:        "day-2 09:00",
		AppointmentTime: env.Now().Add(48 * time.Hour),
		LeadTimes:       []time.Duration{time.Hour},
	})

	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow did not complete")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("workflow error after retry: %v", err)
	}
	if notifier.calls < 2 {
		t.Fatalf("expected the activity to be retried (>=2 calls), got %d", notifier.calls)
	}
	if got := len(notifier.inner.Delivered()); got != 1 {
		t.Fatalf("expected exactly one delivery despite retry, got %d", got)
	}
}

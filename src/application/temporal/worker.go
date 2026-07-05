package temporal

import (
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// Register wires every workflow and the activity set onto a Temporal worker (or
// any registry, including a test environment). Keeping registration in one place
// means the worker process and the tests register an identical surface, so a
// workflow exercised under the test framework is the same one the worker runs.
func Register(r worker.Registry, a *Activities) {
	r.RegisterWorkflow(ReminderWorkflow)
	r.RegisterWorkflow(BillingEligibilitySagaWorkflow)
	r.RegisterWorkflow(ResultsReadyWorkflow)
	r.RegisterWorkflow(AnalyticsRollupWorkflow)
	r.RegisterActivity(a)
}

// NewWorker builds a worker polling the configured task queue with every
// workflow and activity registered.
func NewWorker(c client.Client, cfg Config, a *Activities) worker.Worker {
	w := worker.New(c, cfg.TaskQueue, worker.Options{})
	Register(w, a)
	return w
}

// Run dials Temporal with the deploy-time config, starts a worker on the task
// queue, and blocks until interrupted (SIGINT/SIGTERM). It is the entry point a
// deployable worker process calls. The returned error is non-nil when the
// frontend cannot be reached or the worker stops abnormally.
func Run(cfg Config, a *Activities) error {
	c, err := client.Dial(client.Options{
		HostPort:  cfg.HostPort,
		Namespace: cfg.Namespace,
	})
	if err != nil {
		return err
	}
	defer c.Close()

	w := NewWorker(c, cfg, a)
	return w.Run(worker.InterruptCh())
}

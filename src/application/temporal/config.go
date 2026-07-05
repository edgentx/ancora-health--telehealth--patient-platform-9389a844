// Package temporal hosts the Temporal Go SDK worker(s) for the platform: the
// durable workflows and activities that drive appointment reminders,
// results-ready notifications, the billing/eligibility saga, and the scheduled
// analytics rollups feeding the AnalyticsDashboard.
//
// The package follows the codebase's hexagonal layering. Workflows are the
// deterministic orchestration layer and may only call other workflows,
// activities, and the deterministic workflow APIs (timers, sleeps). Every
// side effect — repository reads/writes (S-69/S-70), external adapter calls
// (S-72), and notification fan-out through the realtime gateway — lives behind
// a narrow port and is performed from an Activity, so the workflow history can
// be replayed deterministically.
package temporal

import (
	"os"
	"time"
)

// Environment variables read by ConfigFromEnv. They are the deploy-time
// Temporal connection settings the worker process binds to.
const (
	envHostPort  = "TEMPORAL_HOSTPORT"
	envNamespace = "TEMPORAL_NAMESPACE"
	envTaskQueue = "TEMPORAL_TASK_QUEUE"
)

// Defaults applied when the corresponding environment variable is unset. They
// match a local Temporal dev server so the worker runs out of the box.
const (
	// DefaultHostPort is the Temporal frontend address a worker dials when
	// TEMPORAL_HOSTPORT is not configured.
	DefaultHostPort = "127.0.0.1:7233"
	// DefaultNamespace is the Temporal namespace used when TEMPORAL_NAMESPACE is
	// not configured.
	DefaultNamespace = "default"
	// DefaultTaskQueue is the task queue the worker polls, and the queue callers
	// start these workflows on, when TEMPORAL_TASK_QUEUE is not configured.
	DefaultTaskQueue = "ancora-workers"
)

// Config captures the deploy-time settings a worker process binds to: where to
// reach the Temporal frontend, which namespace to operate in, and which task
// queue to poll. Callers starting these workflows must target the same queue.
type Config struct {
	// HostPort is the Temporal frontend gRPC address, e.g. "temporal:7233".
	HostPort string
	// Namespace is the Temporal namespace the worker operates in.
	Namespace string
	// TaskQueue is the queue the worker polls for workflow and activity tasks.
	TaskQueue string
}

// ConfigFromEnv reads the Temporal worker configuration from the environment,
// falling back to the local-dev defaults for any unset value so the process is
// always launchable. It never errors; an unreachable frontend surfaces later,
// when the worker dials.
func ConfigFromEnv() Config {
	return Config{
		HostPort:  envOr(envHostPort, DefaultHostPort),
		Namespace: envOr(envNamespace, DefaultNamespace),
		TaskQueue: envOr(envTaskQueue, DefaultTaskQueue),
	}
}

// envOr returns the value of the named environment variable, or def when it is
// unset or empty.
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// Activity execution tuning shared by the workflows. Every activity is bounded
// by a start-to-close timeout and retried with capped exponential backoff, so a
// transient repository or adapter failure is retried transparently while a
// genuinely stuck activity is not left running forever.
const (
	// defaultActivityTimeout bounds a single activity attempt.
	defaultActivityTimeout = 30 * time.Second
	// defaultMaxAttempts caps how many times an activity is retried before the
	// workflow observes the failure (and, for the saga, compensates).
	defaultMaxAttempts = 5
)

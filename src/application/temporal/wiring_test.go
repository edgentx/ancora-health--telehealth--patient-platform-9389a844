package temporal

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/testsuite"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/pubsub"
)

// --- config -----------------------------------------------------------------

func TestConfigFromEnv(t *testing.T) {
	t.Run("defaults when unset", func(t *testing.T) {
		t.Setenv(envHostPort, "")
		t.Setenv(envNamespace, "")
		t.Setenv(envTaskQueue, "")
		cfg := ConfigFromEnv()
		if cfg.HostPort != DefaultHostPort || cfg.Namespace != DefaultNamespace || cfg.TaskQueue != DefaultTaskQueue {
			t.Fatalf("expected defaults, got %+v", cfg)
		}
	})

	t.Run("overrides from env", func(t *testing.T) {
		t.Setenv(envHostPort, "temporal:7233")
		t.Setenv(envNamespace, "ns")
		t.Setenv(envTaskQueue, "queue")
		cfg := ConfigFromEnv()
		if cfg.HostPort != "temporal:7233" || cfg.Namespace != "ns" || cfg.TaskQueue != "queue" {
			t.Fatalf("expected overrides, got %+v", cfg)
		}
	})
}

func TestEnvOr(t *testing.T) {
	t.Setenv("TMP_ENVOR", "")
	if got := envOr("TMP_ENVOR", "fallback"); got != "fallback" {
		t.Fatalf("empty => %q, want fallback", got)
	}
	t.Setenv("TMP_ENVOR", "set")
	if got := envOr("TMP_ENVOR", "fallback"); got != "set" {
		t.Fatalf("set => %q, want set", got)
	}
}

// --- BrokerNotifier ---------------------------------------------------------

func TestBrokerNotifier_Notify(t *testing.T) {
	broker := pubsub.NewMemoryBroker(8)
	ctx := context.Background()

	sub, err := broker.Subscribe(ctx, notifyChannel("pat-1"))
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer sub.Close()

	n := NewBrokerNotifier(broker)
	note := Notification{
		Kind:        "appointment_reminder",
		RecipientID: "pat-1",
		Subject:     "Upcoming",
		Body:        "hi",
		DedupeKey:   "reminder:a1:60",
	}
	if err := n.Notify(ctx, note); err != nil {
		t.Fatalf("notify: %v", err)
	}

	msg := <-sub.C()
	var got Notification
	if err := json.Unmarshal(msg.Payload, &got); err != nil {
		t.Fatalf("unmarshal frame: %v", err)
	}
	if got != note {
		t.Fatalf("delivered %+v, want %+v", got, note)
	}
	if msg.Channel != "notify:pat-1" {
		t.Fatalf("channel = %q", msg.Channel)
	}
}

func TestNotifyChannel(t *testing.T) {
	if got := notifyChannel("u1"); got != "notify:u1" {
		t.Fatalf("notifyChannel = %q", got)
	}
}

// --- NotifyResultsReady activity + workflow ---------------------------------

func TestNotifyResultsReady(t *testing.T) {
	ctx := context.Background()

	t.Run("unconfigured notifier", func(t *testing.T) {
		a := &Activities{}
		if err := a.NotifyResultsReady(ctx, ResultsReadyInput{LabOrderID: "lab-1"}); !errors.Is(err, errUnconfigured) {
			t.Fatalf("want errUnconfigured, got %v", err)
		}
	})

	t.Run("delivers once", func(t *testing.T) {
		notifier := &MemNotifier{}
		a := &Activities{Notifier: notifier}
		if err := a.NotifyResultsReady(ctx, ResultsReadyInput{LabOrderID: "lab-1", PatientID: "pat-1"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		d := notifier.Delivered()
		if len(d) != 1 || d[0].DedupeKey != "results_ready:lab-1" {
			t.Fatalf("unexpected delivery: %+v", d)
		}
	})
}

func TestResultsReadyWorkflow(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()

	notifier := &MemNotifier{}
	env.RegisterActivity(&Activities{Notifier: notifier})

	env.ExecuteWorkflow(ResultsReadyWorkflow, ResultsReadyWorkflowInput{
		LabOrderID:  "lab-1",
		PatientID:   "pat-1",
		EncounterID: "enc-1",
	})

	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow did not complete")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("workflow error: %v", err)
	}
	if got := len(notifier.Delivered()); got != 1 {
		t.Fatalf("want 1 delivery, got %d", got)
	}
}

// --- worker registration ----------------------------------------------------

// TestNewWorkerRegistersEverything builds a worker over a lazy (non-dialing)
// client, exercising NewWorker and Register without a live Temporal frontend.
func TestNewWorkerRegistersEverything(t *testing.T) {
	c, err := client.NewLazyClient(client.Options{HostPort: DefaultHostPort})
	if err != nil {
		t.Fatalf("NewLazyClient: %v", err)
	}
	defer c.Close()

	w := NewWorker(c, Config{TaskQueue: DefaultTaskQueue}, &Activities{})
	if w == nil {
		t.Fatal("expected a worker")
	}
}

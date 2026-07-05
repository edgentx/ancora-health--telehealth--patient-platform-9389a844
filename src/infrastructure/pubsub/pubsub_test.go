package pubsub

import (
	"context"
	"testing"
	"time"
)

// TestMemoryBrokerFansOutToEverySubscriber proves a single payload published to
// a channel reaches every subscriber on that channel — the property that lets a
// shared broker model Redis pub/sub across replicas.
func TestMemoryBrokerFansOutToEverySubscriber(t *testing.T) {
	broker := NewMemoryBroker(8)
	ctx := context.Background()

	// Two subscriptions on the same channel stand in for two replicas each with a
	// connected client.
	subA, err := broker.Subscribe(ctx, "thread:1")
	if err != nil {
		t.Fatalf("subscribe A: %v", err)
	}
	defer subA.Close()
	subB, err := broker.Subscribe(ctx, "thread:1")
	if err != nil {
		t.Fatalf("subscribe B: %v", err)
	}
	defer subB.Close()

	// A subscription on a different channel must not receive the payload.
	other, err := broker.Subscribe(ctx, "thread:2")
	if err != nil {
		t.Fatalf("subscribe other: %v", err)
	}
	defer other.Close()

	if err := broker.Publish(ctx, "thread:1", []byte("hello")); err != nil {
		t.Fatalf("publish: %v", err)
	}

	for name, sub := range map[string]Subscription{"A": subA, "B": subB} {
		select {
		case m := <-sub.C():
			if string(m.Payload) != "hello" {
				t.Fatalf("subscriber %s: got %q, want %q", name, m.Payload, "hello")
			}
		case <-time.After(time.Second):
			t.Fatalf("subscriber %s: no message delivered", name)
		}
	}

	select {
	case m := <-other.C():
		t.Fatalf("unrelated channel received %q", m.Payload)
	case <-time.After(50 * time.Millisecond):
	}
}

// TestMemoryBrokerCloseStopsDelivery verifies a closed subscription no longer
// receives published payloads.
func TestMemoryBrokerCloseStopsDelivery(t *testing.T) {
	broker := NewMemoryBroker(8)
	ctx := context.Background()

	sub, err := broker.Subscribe(ctx, "c")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if err := sub.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	// Double close is a no-op, not a panic.
	if err := sub.Close(); err != nil {
		t.Fatalf("double close: %v", err)
	}

	if err := broker.Publish(ctx, "c", []byte("x")); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if _, ok := <-sub.C(); ok {
		t.Fatal("expected closed channel to yield no message")
	}
}

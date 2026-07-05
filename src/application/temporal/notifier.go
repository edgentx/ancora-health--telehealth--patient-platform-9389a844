package temporal

import (
	"context"
	"encoding/json"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/pubsub"
)

// BrokerNotifier is the production Notifier. It fans a notification out over the
// realtime pub/sub broker (the same broker the WebSocket gateways use, S-73), so
// a notification emitted from a worker replica reaches whichever gateway replica
// the recipient is connected to. The recipient's client subscribes to its own
// per-user channel and renders the delivered frame.
//
// The broker is fire-and-forget, so idempotency is carried in the payload's
// DedupeKey: a delivered-more-than-once notification is collapsed by the client
// on the key, matching the at-least-once activity contract.
type BrokerNotifier struct {
	broker pubsub.Broker
}

// NewBrokerNotifier wires a notifier over a pub/sub broker.
func NewBrokerNotifier(broker pubsub.Broker) *BrokerNotifier {
	return &BrokerNotifier{broker: broker}
}

// notifyChannel is the per-recipient fan-out channel a notification is published
// on. Every gateway replica with the recipient connected subscribes to it.
func notifyChannel(recipientID string) string { return "notify:" + recipientID }

// Notify publishes the notification as a JSON frame on the recipient's channel.
func (n *BrokerNotifier) Notify(ctx context.Context, note Notification) error {
	payload, err := json.Marshal(note)
	if err != nil {
		return err
	}
	return n.broker.Publish(ctx, notifyChannel(note.RecipientID), payload)
}

// Compile-time assertion that BrokerNotifier satisfies the Notifier port.
var _ Notifier = (*BrokerNotifier)(nil)

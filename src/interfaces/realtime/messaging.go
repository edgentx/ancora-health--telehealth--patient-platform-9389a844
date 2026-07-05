package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	engagemodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/patientengagement/model"
	engagerepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/patientengagement/repository"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/pubsub"
)

// ErrThreadAccessDenied is returned when a messaging connection cannot be tied to
// a thread the authenticated principal participates in: no principal, no thread,
// an unknown thread, or a principal outside the thread's participant set.
var ErrThreadAccessDenied = errors.New("realtime: connection is not a participant of the message thread")

// MessageFrame is one secure-messaging frame. Inbound from a client it carries
// only Body; the gateway stamps ThreadID and AuthorID from the trusted handshake
// before persisting and fanning out, so a client cannot spoof the author or post
// to a thread it is not scoped to.
type MessageFrame struct {
	Type     string `json:"type"`
	ThreadID string `json:"threadId"`
	AuthorID string `json:"authorId"`
	Body     string `json:"body"`
}

// MessageType is the frame type for a delivered secure message.
const MessageType = "message"

// threadChannel is the pub/sub channel a thread's messages fan out on. Every
// replica with a participant connected to the thread subscribes to it, so a post
// on any replica reaches connections on all of them.
func threadChannel(threadID string) string { return "thread:" + threadID }

// MessagingGateway is the secure-messaging WebSocket gateway. It authenticates a
// connection from the trusted handshake, scopes it to a MessageThread the
// principal participates in, persists posted messages through the MessageThread
// repository, and fans out via the pub/sub broker so participants on every
// replica receive them. Thread open/close and PHI message access are recorded to
// the audit trail.
type MessagingGateway struct {
	threads engagerepo.MessageThreadRepository
	broker  pubsub.Broker
	audit   AuditRecorder
	now     func() time.Time
}

// NewMessagingGateway wires a messaging gateway. now defaults to time.Now when
// nil, injectable for deterministic audit timestamps under test.
func NewMessagingGateway(
	threads engagerepo.MessageThreadRepository,
	broker pubsub.Broker,
	audit AuditRecorder,
	now func() time.Time,
) *MessagingGateway {
	if now == nil {
		now = time.Now
	}
	return &MessagingGateway{threads: threads, broker: broker, audit: audit, now: now}
}

// Handle drives one messaging connection end to end: it authenticates and scopes
// the connection to a thread, subscribes to the thread's fan-out channel to
// deliver messages posted anywhere in the fleet, and persists-then-publishes each
// message the client posts until the connection closes. It blocks until the
// connection ends and always closes conn.
func (g *MessagingGateway) Handle(ctx context.Context, conn Conn, h Handshake) error {
	defer conn.Close()

	if h.UserID == "" || h.ThreadID == "" {
		_ = conn.WriteJSON(MessageFrame{Type: SignalError, Body: ErrThreadAccessDenied.Error()})
		return ErrThreadAccessDenied
	}

	thread, err := g.threads.FindByID(ctx, h.ThreadID)
	if err != nil || thread == nil || !threadParticipant(thread, h.UserID) {
		_ = g.audit.Record(ctx, auditActor(h.UserID, h.Role), "thread:"+h.ThreadID, "message.thread.access.denied", g.now())
		_ = conn.WriteJSON(MessageFrame{Type: SignalError, Body: ErrThreadAccessDenied.Error()})
		return ErrThreadAccessDenied
	}

	sub, err := g.broker.Subscribe(ctx, threadChannel(h.ThreadID))
	if err != nil {
		return err
	}
	defer sub.Close()

	_ = g.audit.Record(ctx, auditActor(h.UserID, h.Role), "thread:"+h.ThreadID, "message.thread.opened", g.now())
	defer func() {
		_ = g.audit.Record(context.WithoutCancel(ctx), auditActor(h.UserID, h.Role), "thread:"+h.ThreadID, "message.thread.closed", g.now())
	}()

	// The pump is the connection's sole writer: it delivers fanned-out messages
	// from the broker to this client, recording each as a PHI access.
	go g.deliver(ctx, conn, sub, h)

	for {
		var frame MessageFrame
		if err := conn.ReadJSON(&frame); err != nil {
			return err
		}
		if frame.Body == "" {
			continue
		}
		if err := g.post(ctx, h, frame.Body); err != nil {
			// A rejected post (empty body, broken invariant) closes the connection
			// so the client resynchronizes rather than silently dropping content.
			return err
		}
	}
}

// post persists a posted message through the MessageThread repository, records
// the PHI write to the audit trail, and publishes it to the thread's fan-out
// channel. Persisting first means a message is durably recorded before any
// replica delivers it.
func (g *MessagingGateway) post(ctx context.Context, h Handshake, body string) error {
	thread, err := g.threads.FindByID(ctx, h.ThreadID)
	if err != nil || thread == nil {
		return ErrThreadAccessDenied
	}
	if _, err := thread.Execute(engagemodel.PostSecureMessageCmd{
		ThreadId: h.ThreadID,
		AuthorId: h.UserID,
		Body:     body,
	}); err != nil {
		return err
	}
	if err := g.threads.Save(ctx, thread); err != nil {
		return err
	}

	_ = g.audit.Record(ctx, auditActor(h.UserID, h.Role), "thread:"+h.ThreadID, "message.secure.posted", g.now())

	payload, err := json.Marshal(MessageFrame{
		Type:     MessageType,
		ThreadID: h.ThreadID,
		AuthorID: h.UserID,
		Body:     body,
	})
	if err != nil {
		return err
	}
	return g.broker.Publish(ctx, threadChannel(h.ThreadID), payload)
}

// deliver forwards fanned-out messages from the broker subscription to the
// connection, recording each delivery as a PHI access. It is the connection's
// only writer, so no write serialization is needed.
func (g *MessagingGateway) deliver(ctx context.Context, conn Conn, sub pubsub.Subscription, h Handshake) {
	for {
		select {
		case <-ctx.Done():
			return
		case m, ok := <-sub.C():
			if !ok {
				return
			}
			var frame MessageFrame
			if err := json.Unmarshal(m.Payload, &frame); err != nil {
				continue
			}
			_ = g.audit.Record(ctx, auditActor(h.UserID, h.Role), "thread:"+h.ThreadID, "message.phi.accessed", g.now())
			if err := conn.WriteJSON(frame); err != nil {
				return
			}
		}
	}
}

// threadParticipant reports whether userID is the thread's scoped patient or one
// of its care-team participants — the set permitted to post to or read it.
func threadParticipant(thread *engagemodel.MessageThreadAggregate, userID string) bool {
	if userID == thread.ScopedPatientID {
		return true
	}
	for _, id := range thread.ScopedCareTeamMemberIDs {
		if id == userID {
			return true
		}
	}
	return false
}

// auditActor renders the audit actor context from the connection's principal and
// role. Role-qualified when known, bare principal otherwise.
func auditActor(userID, role string) string {
	if role == "" {
		return userID
	}
	return role + ":" + userID
}

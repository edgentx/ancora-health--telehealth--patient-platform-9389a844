// Command realtime is the WebSocket gateway entrypoint for the Ancora Health
// platform. It exposes the S-73 WebRTC signaling gateway (/signaling) and the
// secure-messaging gateway (/messaging) alongside the standard operational
// surface (/health, /ready, /metrics, /version). It is one of the three
// selectable backend entrypoints the container image builds (api, realtime,
// worker); the image runs it via the `realtime` build target.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/locking"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/persistence/mongodb"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/pubsub"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/interfaces/realtime"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/platform"
)

// coturnTTL is the ephemeral TURN credential lifetime advertised to peers when
// COTURN_TTL is not set; one hour is the conventional coturn REST TTL.
const coturnTTL = time.Hour

func main() {
	healthcheck := flag.Bool("healthcheck", false, "probe /ready on the local listen address and exit")
	flag.Parse()
	if *healthcheck {
		if err := platform.SelfCheck(); err != nil {
			log.Fatalf("healthcheck failed: %v", err)
		}
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Shared in-memory backing store keeps the gateway bootable without a live
	// MongoDB or Redis, while wiring the real gateway types. Point MONGODB_URI and
	// a Redis broker in for production fan-out across replicas.
	store := mongodb.NewMemStore()
	locker := locking.NewMemorySlotLocker()

	auditStore := realtime.NewMemoryAuditTrailStore()
	audit := realtime.NewTrailAuditRecorder(auditStore, "")

	authorizer := realtime.NewAppointmentSessionAuthorizer(
		mongodb.NewAppointmentRepository(store, store, locker, ""),
		nil, // no care-relationship store: handshakes that reference one are refused
	)
	coturn := realtime.NewCoturnIssuer(os.Getenv("COTURN_SECRET"), coturnURIs(), coturnTTL)
	signaling := realtime.NewSignalingGateway(authorizer, coturn, audit, nil)

	broker := pubsub.NewMemoryBroker(0)
	messaging := realtime.NewMessagingGateway(mongodb.NewMessageThreadRepository(store), broker, audit, nil)

	mux := platform.NewOpsMux(platform.NewMetricsRegistry(), nil)
	mux.Handle("/signaling", signaling.SignalingHTTPHandler())
	mux.Handle("/messaging", messaging.MessagingHTTPHandler())

	addr := platform.ListenAddr()
	platform.LogStartup("ancora-realtime", addr)
	if err := platform.Serve(ctx, addr, mux); err != nil {
		log.Fatalf("realtime gateway error: %v", err)
	}
}

// coturnURIs parses the comma-separated TURN/STUN URIs to advertise from
// COTURN_URIS, returning nil (no advertised servers) when unset.
func coturnURIs() []string {
	raw := strings.TrimSpace(os.Getenv("COTURN_URIS"))
	if raw == "" {
		return nil
	}
	var uris []string
	for _, u := range strings.Split(raw, ",") {
		if u = strings.TrimSpace(u); u != "" {
			uris = append(uris, u)
		}
	}
	return uris
}

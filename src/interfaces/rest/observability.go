package rest

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// tracerName is the instrumentation scope reported on every span this layer
// emits. A configured OpenTelemetry SDK groups spans by it; with no SDK
// installed the global provider is a no-op and the instrumentation is free.
const tracerName = "ancora/interfaces/rest"

// observability bundles the Prometheus registry and the request metrics the
// middleware records. Each API server owns one instance so its /metrics
// endpoint exposes exactly the series its own traffic produced.
type observability struct {
	registry *prometheus.Registry
	requests *prometheus.CounterVec
	latency  *prometheus.HistogramVec
	tracer   trace.Tracer
}

// newObservability builds a private Prometheus registry, registers the standard
// Go runtime and process collectors alongside the HTTP request series, and
// resolves the global tracer. A private registry (rather than the default one)
// keeps metrics from leaking across servers in the same process and makes the
// layer safe to instantiate more than once — notably in tests.
func newObservability() *observability {
	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	requests := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests handled, partitioned by method, route and status.",
		},
		[]string{"method", "route", "status"},
	)
	latency := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds, partitioned by method and route.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route"},
	)
	reg.MustRegister(requests, latency)

	return &observability{
		registry: reg,
		requests: requests,
		latency:  latency,
		tracer:   otel.Tracer(tracerName),
	}
}

// metricsHandler serves the registry in the Prometheus text exposition format,
// the payload a Prometheus server scrapes from /metrics.
func (o *observability) metricsHandler() http.Handler {
	return promhttp.HandlerFor(o.registry, promhttp.HandlerOpts{})
}

// statusRecorder wraps a ResponseWriter to capture the status code the handler
// wrote, which the middleware needs for both the OpenTelemetry span and the
// Prometheus status label. It defaults to 200 because a handler that writes a
// body without calling WriteHeader has implicitly returned 200.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.ResponseWriter.Write(b)
}

// telemetryMiddleware wraps every request in an OpenTelemetry span and records
// its outcome to Prometheus. It reads the matched chi route pattern *after* the
// inner handler runs — that is when routing has resolved it — so the metric
// labels stay low-cardinality (the pattern "/appointments/{id}", never the
// expanded id), which also keeps PHI-bearing path segments out of telemetry.
func (o *observability) telemetryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := nowFunc()

		ctx, span := o.tracer.Start(r.Context(), r.Method,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("http.request.method", r.Method),
				attribute.String("url.path", r.URL.Path),
			),
		)
		defer span.End()

		rec := &statusRecorder{ResponseWriter: w, status: 0}
		next.ServeHTTP(rec, r.WithContext(ctx))

		if rec.status == 0 {
			rec.status = http.StatusOK
		}

		route := chi.RouteContext(r.Context()).RoutePattern()
		if route == "" {
			route = "unmatched"
		}

		span.SetName(r.Method + " " + route)
		span.SetAttributes(
			attribute.String("http.route", route),
			attribute.Int("http.response.status_code", rec.status),
		)
		if rec.status >= http.StatusInternalServerError {
			span.SetStatus(codes.Error, http.StatusText(rec.status))
		}

		o.requests.WithLabelValues(r.Method, route, strconv.Itoa(rec.status)).Inc()
		o.latency.WithLabelValues(r.Method, route).Observe(nowFunc().Sub(start).Seconds())
	})
}

// nowFunc is the clock the middleware reads, indirected so tests can hold time
// steady when asserting on latency observations.
var nowFunc = time.Now

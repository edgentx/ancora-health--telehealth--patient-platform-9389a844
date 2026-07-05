# syntax=docker/dockerfile:1
#
# Multi-stage build for the Ancora Health Go backend (S-76).
#
# Three selectable entrypoints ship as distinct build targets sharing one build
# base: `api` (cmd/server), `realtime` (cmd/realtime) and `worker` (cmd/worker).
# `api` is the default target, so `docker build .` produces the API image; the
# others are built with `--target realtime` / `--target worker`.
#
#   docker build -t ancora/api      .
#   docker build -t ancora/realtime --target realtime .
#   docker build -t ancora/worker   --target worker   .
#
# Build metadata is injected at link time from --build-arg VERSION / COMMIT /
# BUILD_DATE and surfaced at runtime on /version and in the startup log line.
#
#   docker build -t ancora/api \
#     --build-arg VERSION=$(git describe --tags --always) \
#     --build-arg COMMIT=$(git rev-parse --short HEAD) \
#     --build-arg BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ) .
#
# The runtime stage is distroless/static:nonroot — no shell, no package manager,
# runs as an unprivileged user — so the readiness probe is the app's own
# `-healthcheck` self-probe rather than a shelled-out curl.

########################  Dependency + source base  ########################
# Build on the latest Go 1.25 patch: the go.mod `go` directive pins the language
# version, while the toolchain supplies the compiled-in standard library.
# Building on a current patch release is what keeps the binary free of Go stdlib
# CVEs (a Trivy HIGH/CRITICAL gate would otherwise flag an older stdlib). The
# floating -bookworm tag picks up future patches on rebuild.
FROM golang:1.25-bookworm AS base
WORKDIR /src

# The Go module path; ldflags target the platform package under it.
ENV MODULE=github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844
ENV CGO_ENABLED=0 GOOS=linux GOFLAGS=-mod=mod

# Module layer: cached independently of source so a code-only change does not
# re-download dependencies.
COPY go.mod go.sum ./
RUN go mod download

# Application source.
COPY . .

########################  Per-entrypoint build stages  ########################
# Each stage statically links one entrypoint from the shared `base` layer, so
# the go.mod download and source COPY are cached once and reused by all three.
# -trimpath keeps builds reproducible; -s -w drops the symbol table and DWARF to
# shrink the binary.
#
# The build-metadata ARGs are re-declared in every build stage on purpose: an
# ARG is scoped to the stage that declares it and is NOT inherited across a FROM,
# so declaring them here (not in `base`) is what actually threads --build-arg
# values into each ldflags -X.

FROM base AS build-api
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown
RUN go build -trimpath \
    -ldflags "-s -w -X ${MODULE}/src/platform.Version=${VERSION} -X ${MODULE}/src/platform.Commit=${COMMIT} -X ${MODULE}/src/platform.Date=${BUILD_DATE}" \
    -o /out/app ./cmd/server

FROM base AS build-realtime
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown
RUN go build -trimpath \
    -ldflags "-s -w -X ${MODULE}/src/platform.Version=${VERSION} -X ${MODULE}/src/platform.Commit=${COMMIT} -X ${MODULE}/src/platform.Date=${BUILD_DATE}" \
    -o /out/app ./cmd/realtime

FROM base AS build-worker
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown
RUN go build -trimpath \
    -ldflags "-s -w -X ${MODULE}/src/platform.Version=${VERSION} -X ${MODULE}/src/platform.Commit=${COMMIT} -X ${MODULE}/src/platform.Date=${BUILD_DATE}" \
    -o /out/app ./cmd/worker

########################  Runtime images (one per entrypoint)  ################
# distroless/static:nonroot — minimal, no shell, runs as uid 65532. The single
# static binary is the whole userland. HEALTHCHECK runs the binary's own
# in-process /ready probe. PORT (default 8000) selects the listen port at run
# time. Kubernetes uses its own probes; the HEALTHCHECK serves docker/compose.

FROM gcr.io/distroless/static-debian12:nonroot AS realtime
COPY --from=build-realtime /out/app /app
USER nonroot:nonroot
EXPOSE 8000
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/app", "-healthcheck"]
ENTRYPOINT ["/app"]

FROM gcr.io/distroless/static-debian12:nonroot AS worker
COPY --from=build-worker /out/app /app
USER nonroot:nonroot
EXPOSE 8000
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/app", "-healthcheck"]
ENTRYPOINT ["/app"]

# api is defined last so it is the default target for `docker build .`.
FROM gcr.io/distroless/static-debian12:nonroot AS api
COPY --from=build-api /out/app /app
USER nonroot:nonroot
EXPOSE 8000
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/app", "-healthcheck"]
ENTRYPOINT ["/app"]

# Backend container image (S-76)

The Go backend ships as a multi-stage container image with three selectable
entrypoints. The `Dockerfile` at the repo root builds a statically linked binary
in a `golang:1.25-bookworm` stage and copies it into a minimal, non-root
`gcr.io/distroless/static-debian12:nonroot` runtime stage.

## Entrypoints (build targets)

| Target     | Command        | Serves                                                    |
|------------|----------------|-----------------------------------------------------------|
| `api`      | `cmd/server`   | Versioned REST API under `/api/v1` + ops endpoints        |
| `realtime` | `cmd/realtime` | WebRTC signaling (`/signaling`) + messaging (`/messaging`)|
| `worker`   | `cmd/worker`   | Background pub/sub consumer, ops endpoints only           |

`api` is the default target, so `docker build .` produces the API image. Select
the others with `--target`:

```bash
make docker            # build all three (ancora/backend-{api,realtime,worker})
make docker-realtime   # or a single target

# or directly:
docker build -t ancora/backend-api      .
docker build -t ancora/backend-realtime --target realtime .
docker build -t ancora/backend-worker   --target worker   .
```

Every image is the same shape: one static binary, runs as uid 65532, listens on
`:8000` (override with `PORT`), and declares a `HEALTHCHECK`.

## Build metadata

Version, commit and build date are injected at link time and surfaced at runtime
on `GET /version` and in the startup log line:

```bash
docker build -t ancora/backend-api \
  --build-arg VERSION=$(git describe --tags --always) \
  --build-arg COMMIT=$(git rev-parse --short HEAD) \
  --build-arg BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ) .
```

`make docker` fills these in from git automatically.

## Operational endpoints

Every entrypoint exposes:

| Endpoint   | Purpose                                                              |
|------------|---------------------------------------------------------------------|
| `/health`  | Liveness — the process is up.                                        |
| `/ready`   | Readiness — able to serve. The `api` probes MongoDB (503 until `MONGODB_URI` is set and reachable); `realtime`/`worker` are dependency-free and report ready immediately. |
| `/metrics` | Prometheus exposition (Go runtime + process + per-service series).   |
| `/version` | Build identity (version, commit, date, Go toolchain).               |

### Healthcheck on a shell-less image

distroless has no shell, so the `HEALTHCHECK` runs the binary's own in-process
probe — `/app -healthcheck` performs an HTTP `GET /ready` against the local
listen address and exits non-zero if the service is not ready. No `curl`/`wget`
in the image.

```bash
docker run -d -p 8000:8000 -e MONGODB_URI=mongodb://... ancora/backend-api
curl -fsS localhost:8000/ready
curl -fsS localhost:8000/metrics | head
curl -fsS localhost:8000/version
```

## Vulnerability scanning

The image must scan free of HIGH/CRITICAL findings. `make scan` builds the API
image and runs Trivy against it, failing on any HIGH/CRITICAL:

```bash
make scan
# or, without a local trivy install:
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
  aquasec/trivy:latest image --severity HIGH,CRITICAL --exit-code 1 \
  --ignorefile .trivyignore ancora/backend-api:latest
```

Two things keep the image clean:

1. **Build on a current Go patch release.** The go.mod `go` directive pins the
   language version; the builder toolchain supplies the compiled-in standard
   library. Building on the latest `golang:1.25` patch keeps Go stdlib CVEs out
   of the binary.
2. **Keep module dependencies patched.** Findings in Go module dependencies are
   fixed by bumping the module (`go get -u <module>` + `go mod tidy`), not
   waived. Prefer a fix over a waiver.

Documented waivers, if ever unavoidable, live in `.trivyignore` — each must name
the CVE, the affected module, why it is not exploitable here, and a review date.
The image currently scans clean with no waivers.

.PHONY: build test lint gate coverage cov-check sast scan-deps \
	docker docker-api docker-realtime docker-worker docker-web scan \
	integration integration-docker

build:
	go build ./...

test:
	go test ./...

lint:
	go vet ./...

# --- Container build + scan (S-76) ---
# Image coordinates and the build metadata stamped into every image via ldflags.
IMAGE   ?= ancora/backend
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
BUILD_ARGS = --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) --build-arg BUILD_DATE=$(DATE)

# Build all three entrypoint images (api is the default target).
docker: docker-api docker-realtime docker-worker

docker-api:
	docker build $(BUILD_ARGS) --target api      -t $(IMAGE)-api:$(VERSION)      -t $(IMAGE)-api:latest .

docker-realtime:
	docker build $(BUILD_ARGS) --target realtime -t $(IMAGE)-realtime:$(VERSION) -t $(IMAGE)-realtime:latest .

docker-worker:
	docker build $(BUILD_ARGS) --target worker   -t $(IMAGE)-worker:$(VERSION)   -t $(IMAGE)-worker:latest .

# Frontend image (S-78). The Next.js standalone build under web/ ships as its own
# distroless/non-root image alongside the three backend targets.
WEB_IMAGE ?= ancora/web
docker-web:
	docker build -t $(WEB_IMAGE):$(VERSION) -t $(WEB_IMAGE):latest web

# Fail the build on any HIGH/CRITICAL finding. Documented waivers live in
# .trivyignore. Requires trivy (https://trivy.dev) on PATH.
scan: docker-api
	trivy image --severity HIGH,CRITICAL --ignorefile .trivyignore --exit-code 1 $(IMAGE)-api:$(VERSION)

# gate — S-67 domain gate. Verifies the whole module builds and every domain
# behavior (BDD) scenario under src/domain/**/model passes with zero failures.
# This is the single command CI runs to keep the domain green.
gate:
	go build ./...
	go vet ./...
	go test ./...

# --- Coverage gates (S-78) ---
# The PR pipeline fails below a coverage floor. Thresholds are variables so they
# can be raised over time (or overridden per-run) without editing recipes.
COVERAGE_MIN             ?= 30
INTEGRATION_COVERAGE_MIN ?= 30

# coverage — module-wide unit coverage gate. Runs the full unit suite with a
# coverage profile, prints the total, and fails if it is under COVERAGE_MIN.
#
# NOTE: the reported total is only trustworthy when the `covdata` tool is present
# in the active toolchain. Some downloaded toolchains ship without it; under
# `-coverprofile` every package that has no test files then fails to emit its
# zero-coverage entry and drops out of coverage.unit.out entirely, INFLATING the
# local total so it no longer matches CI. If a local number disagrees with CI,
# build the tool first:  go build -o "$(shell go env GOROOT)/pkg/tool/$(shell go env GOOS)_$(shell go env GOARCH)/covdata" cmd/covdata
coverage:
	go test -covermode=atomic -coverprofile=coverage.unit.out ./...
	@$(MAKE) --no-print-directory cov-check PROFILE=coverage.unit.out COVERAGE_MIN=$(COVERAGE_MIN)

# cov-check — reusable floor check over an existing coverprofile. Usage:
#   make cov-check PROFILE=coverage.integration.out COVERAGE_MIN=30
cov-check:
	@total=$$(go tool cover -func=$(PROFILE) | awk '/^total:/ {gsub(/%/,"",$$3); print $$3}'); \
	echo "coverage($(PROFILE)) = $$total%  (floor $(COVERAGE_MIN)%)"; \
	awk -v t="$$total" -v m="$(COVERAGE_MIN)" 'BEGIN { exit (t+0 < m+0) }' || { \
		echo "FAIL: coverage $$total% is below the $(COVERAGE_MIN)% threshold"; exit 1; }

# --- Security scanning (S-78) ---
# sast — basic static analysis over the Go source with gosec. Findings at or
# above medium severity/confidence fail the build; waive a checked line inline
# with a justified `#nosec Gxxx -- reason` comment. Requires gosec on PATH.
sast:
	gosec -quiet -severity medium -confidence medium -exclude-dir=web ./...

# scan-deps — dependency vulnerability scan: Go modules + stdlib via govulncheck
# (reports only vulns actually reachable from the code) and the filesystem
# manifests (go.sum, package-lock.json) via Trivy, failing on HIGH/CRITICAL.
# Documented waivers live in .trivyignore. Requires govulncheck and trivy.
scan-deps:
	govulncheck ./...
	trivy fs --severity HIGH,CRITICAL --ignorefile .trivyignore --exit-code 1 --scanners vuln .

# integration — S-75 API-level integration suite. Boots the HTTP server over the
# real repositories and stubbed external adapters and drives representative flows
# across every bounded context, emitting a coverage profile over the handler and
# infrastructure layers. It runs against live MongoDB/Redis when MONGODB_URI and
# ANCO_REDIS_ADDR are set (see integration-docker and CI) and against the
# in-memory equivalents otherwise, so it is green with or without infrastructure.
integration:
	go test -race -covermode=atomic \
		-coverpkg=./src/interfaces/rest/...,./src/infrastructure/... \
		-coverprofile=coverage.integration.out \
		./test/apiintegration/...
	go tool cover -func=coverage.integration.out | tail -n 1

# integration-docker — the single self-contained target: spin up ephemeral
# MongoDB and Redis, point the suite at them, run it with coverage, and tear the
# containers down regardless of outcome. Requires Docker.
integration-docker:
	@docker run -d --rm --name anco-it-mongo -p 27017:27017 mongo:7 >/dev/null
	@docker run -d --rm --name anco-it-redis -p 6379:6379 redis:7 >/dev/null
	@trap 'docker rm -f anco-it-mongo anco-it-redis >/dev/null 2>&1 || true' EXIT; \
		sleep 3; \
		MONGODB_URI=mongodb://localhost:27017 ANCO_REDIS_ADDR=localhost:6379 $(MAKE) integration

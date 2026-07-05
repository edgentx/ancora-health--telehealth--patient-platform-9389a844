.PHONY: build test lint gate docker docker-api docker-realtime docker-worker scan integration integration-docker

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

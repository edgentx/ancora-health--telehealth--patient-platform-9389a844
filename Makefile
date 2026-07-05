.PHONY: build test lint gate integration integration-docker

build:
	go build ./...

test:
	go test ./...

lint:
	go vet ./...

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

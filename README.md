# ancora-health--telehealth--patient-platform-9389a844
VForce360 project: Ancora Health — Telehealth &amp; Patient Platform

## Backend container image

The Go backend builds a minimal, non-root, statically linked image with three
selectable entrypoints (`api`, `realtime`, `worker`). See
[docs/container.md](docs/container.md); `make docker` builds all three and
`make scan` runs the Trivy HIGH/CRITICAL gate. The Next.js frontend ships as its
own distroless image via `web/Dockerfile` (`make docker-web`).

## CI/CD pipeline

GitHub Actions lint, test and scan every pull request and build, push and deploy
on merge to `main` (Helm chart, S-77). Coverage floors, security scan waivers,
required status checks, and the registry/cluster secrets are documented in
[docs/pipeline.md](docs/pipeline.md).

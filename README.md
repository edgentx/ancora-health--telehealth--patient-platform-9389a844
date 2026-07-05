# ancora-health--telehealth--patient-platform-9389a844
VForce360 project: Ancora Health — Telehealth &amp; Patient Platform

## Backend container image

The Go backend builds a minimal, non-root, statically linked image with three
selectable entrypoints (`api`, `realtime`, `worker`). See
[docs/container.md](docs/container.md); `make docker` builds all three and
`make scan` runs the Trivy HIGH/CRITICAL gate.

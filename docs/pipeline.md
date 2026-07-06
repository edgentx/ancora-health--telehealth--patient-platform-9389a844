# CI/CD pipeline (S-78)

The pipeline is four GitHub Actions workflows under `.github/workflows/`. On a
pull request into `main` they lint, test and scan; on merge to `main` (the
release branch) they build, push and deploy. Every job maps to a `make` target
so the same checks run locally.

## Workflows

| Workflow            | File                 | Trigger                | Purpose                                                                 |
|---------------------|----------------------|------------------------|-------------------------------------------------------------------------|
| CI                  | `ci.yml`             | PR + push to `main`    | Frontend lint / type-check / test / build; Go unit tests + coverage floor |
| Domain Gate         | `domain-gate.yml`    | PR + push to `main`    | All domain BDD scenarios green (S-67)                                    |
| API Integration     | `integration.yml`    | PR + push to `main`    | Integration flows over live MongoDB + Redis, with a coverage floor (S-75) |
| Security            | `security.yml`       | PR + push to `main`    | Dependency + image scans (HIGH/CRITICAL block) and Go SAST              |
| Release             | `release.yml`        | push to `main`         | Build & push all images (git-sha tagged), Helm deploy with rollback     |

The first four are the **required status checks** for merging into `main`
(configure under Settings → Branches → Branch protection):

- `CI / Go unit tests + coverage floor`
- `CI / Frontend lint, type-check, test, build`
- `Domain Gate / All domain BDD scenarios green`
- `API Integration Suite / API integration flows (real MongoDB + Redis)`
- `Security / Dependency vulnerabilities (Go + npm)`
- `Security / SAST (gosec)`
- `Security / Image scan (backend + frontend)`

`Release` runs only after a merge and is gated further by the GitHub Environment
(`prod` by default) — attach required reviewers there for a deploy approval step.

## Coverage gates

The PR build fails below a statement-coverage floor (default **30%**, room to
rise). Thresholds are Makefile variables, so no recipe edits are needed to raise
them:

- Unit: `make coverage` (`COVERAGE_MIN`, default 30) over `go test ./...`.
- Integration: `make integration` then `make cov-check PROFILE=coverage.integration.out COVERAGE_MIN=30`.

## Security scanning & waivers

| Scan            | Tool          | Blocks on        | Waive via                                                    |
|-----------------|---------------|------------------|--------------------------------------------------------------|
| Go deps + stdlib| `govulncheck` | reachable vulns  | upgrade the module (`go get -u`) — reachability-aware, few false positives |
| Filesystem deps | Trivy `fs`    | HIGH/CRITICAL    | `.trivyignore` (CVE id + reason + review date)               |
| Frontend deps   | `npm audit`   | HIGH+            | `npm audit fix` / bump the dependency                        |
| Go SAST         | `gosec`       | medium+ sev/conf | inline `#nosec Gxxx -- <reason>`                             |
| Container image | Trivy `image` | HIGH/CRITICAL    | `.trivyignore`; prefer rebuilding on a current base image    |

Prefer a fix over a waiver. Every `.trivyignore` entry must carry a CVE id, why
it is not exploitable here, and a review date.

## Required secrets & variables (Release)

Registry and cluster target are **deploy-time configuration**, not baked in.

**Repository variables** (Settings → Secrets and variables → Actions → Variables) — all optional, with the defaults shown:

| Variable           | Default    | Meaning                                            |
|--------------------|------------|----------------------------------------------------|
| `REGISTRY`         | `ghcr.io`  | Container registry host                            |
| `IMAGE_ORG`        | `edgentx`  | Registry org/namespace                             |
| `IMAGE_NAME`       | `ancora`   | Image repository name                              |
| `DEPLOY_NAMESPACE` | `ancora`   | Kubernetes namespace to deploy into                |
| `HELM_ENV`         | `prod`     | Selects `values-<env>.yaml` and the GH Environment |
| `REGISTRY_USERNAME`| `github.actor` | Registry login user (non-ghcr registries)      |

**Repository secrets:**

| Secret             | Required            | Meaning                                                            |
|--------------------|---------------------|--------------------------------------------------------------------|
| `KUBE_CONFIG`      | yes (deploy)        | base64-encoded kubeconfig for the MicroK8s target cluster          |
| `REGISTRY_TOKEN`   | non-ghcr registries | Registry password/token; ghcr.io falls back to the `GITHUB_TOKEN`  |

Produce the kubeconfig secret with, e.g.:

```bash
microk8s config | base64 -w0    # store the output as the KUBE_CONFIG secret
```

## Images produced

Each release push produces four images tagged with the commit sha (and `latest`),
at the coordinates the Helm `ancora.image` helper resolves:

```
<REGISTRY>/<IMAGE_ORG>/<IMAGE_NAME>/{api,realtime,worker,web}:<git-sha>
# e.g. ghcr.io/edgentx/ancora/api:<sha>
```

The deploy step passes `--set image.registry=<REGISTRY>/<IMAGE_ORG>
--set image.repository=<IMAGE_NAME> --set image.tag=<git-sha>` so pods pull the
exact artifacts the build produced. (The `web` image is published for the
frontend surface; wiring it into the chart's component map is a values-only
follow-up in S-77's chart.)

## Deploy behaviour (rollout & rollback)

```
helm upgrade --install ancora deploy/helm/ancora \
  --namespace <ns> --create-namespace \
  -f values.yaml -f values-<env>.yaml \
  --set image.tag=<git-sha> \
  --wait --atomic --timeout 10m
```

`--wait` blocks until every Deployment's rollout is complete; `--atomic` rolls
back to the previous revision automatically if the wait times out or a hook
fails. An explicit `helm rollback` step runs as a fallback on job failure.

## Caching

- **Go**: `actions/setup-go` caches the module + build cache keyed on `go.sum`.
- **Node**: `actions/setup-node` caches the npm cache keyed on `web/package-lock.json`.
- **Docker**: `docker/build-push-action` uses the GitHub Actions layer cache
  (`type=gha`, per-image `scope`) for both the scan builds and the release builds.

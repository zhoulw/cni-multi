# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`cni-multi` is a **Kubernetes operator** built with Kubebuilder v4.6.0 that manages CNI switching/migration (e.g., Calico to FlexNet CNI). Design documents are in `doc/` (`.docx` format). The operator domain is `zlw.domain`.

## Build & Run

```bash
# Build the manager binary
make build              # outputs bin/manager

# Run the controller locally (not in-cluster)
make run

# Build & push the Docker image
make docker-build docker-push IMG=<registry>/cni-multi:tag

# Code generation (DeepCopy methods, CRDs, RBAC, webhooks)
make generate           # DeepCopy methods
make manifests          # CRDs + RBAC + webhook config
```

**Always run `make generate && make manifests` after modifying API types (`api/`) or controller markers.**

## Testing

```bash
# Unit/integration tests (uses envtest — real API server binaries)
make test

# Run a subset of tests
KUBEBUILDER_ASSETS=$(shell setup-envtest use 1.33 --bin-dir ./bin -p path) \
  go test ./internal/controller/... -run TestName -v

# Lint
make lint               # check only
make lint-fix           # auto-fix where possible

# E2E tests (requires Kind — creates/destroys a temporary cluster)
make test-e2e
```

## Architecture

### Binary layout
- **`main.go`** — Stub; not the real entrypoint. Ignore this file.
- **`cmd/main.go`** — The **actual operator entrypoint**. Full controller-runtime manager with metrics server (HTTPS on :8443), health probes (:8081), cert watchers, webhook server, and leader election. Controllers are registered via `// +kubebuilder:scaffold:builder` marker.
- **`api/`** — (not yet created) API type definitions (Go structs for CRDs). Use `kubebuilder create api` to add types here.
- **`internal/controller/`** — (not yet created) Reconciliation logic. Use `kubebuilder create api` to scaffold a controller.

### Scaffold markers
Kubebuilder uses comment markers that get replaced by code generation — do not remove or reorder them:
- `// +kubebuilder:scaffold:scheme` — new API type registrations
- `// +kubebuilder:scaffold:builder` — new controller SetupWithManager calls
- `// +kubebuilder:scaffold:imports` — new import statements

### Config (Kustomize overlays)
- `config/default/` — Main overlay. Deploys to namespace `cni-multi-system` with name prefix `cni-multi-`.
- `config/rbac/` — RBAC manifests (ClusterRole currently only allows pod get/list/watch)
- `config/manager/` — Deployment spec for the controller-manager
- `config/prometheus/` — ServiceMonitor (disabled by default)
- `config/network-policy/` — NetworkPolicy for metrics endpoint (disabled by default)

### Key dependencies
| Dependency | Purpose |
|---|---|
| `sigs.k8s.io/controller-runtime` v0.21.0 | Operator framework |
| `sigs.k8s.io/controller-tools` v0.18.0 | Code generation (controller-gen) |
| `k8s.io/api` v0.33.0 / `k8s.io/client-go` v0.33.0 | Kubernetes API types and client |
| `github.com/onsi/ginkgo/v2` + `github.com/onsi/gomega` | Testing framework (used for both unit and e2e) |

### CI (GitHub Actions)
3 workflows run on every push and PR:
1. **Lint** — golangci-lint v2.1.0 (config in `.golangci.yml`)
2. **Tests** — `go mod tidy && make test` (unit tests with envtest)
3. **E2E Tests** — Installs Kind, runs `make test-e2e` (builds operator image, deploys to temp cluster, validates metrics endpoint)

### DevContainer
VS Code devcontainer defined in `.devcontainer/` — golang:1.24 with Kind, Kubebuilder, kubectl, and docker-in-docker pre-installed.

## Deploying to a cluster

```bash
make install                                          # install CRDs
make deploy IMG=<registry>/cni-multi:tag              # deploy operator
make build-installer IMG=<registry>/cni-multi:tag     # generate dist/install.yaml
```

## Adding a new API type + controller

```bash
kubebuilder create api --group <group> --version v1 --kind <Kind> --resource --controller
make generate && make manifests
```

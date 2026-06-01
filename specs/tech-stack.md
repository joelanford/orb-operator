# Tech Stack

## Language and Runtime

- **Go** (latest stable, currently 1.24)
- Module path: `github.com/joelanford/orb-operator`

## Core Dependencies

| Dependency | Purpose |
|---|---|
| `sigs.k8s.io/controller-runtime` | Controller framework, manager, reconciler, envtest |
| `k8s.io/client-go` | Kubernetes API client |
| `k8s.io/apimachinery` | API types, runtime objects, scheme |
| `package-operator/boxcutter` | Object management primitives for the COSR controller |
| `github.com/spf13/cobra` | CLI framework for the operator binary |
| `github.com/spf13/pflag` | Flag parsing |
| `k8s.io/klog/v2` | Logging implementation |
| `github.com/google/go-jsonnet` | Manifest generation |

## Dev Dependencies

| Dependency | Purpose |
|---|---|
| `github.com/stretchr/testify` | Unit test assertions (assert/require) |
| `github.com/cucumber/godog` | BDD-style e2e tests |
| `sigs.k8s.io/controller-runtime/pkg/envtest` | Integration test environment (API server + etcd) |
| `github.com/golangci/golangci-lint` | Linting (installed via Makefile) |
| `mvdan.cc/gofumpt` | Formatting (stricter than gofmt) |
| `sigs.k8s.io/controller-tools` | CRD/RBAC manifest generation (controller-gen) |

## Project Structure

```
orb-operator/
├── api/
│   └── v1alpha1/           # CRD types (COS, COSR, ClusterObjectSlice)
├── cmd/
│   └── operator/           # cobra entrypoint
├── internal/
│   ├── controller/         # reconcilers (COS, COSR)
│   ├── handler/            # object management (boxcutter integration)
│   └── assertions/         # assertion evaluation logic
├── config/
│   ├── crd/                # generated CRD manifests
│   ├── rbac/               # RBAC manifests
│   └── manager/            # Deployment manifest
├── test/
│   ├── integration/        # envtest-based tests
│   └── e2e/                # godog BDD tests
├── Dockerfile
├── Makefile
└── go.mod
```

## Build Commands

| Command | Purpose |
|---|---|
| `make build` | Build the operator binary |
| `make test` | Run unit and integration tests (envtest) |
| `make test-e2e` | Run godog BDD e2e tests |
| `make lint` | Run golangci-lint |
| `make fmt` | Run gofumpt |
| `make generate` | Run controller-gen (CRDs, RBAC, deepcopy) |
| `make manifests` | Generate CRD and RBAC manifests |
| `make docker-build` | Build the container image |
| `make check` | Run fmt, lint, test, and build (full CI check) |

## Containerization

- Multi-stage Dockerfile: build stage (Go builder) + runtime stage (distroless/static)
- Image: `orb-operator:latest` (configurable via `IMG` variable)

## CI/CD

- GitHub Actions workflow for PRs: `make check` (fmt, lint, test, build)
- GitHub Actions workflow for pushes to main: build + push container image

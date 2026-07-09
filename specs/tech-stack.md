# Tech Stack

## Language and Runtime

- **Go 1.26**
- Module path: `github.com/joelanford/orb-operator`

## Core Dependencies

| Dependency | Purpose |
|---|---|
| `sigs.k8s.io/controller-runtime` | Controller framework, manager, reconciler, envtest |
| `k8s.io/client-go` | Kubernetes API client |
| `k8s.io/apimachinery` | API types, runtime objects, scheme |
| `package-operator/boxcutter` | Object management primitives for the COS controller |
| `github.com/spf13/cobra` | CLI framework for the operator binary |
| `github.com/spf13/pflag` | Flag parsing |
| `k8s.io/klog/v2` | Logging implementation |

## Dev / Tool Dependencies

All build-time Go tools are declared as `tool` directives in go.mod and invoked via `go tool <name>`.

| Dependency | Purpose |
|---|---|
| `github.com/stretchr/testify` | Unit test assertions (assert/require) |
| `github.com/cucumber/godog` | BDD-style e2e tests |
| `sigs.k8s.io/controller-runtime/pkg/envtest` | Integration test environment (API server + etcd) |
| `sigs.k8s.io/controller-runtime/tools/setup-envtest` | Downloads envtest binaries matching k8s.io/api version (`go tool setup-envtest`) |
| `github.com/golangci/golangci-lint` | Linting (`go tool golangci-lint`) |
| `mvdan.cc/gofumpt` | Formatting (`go tool gofumpt`) |
| `sigs.k8s.io/controller-tools` | CRD/RBAC/deepcopy generation (`go tool controller-gen`) |
| `github.com/goreleaser/goreleaser` | Binary + Docker image builds (`go tool goreleaser`) |
| `github.com/google/go-jsonnet` | Jsonnet rendering (`go tool jsonnet`) |
| `sigs.k8s.io/kind` | Local Kubernetes clusters for e2e tests (`go tool kind`) |

## Project Structure

```
orb-operator/
├── api/
│   └── v1alpha1/           # CRD types (COD, COS, ClusterObjectSlice)
├── cmd/
│   └── operator/           # cobra entrypoint
├── internal/
│   ├── controller/         # reconcilers (COD, COS)
│   └── assertions/         # assertion evaluation logic
├── deploy/
│   ├── lib/                # shared jsonnet libraries
│   ├── operator.jsonnet    # main deployment manifest (Deployment, RBAC, etc.)
│   └── crds/               # controller-gen CRD output
├── test/
│   └── e2e/                # godog BDD tests
├── hack/
│   └── diff.sh             # verify script (jj-aware)
├── .github/
│   └── workflows/
│       ├── unit.yml
│       ├── integration.yml
│       ├── e2e.yml
│       ├── verify.yml
│       └── image.yml
├── .golangci.yml
├── .goreleaser.yml
├── Dockerfile
├── Makefile
└── go.mod
```

## Build Commands

| Command | Purpose |
|---|---|
| `make lint` | `go tool golangci-lint run ./...` |
| `make lint-fix` | `go tool golangci-lint run --fix ./...` |
| `make test-unit` | Run unit tests with coverage profile (`_output/unit/coverage.out`) |
| `make test-e2e` | Build coverage-instrumented binary, run godog BDD e2e tests, collect coverage (`_output/e2e/coverage.out`) |
| `make test-coverage` | Run test-unit + test-e2e, merge profiles (`_output/merged/coverage.out`), print summary |
| `make test-all` | Alias for test-coverage |
| `make build` | `go build ./...` (also called by `make verify`) |
| `make tidy` | `go mod tidy` |
| `make generate` | `go generate ./...` (controller-gen: CRDs, deepcopy) |
| `make verify` | lint + `./hack/diff.sh generate` + `go tool goreleaser check` + `go build ./...` (all non-test validation) |
| `make run` | Build image, create kind cluster (if needed), load image, apply manifests |

## Containerization

- goreleaser builds the binary and Docker image (`ghcr.io/joelanford/orb-operator`)
- Dockerfile: single-stage (`gcr.io/distroless/static:nonroot`), binary copied from goreleaser build context

## CI/CD

Separate GitHub Actions workflows per concern:

| Workflow | Triggers | Runs |
|---|---|---|
| `unit.yml` | PR, push to main | `make test-unit` |
| `e2e.yml` | PR, push to main | `make test-e2e` |
| `verify.yml` | PR, push to main | `make verify` |
| `image.yml` | push to main | `go tool goreleaser release --snapshot` |

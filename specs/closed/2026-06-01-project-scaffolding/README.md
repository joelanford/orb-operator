---
status: done
---
# Project Scaffolding

## Summary

Set up the foundational project structure so that the full toolchain works end-to-end: `make check` passes (lint, test, build), `goreleaser` builds the binary and container image, and GitHub Actions CI gates PRs and pushes. This is the foundation everything else builds on — no domain logic yet, just a working skeleton.

## Design

### Module and tooling

- Go 1.26, module path `github.com/joelanford/orb-operator`
- All build-time Go dependencies managed via `go tool` (Go 1.24+ tool directive in go.mod): golangci-lint, gofumpt, controller-gen, goreleaser
- goreleaser handles binary compilation and Docker image building (`ghcr.io/joelanford/orb-operator`)

### Project layout

```
orb-operator/
├── api/
│   └── v1alpha1/         # empty package (placeholder for CRD types)
├── cmd/
│   └── operator/
│       └── main.go       # cobra root command, wires up controller-runtime manager
├── internal/
│   ├── controller/       # empty package (placeholder for reconcilers)
│   ├── handler/          # empty package (placeholder for boxcutter integration)
│   └── assertions/       # empty package (placeholder for assertion logic)
├── test/
│   ├── integration/      # empty package (placeholder for envtest tests)
│   └── e2e/              # empty package (placeholder for godog tests)
├── hack/
│   └── diff.sh           # verify script (same pattern as library-olm)
├── deploy/
│   ├── lib/              # shared jsonnet libraries
│   ├── operator.jsonnet  # main deployment manifest (renders Deployment, RBAC, etc.)
│   └── crds/             # placeholder for controller-gen output (generated, not hand-written)
├── .github/
│   └── workflows/
│       ├── unit.yml          # make test-unit
│       ├── integration.yml   # make test-integration
│       ├── e2e.yml           # make test-e2e
│       ├── verify.yml        # make lint + make verify + make build
│       └── image.yml         # goreleaser image build/push (main only)
├── .golangci.yml         # linter config (modeled on library-olm)
├── .goreleaser.yml       # goreleaser config for binary + Docker image
├── Makefile              # check, lint, lint-fix, test, build, tidy, generate, verify
├── Dockerfile            # used by goreleaser for image build
└── go.mod
```

### Makefile targets

Follows library-olm's pattern with additions for an operator project:

| Target | What it does |
|---|---|
| `check` | Runs `lint verify test-all build` (CI entry point) |
| `lint` | `go tool golangci-lint run ./...` |
| `lint-fix` | `go tool golangci-lint run --fix ./...` |
| `test-unit` | `go test ./internal/... ./api/...` (pure logic, no envtest) |
| `test-integration` | `go test ./test/integration/...` (envtest) |
| `test-e2e` | `go test ./test/e2e/...` (godog BDD) |
| `test-all` | Runs `test-unit test-integration test-e2e` |
| `build` | `go build ./...` |
| `tidy` | `go mod tidy` |
| `generate` | `go generate ./...` (controller-gen deepcopy + CRD) |
| `verify` | `./hack/diff.sh generate` (checks generated code is up to date) |

### cobra entrypoint

Minimal `cmd/operator/main.go` with a cobra root command that:
- Sets up klog flags
- Creates a controller-runtime manager
- Starts the manager (no controllers registered yet)

### CI

Separate GitHub Actions workflows per concern, all triggered on PRs and push to main (except image which is push-to-main only):

| Workflow | File | Triggers | Runs |
|---|---|---|---|
| Unit Tests | `unit.yml` | PR, push to main | `make test-unit` |
| Integration Tests | `integration.yml` | PR, push to main | `make test-integration` |
| E2E Tests | `e2e.yml` | PR, push to main | `make test-e2e` |
| Verify | `verify.yml` | PR, push to main | `make lint`, `make verify`, `make build` |
| Image | `image.yml` | push to main | `go tool goreleaser release --snapshot` |

### golangci-lint config

Based on operator-controller's config, adapted for this project's module path:

- **Formatters:** gci (standard, dot, default, localmodule), gofmt
- **Linters:** asciicheck, bodyclose, errorlint, gosec, importas, misspell, nestif, nonamedreturns, prealloc, staticcheck, testifylint, tparallel, unconvert, unparam, whitespace
- **importas aliases:** standard k8s aliases (metav1, apierrors, apiextensionsv1, utilruntime, core/v1 pattern, ctrl) plus project-specific aliases for `api/v1alpha1`
- **Exclusions:** generated code, comments, common-false-positives, legacy, std-error-handling presets

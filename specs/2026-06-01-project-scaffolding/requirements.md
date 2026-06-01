# Requirements

- go.mod with module path `github.com/joelanford/orb-operator` and Go 1.26
- All build-time Go tools (golangci-lint, gofumpt, controller-gen, goreleaser) declared as `tool` directives in go.mod
- Makefile with targets: check, lint, lint-fix, test, build, tidy, generate, verify
- `make check` passes cleanly on a fresh clone
- `hack/diff.sh` verify script works with jj (same pattern as library-olm)
- cobra-based entrypoint in `cmd/operator/main.go` that starts a controller-runtime manager
- `.golangci.yml` with formatters (gci, gofmt) and linters (errcheck, govet, importas, ineffassign, misspell, staticcheck, unused)
- `.goreleaser.yml` configured to build the operator binary and Docker image (`ghcr.io/joelanford/orb-operator`)
- Dockerfile for goreleaser image builds (multi-stage: Go builder + distroless runtime)
- Separate GitHub Actions workflows: unit tests, integration tests, e2e tests, verify (lint + generate + build), and image build/push (main only)
- Placeholder packages for api/v1alpha1, internal/controller, internal/handler, internal/assertions, test/integration, test/e2e
- `deploy/` directory with jsonnet manifests: `operator.jsonnet` (main entry point), `lib/` (shared libraries), `crds/` (placeholder for controller-gen CRD output)

## Acceptance Criteria

- `make check` exits 0 with no output issues
- `make lint` exits 0
- `make test-unit` exits 0 (no tests to run yet, but no errors)
- `make test-integration` exits 0
- `make test-e2e` exits 0
- `make test-all` exits 0
- `make build` produces no errors
- `make verify` exits 0 (generated code is up to date)
- `go tool goreleaser check` validates the goreleaser config
- `go tool goreleaser build --snapshot --clean` produces a binary
- The cobra entrypoint compiles and prints help when run with `--help`
- All CI workflow files are valid YAML with correct trigger configuration

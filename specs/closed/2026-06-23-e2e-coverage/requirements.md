# Requirements

- `make test-unit` produces a text coverage profile at `_output/unit/coverage.out`
- `make test-e2e` builds a coverage-instrumented binary+image via goreleaser (`GO_BUILD_FLAGS='-cover -tags=cover -covermode=atomic'`), deploys with the `e2e` profile, runs e2e tests, flushes coverage via SIGUSR1, and produces a text profile at `_output/e2e/coverage.out`
- `make test-coverage` merges unit and e2e profiles into `_output/merged/coverage.out` and prints a `go tool cover -func` summary
- `make test-all` produces the merged coverage report (depends on `test-coverage`)
- E2E coverage collection validates that `covcounters.*` files exist before converting, failing loudly on missing counters
- The operator pod stays running after coverage collection so logs remain available for debugging
- `_output/` is in `.gitignore`
- `.goreleaser.yml` maps `GO_BUILD_FLAGS` env var to `GOFLAGS` in the build subprocess so Go handles flag parsing natively
- `cmd/operator/covflush.go` (build-tagged `//go:build cover`) handles SIGUSR1 by calling `runtime/coverage.WriteCountersDir` and `WriteMetaDir`
- `deploy/operator.jsonnet` supports a `profiles` external variable (array of strings) that conditionally applies e2e-specific configuration when `"e2e"` is in the array
- `make run` passes `PROFILES` (default `[]`) and `GO_BUILD_FLAGS` (default empty) through to goreleaser and jsonnet, preserving existing behavior
- All existing `test-unit`, `test-e2e`, and `run` behavior (exit codes, output, kind cluster management) is preserved

## Acceptance Criteria

- Running `make test-unit` produces `_output/unit/coverage.out` with non-zero coverage
- Running `make test-e2e` produces `_output/e2e/coverage.out` with non-zero coverage
- Running `make test-coverage` produces `_output/merged/coverage.out` and prints a function-level summary
- Running `make test-all` produces the merged report
- After `make test-e2e`, the operator pod is still running and `kubectl logs` works
- If SIGUSR1 flush fails (e.g. GOCOVERDIR not mounted), `test-e2e` fails with a clear error about missing counter files
- `make verify` still passes (no lint or build regressions)
- `deploy/operator.jsonnet` with default profiles (`[]`) renders identically to the current output

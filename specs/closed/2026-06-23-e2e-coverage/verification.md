# Verification

## Implementation Correctness

- [ ] `make test-unit` produces `_output/unit/coverage.out` with non-zero statement coverage
- [ ] `make test-e2e` produces `_output/e2e/coverage.out` with non-zero statement coverage
- [ ] After `make test-e2e`, operator pod is still running and logs are accessible
- [ ] `make test-coverage` produces `_output/merged/coverage.out` and prints function-level summary
- [ ] `make test-all` produces the merged report as a side effect
- [ ] Removing the hostPath volume (so SIGUSR1 flush has nowhere to write) causes `test-e2e` to fail with a clear error about missing counters
- [ ] `deploy/operator.jsonnet` with default profiles (`[]`) renders identically to current output
- [ ] `make run` still works unchanged (deploys without e2e config)
- [ ] `_output/` directory is gitignored
- [ ] `go tool goreleaser release --snapshot --clean` (no GO_BUILD_FLAGS) still builds normally
- [ ] `GO_BUILD_FLAGS='-cover -tags=cover -covermode=atomic' go tool goreleaser release --snapshot --clean` produces coverage-instrumented binary
- [ ] `covflush.go` is excluded from normal (non-cover) builds

## Project Conventions

- [ ] No `//nolint` comments added
- [ ] Makefile targets follow existing naming convention (`verb-noun`)
- [ ] Jsonnet follows existing patterns in `deploy/operator.jsonnet` (local variables, same structure)
- [ ] `make verify` passes (lint, generate diff, goreleaser check, build)
- [ ] Coverage artifacts go under `_output/` (consistent output directory, gitignored)
- [ ] No new Go dependencies added (uses only `runtime/coverage` from stdlib, `go tool covdata` and `go tool cover`)

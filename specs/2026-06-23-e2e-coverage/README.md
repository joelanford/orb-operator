---
status: backlog
---
# E2E Coverage Collection

## Summary

Add coverage profiling to `make test-e2e-coverage` so we can measure which lines of operator code are exercised by the e2e test suite running against a real Kind cluster.

## Approach

Build the operator binary with `go build -cover`, deploy it into Kind with `GOCOVERDIR` pointing at a hostPath volume, run e2e tests, gracefully shut down the operator, and collect the coverage counter files.

### Key components

1. **`.gitignore`** — add `/_output/` (coverage artifacts land there)
2. **`deploy/operator.jsonnet`** — conditional coverage support:
   - `GOCOVERDIR=/coverage` env var
   - hostPath volume mount (`/tmp/e2e-coverage` on node → `/coverage` in container)
   - `imagePullPolicy: Never` (image is loaded via `kind load`)
   - `terminationGracePeriodSeconds: 120` (ensure clean shutdown)
3. **`Makefile`** — `test-e2e-coverage` target:
   - Cross-compile with `-cover`
   - Build and load a coverage-instrumented container image
   - Create Kind cluster, deploy, run e2e tests
   - Scale to 0, wait for pod deletion, copy coverage files from node
   - Convert with `go tool covdata textfmt`, report with `go tool cover -func`

## Design notes

### Coverage counter flush requires clean process exit

Go's `-cover` binary writes `covmeta.*` at startup and `covcounters.*` at process exit via an atexit handler. If the binary is SIGKILLed, panics, or calls `os.Exit` on a code path that bypasses atexit, only the metadata file is written and coverage reports 0.0%.

**Mitigations:**
- The operator binary must use `cmd.ExecuteContext(ctrl.SetupSignalHandler())` so SIGTERM triggers graceful manager shutdown and main returns normally. (Landed separately as a standalone fix.)
- The Makefile uses `kubectl scale --replicas=0` + `kubectl wait --for=delete` to ensure the pod exits before collecting files. This sequence is correct — no race condition.

### Validate counter files before reporting

The current pipeline silently produces a valid-but-empty coverage profile (all 0.0%) when counter files are missing. The `covdata textfmt` step should check that `covcounters.*` files exist and fail loudly if they don't, to avoid misleading results.

### Integration tests are redundant with e2e

Coverage analysis (2026-06-23) showed:
- **Integration:** 318/755 statements (42.1%)
- **E2E:** 546/755 statements (72.3%)
- **Merged:** 547/755 statements (72.5%)
- **Overlap:** 317 of integration's 318 covered statements are also covered by e2e

The single exclusively-integration-covered statement is an error-return path in `reconcileArchived` (`cosr_controller.go:241-243`) that would be contrived to trigger in e2e. The envtest/setup-envtest machinery is not worth carrying for this. Integration tests for COSR phase status were removed.

## TODOs

- [ ] Add `/_output/` to `.gitignore`
- [ ] Add conditional coverage support to `deploy/operator.jsonnet` (env, volume, imagePullPolicy, terminationGracePeriodSeconds)
- [ ] Add `test-e2e-coverage` Makefile target
- [ ] Add counter file validation before `covdata textfmt` step

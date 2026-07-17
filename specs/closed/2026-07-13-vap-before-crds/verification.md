# Verification

## Implementation Correctness

- [ ] `go tool jsonnet deploy/main.jsonnet` output contains the same objects as before (diff check).
- [ ] VAP/VAPBs appear before CRDs in the rendered output.
- [ ] Envtest suites install VAP/VAPBs from rendered jsonnet (no hand-duplicated definitions).
- [ ] Envtest suites still pass with VAPs active (existing validation tests exercise VAP-guarded paths).
- [ ] `deploy/lib/api.libsonnet` is the single source for API-surface objects.
- [ ] `deploy/lib/controller.libsonnet` is the single source for workload objects.

## Project Conventions

- [ ] No `//nolint` comments added.
- [ ] Code formatted with gofumpt.
- [ ] `make lint` passes.
- [ ] `make verify` passes.
- [ ] `make test-unit` passes.
- [ ] `make test-e2e` passes.
- [ ] Jsonnet follows existing style conventions in `deploy/`.

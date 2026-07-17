# Requirements

- VAP/VAPBs are installed before CRDs in the deployment manifest order.
- `deploy/lib/api.libsonnet` exports all API-surface objects (VAPs, VAPBs, CRDs) in correct order.
- `deploy/lib/controller.libsonnet` exports all workload objects (Namespace, SA, CRB, Deployment, Service).
- `deploy/main.jsonnet` imports both libraries and concatenates them.
- Envtest suites (`api/v1alpha1/`, `internal/cosutil/`) install VAP/VAPBs by rendering jsonnet at test time.
- The rendered manifests are the single source of truth — no hand-duplicated VAP definitions in test code.

## Acceptance Criteria

- `go tool jsonnet deploy/main.jsonnet` produces the same set of objects as before, but with VAP/VAPBs ordered before CRDs.
- `make verify` passes.
- `make test-unit` passes (envtest suites exercise VAP-guarded paths).
- `make test-e2e` passes.
- `make run` deploys successfully with VAP/VAPBs active before any orb resources exist.

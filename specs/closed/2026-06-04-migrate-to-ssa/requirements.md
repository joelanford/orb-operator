# Requirements

- COS controller uses SSA (`client.Apply`) for all COSR metadata writes: owner references, labels, and `spec.lifecycleState`.
- COSR controller uses SSA for adding the finalizer.
- COSR controller uses optimistic-lock merge patch for removing the finalizer, including clearing its field ownership entry.
- Each controller has a distinct field owner identity (`cos-controller`, `cosr-controller`) so ownership is partitioned.
- Typed apply configurations are auto-generated from the API types and committed to the repo.
- The `applyCOSR` helper short-circuits when no change is needed to avoid unnecessary API calls.
- After finalizer removal, the COSR controller waits for the informer cache to sync before returning.
- COSR names are validated to be usable as Kubernetes field owner strings.

## Acceptance Criteria

- All COSR metadata writes from the COS controller appear under the `cos-controller` managed fields entry.
- The COSR finalizer appears under the `cosr-controller` managed fields entry.
- After finalizer removal, no stale `cosr-controller` ownership entry remains for the finalizer.
- The COS controller self-heals field ownership on every reconcile, even for pre-existing COSRs.
- `make generate` reproduces the `applyconfigurations/` directory without diff.
- All existing tests (unit, integration, e2e) continue to pass.
- COSR names exceeding 128 characters or with leading/trailing whitespace are rejected by validation.

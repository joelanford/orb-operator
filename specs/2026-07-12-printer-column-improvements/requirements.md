# Requirements

- COD printer columns show: NAME, AVAILABLE, SYNCED, TOTAL, AGE.
- COS printer columns show: NAME, GROUP, REV, AVAILABLE, SYNCED, TOTAL, LIFECYCLE, AGE.
- COD status includes an `objectCounts ObjectCounts` field (reuses the existing struct).
- COS status includes an `objectCounts ObjectCounts` field (reuses the existing struct).
- COS counts are sums of per-phase `ObjectCounts` from `observedPhases`.
- COD counts are derived from the latest active (non-archived, highest revision) COS status.
- The invariant `total >= synced >= available` holds on both COD and COS `objectCounts`.
- When no observed phases exist (e.g. internal error), all COS counts are zero.
- When no active COS exists, all COD counts are zero.
- Existing Available and Progressing conditions remain on COD and COS status — only the printer column annotations change.

## Acceptance Criteria

- `kubectl get cod` shows AVAILABLE, SYNCED, TOTAL columns with correct values.
- `kubectl get cos` shows AVAILABLE, SYNCED, TOTAL columns with correct values.
- COS counts update on each reconcile as phases progress.
- COD counts reflect the latest active COS.
- `make verify` passes.
- `make test-unit` passes.
- `make test-e2e` passes.

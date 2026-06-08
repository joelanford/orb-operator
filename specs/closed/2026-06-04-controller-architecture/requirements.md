# Requirements

## COSR Controller

- COSR controller reconciles chains (one reconciliation per `(group, controllerOwnerName)` key), not individual COSRs
- All COSR events map to the latest COSR in the chain via a Watches mapper; the workqueue deduplicates
- Latest active COSR in a chain calls `engine.Reconcile` with all chain predecessors as previous owners
- Non-latest active COSRs in a chain get `Available=False, Reason=Superseded` and retain non-transitioned objects
- Archived COSRs run `engine.Teardown` and get `Available=False, Reason=Archived`
- COSR controller never mutates `spec.lifecycleState` — lifecycle transitions are external
- Every error path sets a condition reflecting the failure before returning

## COS Controller

- Template change detection uses a deterministic hash label (`orb.operatorframework.io/template-hash`) on COSRs instead of deep comparison
- Hash is computed over the full `ClusterObjectSetTemplate` (metadata + spec) via JSON → SHA-256 → 8 hex chars
- COS adopts unowned COSRs in its group by setting a controller ownerRef
- COSRs already owned by a different controller are skipped
- When the latest owned COSR is Available, COS sets `lifecycleState: Archived` on all older Active owned COSRs
- Pruning deletes Archived COSRs (without finalizers) beyond `revisionHistoryLimit`, lowest revision first, best-effort
- Revision numbering uses `max(revision across ALL COSRs in the group) + 1` to avoid name collisions
- Status derived from Active (non-Archived) owned revisions only: 0 → Unavailable, 1 → mirror, >1 → Progressing

## Tests

- Move the archival+cleanup scenario from `cosr_revision_transitions.feature` to `cos_revision_management.feature`
- Change standalone COSR expectations from `reason "Archived"` to `reason "Superseded"` in two scenarios
- Add a scenario verifying superseded standalone COSRs retain non-transitioned objects
- Add a COS scenario testing template change → new revision → archival → teardown
- Add a COS scenario testing adoption of an unowned COSR in its group

## Acceptance Criteria

- All existing e2e tests pass (with the modified expectations for standalone COSRs)
- The COSR controller never writes to `spec.lifecycleState` on any code path
- Template hash label is present on every COSR created by the COS controller
- Modifying COSR labels/annotations externally does not trigger a new revision
- COS adopts an unowned COSR in its group and includes it in status
- COS archives older Active COSRs when the latest becomes Available
- Superseded standalone COSRs retain objects not adopted by the latest
- COS-driven archival + teardown cleans up objects from old revisions
- `make verify` passes (lint, build, generate check)
- `make test-unit` passes
- `make test-e2e` passes

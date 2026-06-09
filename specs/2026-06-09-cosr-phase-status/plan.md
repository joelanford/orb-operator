# Implementation Plan

1. **Add API types** — add `PhaseStatus` enum (`Reconciling`/`Complete`/`Unknown`), `ObservedPhase` and `ObjectStatus` structs, `ObservedPhases []ObservedPhase` and `CompletedAt *metav1.Time` fields to `ClusterObjectSetRevisionStatus` in `api/v1alpha1/types_clusterobjectsetrevision.go`. Run `make generate` to update deepcopy and CRDs.

2. **Build phase status from boxcutter results** — add a helper function that takes the spec's phase list and `[]machinery.PhaseResult` and produces `[]ObservedPhase`. Phases returned by boxcutter are `Complete` or `Reconciling` based on `IsComplete()`. For `Reconciling` phases, iterate `GetObjects()` to build `ObjectStatus` entries for incomplete objects. Phases not returned by boxcutter are set to `Unknown`.

3. **Wire into COSR reconciliation** — update `doReconcileLatest` to populate `ObservedPhases` and set `CompletedAt` once when all phases first complete. Clear `ObservedPhases` in `doReconcilePredecessor` (superseded) and `doReconcileArchived` (archived/teardown). Preserve existing `CompletedAt` across all paths (never clear it).

4. **Update integration tests** — add test cases to the existing envtest suite that verify: all four phase statuses appear correctly for multi-phase COSRs (Reconciling/Complete/Unknown), `completedAt` is set once and preserved through regression, and archived/superseded COSRs clear `observedPhases` but keep `completedAt`.

5. **Add e2e test scenarios** — add godog scenarios covering: phase status progression (Unknown → Reconciling → Complete), unavailable objects in active phases, completedAt set on completion, completedAt preserved through regression, and observedPhases cleared on archival/supersession.

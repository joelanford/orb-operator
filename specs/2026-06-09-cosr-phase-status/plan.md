# Implementation Plan

1. **Add API types** — add `PhaseStatus` enum (`Reconciling`/`Available`/`Unknown`/`Superseded`/`TearingDown`/`TeardownComplete`), `ObservedPhase` and `ObjectStatus` structs, `ObservedPhases []ObservedPhase` and `CompletedAt *metav1.Time` fields to `ClusterObjectSetRevisionStatus` in `api/v1alpha1/types_clusterobjectsetrevision.go`. Run `make generate` to update deepcopy and CRDs.

2. **Build phase status from boxcutter results** — add a helper function that takes the spec's phase list and `[]machinery.PhaseResult` and produces `[]ObservedPhase`. Phases returned by boxcutter are `Available` or `Reconciling` based on `IsComplete()`. For `Reconciling` phases, iterate `GetObjects()` to build `ObjectStatus` entries for incomplete objects. Phases not returned by boxcutter are set to `Unknown`.

3. **Wire into COSR reconciliation** — update `doReconcileLatest` to populate `ObservedPhases` and set `CompletedAt` once when all phases first complete. Clear `ObservedPhases` for superseded COSRs. Preserve existing `CompletedAt` across all paths (never clear it).

3b. **Add teardown phase status** — add `TearingDown` and `TeardownComplete` enum values. Add `buildTeardownObservedPhases` to map `PhaseTeardownResult` to `ObservedPhase`. Add `updateTeardownStatus` helper that populates phases during teardown and clears them on completion. Refactor `teardownCOSR` to call `updateTeardownStatus`, simplifying both `reconcileArchived` and `handleDeletion`.

4. **Update integration tests** — add test cases to the existing envtest suite that verify: all four phase statuses appear correctly for multi-phase COSRs (Reconciling/Available/Unknown), `completedAt` is set once and preserved through regression, and archived/superseded COSRs clear `observedPhases` but keep `completedAt`.

5. **Add e2e test scenarios** — add godog scenarios covering: phase status progression (Unknown → Reconciling → Available), incomplete objects in active phases, completedAt set on completion, completedAt preserved through regression, and observedPhases cleared on archival/supersession.

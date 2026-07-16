# Implementation Plan

## 1. API: add Present field to ObjectCounts

- In `api/v1alpha1/types_clusterobjectset.go`, add `Present int64 `json:"present"`` to `ObjectCounts` (between Total and Synced)
- Add CEL validation rule on `ClusterObjectSetStatus`: `objectCounts.present == observedPhases.sum(p, p.objectCounts.present)`
- Add a PRESENT printer column on `ClusterObjectSet` (between SYNCED and TOTAL, column order: AVAILABLE, SYNCED, PRESENT, TOTAL)
- Run `make generate` to regenerate CRDs and deepcopy

## 2. Status infrastructure: sumObjectCounts

- In `internal/status/cos/status.go`, update `sumObjectCounts` to also sum `Present` from each phase

## 3. Reconcile path: populate present counts

- In `mapSpecPhases`, add a `buildCompleteObjectCounts func(total int64) ObjectCounts` callback parameter to replace the hardcoded `ObjectCounts{Total: total, Synced: total, Available: total}`
- `buildObservedPhases` passes a callback that returns `ObjectCounts{Total: total, Present: total, Synced: total, Available: total}`
- In `incompletePhase`, compute `present` as the count of objects returned by `pr.GetObjects()` (all processed objects exist on the cluster)

## 4. Teardown path: fix complete and tearing-down phase counts

- `buildTeardownObservedPhases` passes a callback that returns `ObjectCounts{Total: total, Present: 0, Synced: 0, Available: 0}` for complete phases
- In `tearingDownPhase`, use the `Phase` argument (currently `_`) to set `Total = len(sp.Objects)`, and set `Present = len(pr.Waiting())`, `Synced = 0`, `Available = 0`

## 5. Read-only teardown phases: read-only presence check

This is the most significant change. Currently, read-only phases (waiting phases from boxcutter's `RevisionTeardownResult`) appear as "unevaluated" with Unknown status and no object-level data. We need to:

- In the controller (`doTeardown` or a post-processing step), after `engine.Teardown` returns:
  - Get the waiting phase names from the `RevisionTeardownResult`
  - For each waiting phase name, find the matching spec phase
  - For each object identity in that spec phase, do a cache Get to check existence
  - Build a result carrying {phaseName, presentCount, total}
- Create a type (e.g., `read-onlyTeardownPhaseResult`) that satisfies the `mapSpecPhases` type constraint (`GetName() string`, `IsComplete() bool` returning false)
- Include these results alongside the boxcutter `PhaseTeardownResult` results when calling `buildTeardownObservedPhases`, so read-only phases go through the `buildIncomplete` path instead of the "unevaluated" path
- In `tearingDownPhase`, handle both active tearing-down results (from boxcutter) and read-only results (from our cache check) - distinguish by type assertion or by a marker on the result

## 6. Unit tests

- Update `TestFromTeardown` in `internal/status/cos/status_test.go` to assert on object counts for all phase states (complete, tearing-down, read-only)
- Add tests for `incompletePhase` verifying present counts during reconcile
- Add tests for the read-only-phase presence computation

## 7. E2e step definitions: add present to count assertions

The existing step definitions use `total:N/synced:N/available:N` format:
- `theCOSShouldHaveObjectCounts` in `steps_assert.go:488`
- `observedPhaseShouldHaveObjectCounts` in `steps_assert.go:473`
- `theCODShouldHaveObjectCounts` in `steps_assert.go:499`

Update all three to accept `present` (e.g., `total:N/present:N/synced:N/available:N`).

## 8. E2e feature files: update existing count assertions

~30 existing assertions across `cos_phase_status.feature`, `cod_status.feature`,
and `cos_object_slice.feature` use the old format. Update each with the correct
`present` value. For reconcile scenarios, `present` generally equals the count of
objects that boxcutter has processed in that phase.

## 9. E2e feature files: add teardown count assertions

Existing teardown scenarios in `cos_lifecycle.feature` and `cos_phase_status.feature`
assert on phase status and object deletion but have zero count assertions. Add:

- Final state assertions after teardown completes: all phases should have
  present:0/synced:0/available:0 with total unchanged
- Aggregate COS-level counts should be 0/0/0/total after teardown completes
- For multi-phase teardown scenarios (e.g., "Archived COS shows TeardownComplete,
  TearingDown, and Unknown during teardown"), assert intermediate counts:
  - TeardownComplete phases: present=0
  - TearingDown phase: present = objects still waiting for deletion
  - Read-only phases (previously Unknown): present = objects found in cache (should
    equal total since teardown hasn't reached them yet)

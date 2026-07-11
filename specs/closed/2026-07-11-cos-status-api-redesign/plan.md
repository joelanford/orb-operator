# Implementation Plan

## 1. Bump boxcutter to latest main

- `go get pkg.package-operator.run/boxcutter@main && go mod tidy`
- Fix any compile errors from the `PhaseValidationError` pointer-receiver change (e.g., `phase_status.go:66` — `applyValidationError` already takes `*validation.PhaseValidationError`, so this should be compatible, but verify).
- Run `make verify && make test-unit` to confirm nothing breaks.

## 2. Add `PhaseStatusInvalid` to the API

- Add `PhaseStatusInvalid PhaseStatus = "Invalid"` to `api/v1alpha1/types_clusterobjectset.go` with godoc.
- Update the `ObservedPhase` CEL validation rule that enumerates valid `PhaseStatus` values to include `Invalid`.
- Run `make generate` to regenerate CRDs and deepcopy.
- Run `make test-unit` to confirm validation tests pass (the existing test at `validation_cos_test.go:135` that uses `PhaseStatus("Invalid")` as an invalid value will need updating — it should now be accepted).

## 3. Implement the wrapper revision engine

Create `internal/controller/revision_engine.go`:

- Define `revisionEngine` struct holding:
  - `*boxcutter.RevisionEngine` (the gated reconcile engine)
  - `*machinery.PhaseEngine` (for direct per-phase reconciliation)
  - `existingPhases []orbv1alpha1.ObservedPhase` (to read `completedAt`)
- Constructor takes `RevisionEngineOptions` and `existingPhases`. Internally calls `boxcutter.NewRevisionEngine(opts)` and `boxcutter.NewPhaseEngine(opts)`.
- `Reconcile(ctx, rev, opts...)` method:
  1. Call `RevisionEngine.Reconcile(ctx, rev, opts...)` → `(RevisionResult, error)`
  2. If `error != nil`, `result.GetValidationError() != nil`, or `result.HasProgressed()` → return as-is, no drift correction. Validation error mapping and superseded handling stay in `observedPhasesFromReconcileResult`.
  3. Otherwise, identify phases from `rev` that:
     - Are NOT in `result.GetPhases()` (were skipped by the gated loop), AND
     - Have `completedAt` set in `existingPhases`
  4. For each such phase, call `PhaseEngine.Reconcile(ctx, rev.Revision, phase, phaseOpts...)` with `WithAggregatePhaseReconcileErrors()` and collect the `PhaseResult`.
     - **Error handling:** If a drift-correction call returns a non-nil error, short-circuit — stop attempting remaining phases and return the error with whatever results have been collected. Boxcutter handles per-object aggregation internally; a non-nil error means it decided to stop.
  5. Return a composite result that merges the gated-loop phase results with the drift-correction phase results, and the error (if any). Phases skipped by both the gated loop and drift correction are NOT included (they remain absent, and `observedPhasesFromReconcileResult` will set them to `Unknown`).

- Define a composite result type that implements `RevisionResult` (or a similar interface consumed by `observedPhasesFromReconcileResult`).

## 4. Update `observedPhasesFromReconcileResult` to handle `Invalid` and `Unknown` messages

In `internal/controller/phase_status.go`:

- When `result.GetValidationError() != nil`, instead of returning `nil`, build `ObservedPhase` entries:
  - For each phase in `RevisionValidationError.Phases` → `Invalid` with error detail.
  - For remaining phases → `Unknown` with `Error` = `"Blocked by preflight errors in other phases"`.
- In `mapSpecPhases`, when a phase is not in the results map (currently produces bare `Unknown`), set `Error` = `"Waiting for earlier phases to complete"`.

## 5. Wire the wrapper into `doReconcileActive`

In `internal/controller/cos_controller.go`:

- Replace `boxcutter.NewRevisionEngine(opts)` with a constructor that creates the wrapper `revisionEngine` (passing `cos.Status.ObservedPhases` for `completedAt` lookup).
- Adjust the `engineForCOS` method to return the wrapper type.
- The rest of `doReconcileActive` should need minimal changes since the wrapper produces the same result shape.

## 6. Unit tests

- `phase_status_test.go`: Add test cases for:
  - `observedPhasesFromReconcileResult` when `RevisionValidationError` is present (per-phase `Invalid` + `Unknown` with blocked message).
  - `Unknown` phases get the "waiting" error message.
- `revision_engine_test.go`: Add test cases for:
  - Drift correction: completed-but-skipped phases get `PhaseEngine.Reconcile` called.
  - Never-completed skipped phases remain `Unknown`.
  - Validation error passthrough.

## 7. E2e scenarios

Add godog scenarios covering:

- **Preflight failure**: Update an existing `InvalidRevision` scenario (e.g., duplicate object across phases) to also assert per-phase status: the erroring phase shows `Invalid` with error detail, other phases show `Unknown` with "Blocked by preflight errors in other phases".
- **Normal gating messages**: Update existing multi-phase gating scenario to assert that unevaluated phases show `Unknown` with "Waiting for earlier phases to complete".
- **Drift correction**: Multi-phase COS that reaches `Available`, then an earlier phase regresses (delete an object from phase 1). Verify that phase 2 continues to self-heal (its objects are re-applied) even while phase 1 is `Reconciling`.

## 8. Final verification

- `make verify` passes
- `make test-unit` passes
- `make test-e2e` passes (all existing + new scenarios)

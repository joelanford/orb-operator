# Requirements

## API

- Add `PhaseStatusInvalid PhaseStatus = "Invalid"` to the `PhaseStatus` enum in `api/v1alpha1/types_clusterobjectset.go`.
- Update the `ObservedPhase` CEL validation to accept `Invalid` as a valid `PhaseStatus` value.
- Update godoc: `Invalid` indicates the phase failed preflight validation and is terminal for this COS revision.

## Wrapper revision engine

- Introduce a `revisionEngine` type in `internal/controller/` that wraps boxcutter's `RevisionEngine` and `PhaseEngine`.
- The wrapper is constructed with the same `RevisionEngineOptions` used today, plus the existing `ObservedPhases` (to read `completedAt`).
- Its `Reconcile` method returns a result type that `observedPhasesFromReconcileResult` can consume without changes to the caller in `doReconcileActive` (or with minimal changes).

## Preflight error mapping

- When `RevisionEngine.Reconcile` returns a `RevisionValidationError`, the wrapper maps it to per-phase statuses:
  - Phases listed in `RevisionValidationError.Phases` → `Invalid`, with `Error` and `IncompleteObjects` populated from the structured validation error tree.
  - Phases not listed → `Unknown`, with `Error` = `"Blocked by preflight errors in other phases"`.
- When a phase-level `PhaseValidationError` is returned during the gated reconcile loop (per-phase preflight), the phase is `Invalid` with the same error mapping used today.

## Disambiguated Unknown

- When a phase is `Unknown` because the gated loop stopped at an earlier incomplete phase, set `ObservedPhase.Error` = `"Waiting for earlier phases to complete"`.
- When a phase is `Unknown` because of revision-level preflight failure, set `ObservedPhase.Error` = `"Blocked by preflight errors in other phases"`.

## Steady-state drift correction

- After the gated `RevisionEngine.Reconcile`, for each phase that was skipped (not in the phase results) AND has `completedAt` set in the existing `ObservedPhases`:
  - Call `PhaseEngine.Reconcile` directly with the phase's objects and `WithAggregatePhaseReconcileErrors()`.
  - Build an `ObservedPhase` from the `PhaseResult` using the same logic as the gated reconcile (`Reconciling` with `IncompleteObjects`, or `Available`).
- Phases that were skipped and do NOT have `completedAt` remain `Unknown` with the gating message.
- **Error handling:** A non-nil error from `RevisionEngine.Reconcile` or any drift-correction `PhaseEngine.Reconcile` call means short-circuit — no further drift-correction phases are attempted. The wrapper treats boxcutter's error/nil-error boundary as the contract and does not interpret error types. If boxcutter returns a non-nil error in a case where it should have aggregated and continued, that's a boxcutter bug — write a failing test that proves the bug exists so we can fix it upstream.

## Boxcutter dependency

- Bump boxcutter to latest `main` (`v0.14.1-0.20260710084406-8f7a02854da8`).

## Acceptance Criteria

- `Invalid` status appears in COS status when a phase has preflight errors (bad metadata, namespace scope violations, dry-run failures, cross-phase duplication).
- All phases without errors show `Unknown` with the "blocked by preflight" message when any phase is `Invalid`.
- During normal gated rollout, unevaluated phases show `Unknown` with the "waiting" message.
- A completed phase that regresses (object deleted, drifted, probe failure) while an earlier phase is also incomplete gets active drift correction (re-applied, not just status-reported).
- A phase that has never completed stays gated behind earlier phases regardless of drift.
- Existing e2e scenarios continue to pass.
- New e2e scenarios cover: preflight failure with `Invalid` status, disambiguated `Unknown` messages, and drift correction for completed-but-skipped phases.

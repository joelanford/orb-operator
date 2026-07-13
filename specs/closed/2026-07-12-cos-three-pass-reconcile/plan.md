# Implementation Plan

1. **Replace `driftPhases` with `splitPhases`**
   - Add `splitPhases(rev, gatedPhaseNames, completedPhases) (drift, readOnly []types.Phase)` in `internal/revision/engine.go`.
   - Add `phasesAfter(rev, gatedPhaseNames, afterName) []types.Phase` helper.
   - Drift group: previously completed phases + gate+1 (same boundary as current `driftPhases`).
   - Read-only group: all remaining phases.
   - Remove the `driftPhases` iterator.
   - Unit test the split function with cases: all gated, all completed, mix, single phase, no completed phases, no phases.

2. **Add read-only pass to `Engine.Reconcile`**
   - After the drift loop, iterate over the read-only slice from `splitPhases`.
   - Call `e.phase.Reconcile(ctx, rev.GetRevisionNumber(), phase, append(phaseOpts, types.WithPaused{})...)` for each read-only phase.
   - Collect results into `readOnlyResults`.
   - Only run when no drift error occurred.
   - Stop on error within the read-only pass (same pattern as drift loop).

3. **Extend `Result` struct**
   - Add `readOnlyResults []machinery.PhaseResult` field.
   - Update `GetPhases()` to return gated + drift + readOnly.
   - `IsComplete()` and `HasProgressed()` remain based on gated + drift only.
   - Unit test with mixed results.

4. **API type changes**
   - Add `PhaseStatusPending` and `PhaseStatusWaitingForAssertions` to the `PhaseStatus` enum.
   - Add `ObjectCounts` struct (Total, Synced, Available as int64) to `ObservedPhase`.
   - Rename `Error` → `Message`, `IncompleteObjects` → `ObjectDetails`.
   - Run `make generate`.

5. **Update status builders**
   - Rename `reconcilingPhase` to `incompletePhase`. This single builder handles all non-complete phase results:
     - Paused objects → `Pending` status
     - All objects synced → `WaitingForAssertions` status
     - Otherwise → `Reconciling` status
   - The builder uses `IsPaused()` on each object result to detect read-only phases (boxcutter sets this uniformly per phase when `WithPaused{}` is used).
   - Rename `messagesForObject` to `objectMessages`.
   - Add `pausedObjectMessages` for read-only object message generation (ActionCreated, ActionRecovered, default).
   - Add `compareDetails` for field-level diff extraction from boxcutter's `CompareResult`.
   - Populate `ObjectCounts` in all status builder paths (complete, incomplete, unknown, superseded, teardown).
   - Unit tests for: `incompletePhase` with validation errors, collision (Reconciling), synced-but-probes-failing (WaitingForAssertions), paused objects (Pending); `pausedObjectMessages` branches; `compareDetails` with added/modified/removed fields.

6. **Verify**
   - `make verify` (lint, generate diff, build).
   - `make test-unit`.
   - `make test-e2e`.

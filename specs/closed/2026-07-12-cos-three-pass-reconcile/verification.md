# Verification

## Implementation Correctness

- [x] `splitPhases` produces the same drift boundary as the old `driftPhases` iterator.
- [x] `splitPhases` puts all remaining phases into the read-only group.
- [x] Read-only pass uses `types.WithPaused{}` — no cluster writes occur.
- [x] Read-only pass only runs when drift pass has no error.
- [x] `Result.GetPhases()` returns gated + drift + readOnly in phase order.
- [x] `Result.IsComplete()` ignores read-only results.
- [x] `Result.HasProgressed()` ignores read-only results.
- [x] `PhaseStatusPending` is added to the enum with correct validation marker.
- [x] `PhaseStatusWaitingForAssertions` is added to the enum with correct validation marker.
- [x] `incompletePhase` maps paused phases to `Pending`, all-synced phases to `WaitingForAssertions`, and others to `Reconciling`.
- [x] `incompletePhase` handles phase-level validation errors by returning `Invalid` status.
- [x] `Pending` phases carry `objectDetails` with context-specific messages for read-only objects.
- [x] `pausedObjectMessages` returns correct messages for `ActionCreated`, `ActionRecovered`, and default actions.
- [x] `compareDetails` extracts added, modified, and removed field paths from `CompareResult`.
- [x] `ObjectCounts.Total` matches the spec phase object count for all phase statuses.
- [x] `ObjectCounts.Synced` counts synced-action objects for gated/drift phases, `ActionIdle` objects for read-only phases.
- [x] `ObjectCounts.Available` counts `IsComplete() == true` objects for all phases.
- [x] Invariant `total >= synced >= available` holds per-phase in all test cases.
- [x] `ObservedPhase.Error` renamed to `ObservedPhase.Message`.
- [x] `ObservedPhase.IncompleteObjects` renamed to `ObservedPhase.ObjectDetails`.
- [x] Existing e2e scenarios pass (read-only pass is additive; previously `Unknown` phases now show `Pending` or `Available`).

## Project Conventions

- [x] No `//nolint` comments added (except two pre-existing defensive guards in `engine.go`).
- [x] Code formatted with gofumpt.
- [x] `make lint` passes.
- [x] `make verify` passes (includes lint, generate diff, goreleaser check, build).
- [x] Unit tests use testify assert/require.
- [x] New status fields have godoc comments following existing conventions.
- [x] `make test-unit` passes.
- [x] `make test-e2e` passes.

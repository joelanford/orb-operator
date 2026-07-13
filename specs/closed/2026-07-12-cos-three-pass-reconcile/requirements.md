# Requirements

- The COS reconcile loop runs three passes: gated (active), drift (active), read-only.
- The read-only pass evaluates all phases not covered by gated or drift using `types.WithPaused{}`.
- The read-only pass does not write to the cluster.
- The read-only pass only runs when the drift pass completes without error.
- The read-only pass reports per-object results with accurate `Action` and `IsComplete()` values.
- The `driftPhases` iterator is replaced with a `splitPhases` function returning two slices (drift, readOnly).
- The drift group includes previously completed phases plus the gate+1 phase.
- The read-only group includes all remaining phases after the drift boundary.
- The `Result` struct includes read-only results accessible via `GetPhases()`.
- `Result.IsComplete()` and `Result.HasProgressed()` are unchanged — they do not consider read-only results.
- A new `PhaseStatusPending` value is added to the `PhaseStatus` enum for read-only phases with incomplete objects.
- A new `PhaseStatusWaitingForAssertions` value is added for phases where all objects are synced but assertions are not passing.
- Read-only phases with incomplete objects report as `Pending` with `objectDetails` listing what would need to change.
- Read-only phases with all objects matching report as `Available`.
- Active phases where all objects are synced but probes are failing report as `WaitingForAssertions`.
- `ObservedPhase` includes an `ObjectCounts` struct with `Total`, `Synced`, and `Available` (int64) fields.
- The `Error` field on `ObservedPhase` is renamed to `Message`.
- The `IncompleteObjects` field on `ObservedPhase` is renamed to `ObjectDetails`.
- Per-phase counts are populated for all phase statuses (Available, Reconciling, WaitingForAssertions, Pending, etc.).
- Synced counts: gated/drift phases count `ActionIdle`, `ActionUpdated`, `ActionCreated`, and `ActionRecovered`; read-only phases count `ActionIdle` only.
- Available counts: all phases count `IsComplete() == true` objects.
- The invariant `total >= synced >= available` holds per-phase.
- Paused objects that are not idle carry context-specific messages (e.g. "object does not exist", "object was modified by another actor") with field-level diff details when available.

## Acceptance Criteria

- Read-only pass runs on phases not covered by gated or drift, with no cluster writes.
- Per-phase counts are accurate: synced reflects cluster state match, available reflects probes passing.
- Existing gated and drift reconcile behavior is unchanged.
- `incompletePhase` determines status as `Pending` (paused), `WaitingForAssertions` (all synced), or `Reconciling` (otherwise).
- `Pending` phases carry `objectDetails` with context-specific messages for read-only objects.
- Unit tests cover: all phases gated (no read-only), all phases completed (no read-only), mix of drift and read-only, objects matching vs not matching in read-only phases, per-phase count correctness, validation errors in phase results, collision producing Reconciling status, `pausedObjectMessages` branches, `compareDetails` field-level diffs.
- `make verify` passes.
- `make test-e2e` passes.

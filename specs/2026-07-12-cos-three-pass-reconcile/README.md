---
status: in-progress
---
# COS Three-Pass Reconcile

## Summary

Extend the COS reconcile loop from two passes (gated + drift) to three
passes (gated + drift + read-only) to detect which objects in
unevaluated phases already match the desired state on the cluster. This
enables per-phase synced and available object counts in the COS status,
which is a prerequisite for the printer column improvements work item.

## Design

### Current two-pass model

The `revision.Engine.Reconcile` method runs two passes:

1. **Gated** — boxcutter's `RevisionEngine.Reconcile` applies phases
   sequentially, stopping when a phase is incomplete.
2. **Drift** — `PhaseEngine.Reconcile` re-checks previously completed
   phases (plus the gate+1 phase) to catch regressions.

Phases beyond the drift boundary are not evaluated at all.

### New three-pass model

Add a third pass:

3. **Read-only** — `PhaseEngine.Reconcile` with `types.WithPaused{}`
   runs remaining unevaluated phases without writing. Boxcutter reads
   the cluster state and reports what action it *would* take:
   - `ActionIdle` → object already matches desired state (synced)
   - `ActionCreated` / `ActionUpdated` / `ActionRecovered` → object
     does not match (not synced, and the write was skipped)
   - `ActionProgressed` → object adopted by another revision

### Object state definitions

For each object result from any of the three passes:

- **Synced**: the object on the cluster matches the desired state.
  - Gated/drift phases: `ActionIdle`, `ActionUpdated`, `ActionCreated`,
    or `ActionRecovered` (all count as synced since the write was
    applied).
  - Read-only phases: `Action() == ActionIdle`.
- **Available**: the object is synced AND passes its probes.
  - All phases: `IsComplete() == true`.

The invariant `total >= synced >= available` holds per-phase and when
summed across phases.

### Phase split function

Replace the current `driftPhases` iterator with a function that splits
non-gated phases into two slices in one pass:

```go
func splitPhases(
    rev types.Revision,
    gatedPhaseNames map[string]struct{},
    completedPhases map[string]bool,
) (drift []types.Phase, readOnly []types.Phase)
```

- **Drift group**: previously completed phases plus the gate+1 phase
  (same boundary as today). These get active reconciliation.
- **Read-only group**: all remaining phases after the drift boundary.
  These get read-only evaluation via `types.WithPaused{}`.

A helper `phasesAfter` returns all non-gated phases following a named
phase in the revision's phase order.

### Extending the Result type

Add a `readOnlyResults []machinery.PhaseResult` field to the existing
`revision.Result` struct:

```go
type Result struct {
    gated           machinery.RevisionResult
    driftResults    []machinery.PhaseResult
    readOnlyResults []machinery.PhaseResult
}
```

`GetPhases()` returns all three groups concatenated.

`IsComplete()` and `HasProgressed()` remain based on gated + drift
only — read-only results do not affect reconcile completeness.

### New phase statuses

Add two new values to the `PhaseStatus` enum:

- `PhaseStatusPending` — the phase is not being actively reconciled,
  but the controller has checked and some objects do not match the
  desired state. The `objectDetails` field lists what would need to
  change. Used for read-only phases with incomplete objects.

- `PhaseStatusWaitingForAssertions` — all objects in this phase are
  synced but some assertions are not yet passing. The controller is
  not actively writing; it is waiting for other controllers or
  external actions to bring the objects into the expected state. Used
  for both active and read-only phases where all objects are synced
  but probes are failing.

`Unknown` now only applies when a phase genuinely wasn't evaluated
(e.g. validation error blocked the reconcile before reaching it, or
the read-only pass was skipped due to a drift error).

### Per-phase counts in ObservedPhase

Add an `ObjectCounts` struct to `ObservedPhase`:

```go
type ObjectCounts struct {
    Total     int64 `json:"total"`
    Synced    int64 `json:"synced"`
    Available int64 `json:"available"`
}

type ObservedPhase struct {
    // ... existing fields ...
    ObjectCounts ObjectCounts `json:"objectCounts"`
}
```

- `Total`: number of objects in the phase (from spec).
- `Synced`: objects where the cluster state matches the desired state.
  - Gated/drift phases: `ActionIdle`, `ActionUpdated`, `ActionCreated`,
    or `ActionRecovered`.
  - Read-only phases: `ActionIdle` only.
- `Available`: objects that are synced and pass probes.
  - All phases: `IsComplete() == true`.

COS-level and COD-level totals are derived by summing across observed
phases.

### Field renames

Two fields on `ObservedPhase` are renamed:

- `error` → `message` — broadened from error-only to a general status
  message (e.g. a summary for Pending phases, not just validation
  errors).
- `incompleteObjects` → `objectDetails` — reflects broader usage for
  both active and read-only phases.

### Status integration

The COS status updater (`internal/status/cos`) computes the per-phase
counts from the phase results. A single `incompletePhase` builder
function handles all non-complete, non-teardown phase results and
determines the status based on object state:

- All objects paused → `Pending` (read-only phase not yet reconciled)
- All objects synced → `WaitingForAssertions` (synced but probes
  failing)
- Otherwise → `Reconciling` (active phase with unsynced objects)

### Read-only object messages

For paused objects that are not idle, `pausedObjectMessages` generates
context-specific messages:

- `ActionCreated` → `"object does not exist"`
- `ActionRecovered` → `"object was modified by another actor"` with
  field-level diff details
- Other actions → `"object content has changed"` with field-level
  diff details

Field-level diffs are extracted by `compareDetails` from boxcutter's
`CompareResult`, listing added, modified, and removed field paths.

---
status: in-progress
---
# COS Status API Redesign: Preflight, Phase Gating, and Steady-State Drift Correction

## Summary

Redesign how the COS controller populates `ObservedPhase` status to accurately reflect all stages of the reconcile lifecycle — preflight validation, gated rollout, and steady-state drift correction. The current status model doesn't distinguish why a phase is `Unknown` or capture preflight errors structurally. Additionally, the gated reconcile loop prevents drift correction in later phases when an earlier phase regresses, even though those later phases have already completed initial rollout.

## Motivation

Three gaps in the current status reporting:

1. **Preflight errors aren't surfaced per-phase.** When revision-level preflight fails, there's no way to distinguish "this phase has invalid objects" from "this phase just hasn't been evaluated yet."

2. **`Unknown` is ambiguous.** A phase can be `Unknown` because (a) the entire revision is broken by preflight errors, or (b) it's simply waiting for an earlier phase to complete during normal rollout. These are operationally different — one is terminal for this COS, the other is transient.

3. **Steady-state drift in later phases is masked.** Boxcutter's `RevisionEngine.Reconcile` stops at the first incomplete phase. If a completed phase 1 regresses (object deleted, drifted, or probe failure), phases 2+ are neither corrected nor status-reported, even though they completed previously and could independently self-heal.

## Design

### New `PhaseStatus` value: `Invalid`

Add `Invalid` to the `PhaseStatus` enum. A phase is `Invalid` when it fails preflight validation. Some errors are permanent (e.g. cross-phase duplication), while others may resolve on a future reconcile (e.g. a missing CRD is installed).

- `ObservedPhase.Error` carries the phase-level error (e.g. invalid phase name).
- `ObservedPhase.IncompleteObjects` carries per-object validation errors (bad metadata, namespace scope violations, dry-run failures, cross-phase duplication) with details in `ObjectStatus.Messages`.

### Disambiguated `Unknown` messages

When a phase is `Unknown`, the `ObservedPhase.Error` field explains why:

- **"Blocked by preflight errors in other phases"** — revision-level preflight failed. Other phases are `Invalid`. This COS will never reconcile; create a new revision.
- **"Waiting for earlier phases to complete"** — normal gated rollout. An earlier phase is `Reconciling` and hasn't completed yet. This is transient.

### Steady-state drift correction for completed phases

After the gated `RevisionEngine.Reconcile` call, the COS controller calls `PhaseEngine.Reconcile` directly for any phases that:
- Were skipped by the gated loop (an earlier phase is incomplete), AND
- Have `completedAt` set in COS status (they previously completed initial rollout)

This gives completed phases active drift correction (writes, not just status) even when an earlier phase has regressed. The `completedAt` timestamp is the key — a phase that has never completed stays gated behind earlier phases (initial rollout safety), while a phase that has completed before gets independent maintenance.

### Preflight error flow

1. Run `RevisionEngine.Reconcile`. If `RevisionValidationError` is returned:
   - Phases with errors → `Invalid`, with `Error`/`IncompleteObjects` populated from the structured validation error tree.
   - Phases without errors → `Unknown`, with `Error` = "Blocked by preflight errors in other phases".
   - Stop. No phase-level or object-level reconciliation occurs.

2. If revision preflight passes, the gated reconcile loop runs per-phase preflight (namespace scope, dry-run) just before each phase is applied. A phase-level preflight failure also results in `Invalid`.

### Reconcile flow (no preflight errors)

1. Gated `RevisionEngine.Reconcile` iterates phases in order:
   - Phase completes → `Available`
   - Phase incomplete (probe failure, collision, apply error) → `Reconciling` with `IncompleteObjects`
   - Stops at first incomplete phase. Remaining phases → `Unknown` with "Waiting for earlier phases to complete"

2. For phases skipped by the gated loop that have `completedAt` set:
   - Call `PhaseEngine.Reconcile` directly for active drift correction.
   - Report `Reconciling` or `Available` based on result.

### Full `PhaseStatus` state space

| Status | Meaning | Terminal? |
|---|---|---|
| `Invalid` | Failed preflight validation | Depends on error |
| `Unknown` | Not evaluated; message explains why | Depends on cause |
| `Reconciling` | Actively evaluated, not yet complete | No |
| `Available` | All objects reconciled, all assertions pass | No (can regress) |
| `Superseded` | Objects adopted by a newer revision | Yes |
| `TearingDown` | Reverse-order deletion in progress | No |
| `TeardownComplete` | All objects deleted | Yes |

## Deliverables

- Add `Invalid` to `PhaseStatus` enum and update godoc
- Populate `Invalid` status with structured error detail from `RevisionValidationError` and `PhaseValidationError`
- Set disambiguated `Unknown` messages based on cause (preflight blocked vs. gating)
- After gated reconcile, call `PhaseEngine.Reconcile` directly for completed-but-skipped phases
- Update e2e tests covering preflight failures, normal gating, and steady-state regression scenarios

## Architecture

The implementation introduces a wrapper revision engine (`revisionEngine` in `internal/controller/`) that composes boxcutter's `RevisionEngine` and `PhaseEngine`. The wrapper satisfies the same call pattern as `RevisionEngine.Reconcile` today, making the change transparent to `doReconcileActive`. Internally it:

1. Delegates to `RevisionEngine.Reconcile` for the gated reconcile loop
2. Calls `PhaseEngine.Reconcile` directly for completed-but-skipped phases (drift correction)
3. Returns a composite result that includes phase results from both the gated loop and drift correction

Validation errors, `Invalid`/`Unknown` status mapping, and all other COS status concerns remain in `observedPhasesFromReconcileResult` — the wrapper is purely a reconcile-loop concern, not a status concern.

## Dependencies

- Requires bumping boxcutter to latest `main` (`v0.14.1-0.20260710084406-8f7a02854da8`) for:
  - Pointer-receiver fix on `PhaseValidationError` (value → `*validation.PhaseValidationError`)
  - `PhaseResult.IsComplete()` correctly returns `false` when validation error is present
  - `NoMatchError` handling in dry-run validation

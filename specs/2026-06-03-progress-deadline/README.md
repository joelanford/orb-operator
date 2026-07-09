---
status: in-progress
---
# COD Progress Deadline

## Summary

Add `progressDeadlineMinutes` to the ClusterObjectDeployment spec and per-phase `completedAt` timestamps to COS status. Together, these let the COD controller detect stalled rollouts and surface a `Progressing` condition — mirroring Kubernetes Deployment semantics using a purely level-based evaluation.

## Design

### Progress signal: per-phase `completedAt`

Each `ObservedPhase` in COS status gains a `completedAt *metav1.Time` field, set when the phase first becomes Available and immutable thereafter. This parallels the existing COS-level `completedAt` but at phase granularity.

The COS controller sets `completedAt` in `doReconcileActive` by comparing the newly computed phase status against the previous status. When a phase transitions to Available and has no existing `completedAt`, it records the current time. Existing `completedAt` values are preserved across reconciles (immutable once set).

### Deadline evaluation (level-based)

The COD controller evaluates the deadline using only current-state timestamps — no edge detection or stored history:

```
lastMilestone = max(cos.creationTimestamp, max(phase.completedAt for all phases with completedAt set))
exceeded = time.Since(lastMilestone) > progressDeadlineMinutes AND cos is not fully available
```

This mirrors how Kubernetes Deployment uses `LastUpdateTime` on the `Progressing` condition, but avoids the need for `LastUpdateTime` (which `metav1.Condition` lacks) by using per-phase completion timestamps instead.

### COD `Progressing` condition

A new `Progressing` condition type is added to COD status, orthogonal to the existing `Available` condition:

| Progressing | Reason | When |
|---|---|---|
| `True` | `NewClusterObjectSetProgressing` | Latest COS is rolling out, within deadline (or no deadline set) |
| `True` | `NewClusterObjectSetProgressed` | Latest COS is fully available |
| `False` | `ProgressDeadlineExceeded` | Rollout stalled, deadline exceeded |

When `progressDeadlineMinutes` is unset (nil), no deadline is evaluated. The `Progressing` condition is still set — it just never transitions to `ProgressDeadlineExceeded`.

### Requeue for deadline expiry

The COD controller returns `RequeueAfter: remainingDeadline` when a deadline is active and not yet exceeded. This ensures the controller wakes up at the right moment to set `Progressing=False`, even if no COS status changes occur.

### E2e testing: `ORB_DEADLINE_DURATION_UNIT_OVERRIDE` override

Waiting real minutes in e2e tests is impractical. When `ORB_DEADLINE_DURATION_UNIT_OVERRIDE` is set to a `time.Duration` string (e.g., `1ms`, `1s`), the operator uses that value as the multiplier for the `progressDeadlineMinutes` field instead of the default `time.Minute`. This is implemented as a `time.Duration` multiplier injected into `CODReconciler` at construction — the controller logic is identical, only the unit changes. The env var is read and parsed with `time.ParseDuration` in `main.go` and is not part of the public API.

### Scope

- Deadline applies to the **latest active COS** only. Predecessor revisions being superseded are not subject to deadline evaluation.
- No automatic rollback or archiving on deadline exceeded — the condition is purely informational. Higher-level controllers or users decide what action to take.
- `Progressing` transitions to `NewClusterObjectSetProgressed` as soon as the latest COS is available. It does not wait for predecessor COSs to finish teardown. A future enhancement could keep `Progressing=True, Reason=TearingDown` until all predecessors are fully archived.
- COD-level phase aggregation (unioning `observedPhases` across active COSs) is a separate work item.

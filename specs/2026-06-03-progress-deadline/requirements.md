# Requirements

- COD spec gains a `progressDeadlineMinutes *int32` field. When set, the COD controller evaluates whether the latest active COS is making forward progress. When nil, no deadline is evaluated.
- COS `ObservedPhase` gains a `completedAt *metav1.Time` field. Set once when the phase first becomes Available. Immutable thereafter.
- COD status gains a `Progressing` condition (alongside existing `Available`).
- When the deadline expires without a new phase completing, the COD sets `Progressing=False` with reason `ProgressDeadlineExceeded`.
- The COD controller requeues after the remaining deadline interval so it can detect expiry even without COS status changes.
- No automatic rollback or archiving on deadline expiry.
- When `progressDeadlineMinutes` is nil, the `Progressing` condition still reflects rollout state (True/NewClusterObjectSetProgressing or True/NewClusterObjectSetProgressed) — it just never transitions to `ProgressDeadlineExceeded`.

## Acceptance Criteria

- A COD with `progressDeadlineMinutes` set and a COS that doesn't complete any phases within the deadline reports `Progressing=False, Reason=ProgressDeadlineExceeded`.
- A COD with `progressDeadlineMinutes` set and a COS that completes phases incrementally within the deadline reports `Progressing=True, Reason=NewClusterObjectSetProgressing`.
- A COD with `progressDeadlineMinutes` set and a fully available COS reports `Progressing=True, Reason=NewClusterObjectSetProgressed`.
- A COD without `progressDeadlineMinutes` never reports `ProgressDeadlineExceeded`.
- Per-phase `completedAt` is set exactly once when the phase first becomes Available and preserved on subsequent reconciles.
- Per-phase `completedAt` is nil for phases that have never been Available (Reconciling, Unknown, Superseded, etc.).
- The deadline resets when a new COS is created (new revision), since the clock starts from `cos.creationTimestamp`.

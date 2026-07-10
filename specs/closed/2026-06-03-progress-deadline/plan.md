# Implementation Plan

1. **API type changes**
   - Add `ProgressDeadlineMinutes *int32` to `ClusterObjectDeploymentSpec` with kubebuilder validation (`Minimum=1`).
   - Add `CompletedAt *metav1.Time` to `ObservedPhase`.
   - Add condition type `ConditionTypeProgressing = "Progressing"` and reasons `ReasonNewClusterObjectSetProgressing`, `ReasonNewClusterObjectSetProgressed`, `ReasonProgressDeadlineExceeded` to `types.go`.
   - Run `make generate` to regenerate deepcopy, CRDs, and apply configurations.

2. **COS controller: per-phase `completedAt`**
   - In `reconcileActive`, after `doReconcileActive` computes new `observedPhases`, call a new function `preservePhaseCompletionTimes(existing.Status.ObservedPhases, cos.Status.ObservedPhases)` that:
     - Builds a name→completedAt map from the existing (pre-reconcile) phases.
     - For each new phase: if it has status Available and no completedAt in the existing map, set `completedAt = now()`. If the existing map has a completedAt for it, copy that value (immutability).
   - Unit test: verify completedAt is set on first Available, preserved on re-reconcile, and nil for non-Available phases.

3. **COD controller: deadline unit and `Progressing` condition**
   - Add a `deadlineUnit time.Duration` field to `CODReconciler`. Default to `time.Minute`. `NewCODReconciler` accepts it as a parameter.
   - In `main.go`, read `ORB_DEADLINE_DURATION_UNIT_OVERRIDE`. When set, parse it with `time.ParseDuration` and pass the result to `NewCODReconciler`; otherwise pass `time.Minute`.
   - Extract deadline evaluation into a new function `evaluateProgressDeadline(cod, latestCOS) (metav1.Condition, *time.Duration)` that returns the Progressing condition and an optional requeue duration. Uses `r.deadlineUnit` instead of hardcoding `time.Minute`.
   - Logic:
     - If no active revisions → `Progressing=False, Reason=ReasonUnavailable` (no requeue).
     - If latest COS has ever been fully available (`cos.Status.CompletedAt != nil`) → `Progressing=True, Reason=NewClusterObjectSetProgressed` (no requeue). This uses `CompletedAt` rather than the current `Available` condition to prevent the deadline from re-triggering after a post-rollout regression.
     - If latest COS has never been available and no deadline set → `Progressing=True, Reason=NewClusterObjectSetProgressing` (no requeue).
     - If latest COS has never been available and deadline set:
       - Compute `lastMilestone = max(cos.creationTimestamp, max(phase.completedAt for phases with completedAt))`.
       - `elapsed = time.Since(lastMilestone)`, `deadline = progressDeadlineMinutes * time.Minute`.
       - If `elapsed >= deadline` → `Progressing=False, Reason=ProgressDeadlineExceeded` (no requeue).
       - Else → `Progressing=True, Reason=NewClusterObjectSetProgressing` (requeue after `deadline - elapsed`).
   - Call from `setStatus` and use the requeue duration in `reconcile`'s return value.
   - Unit test: verify all condition states and requeue duration computation.

4. **Wire up `RequeueAfter` in COD reconciler**
   - Change `reconcile` to propagate the requeue duration from `evaluateProgressDeadline` into the `ctrl.Result` returned by `Reconcile`.

5. **E2e test infrastructure**
   - Set `ORB_DEADLINE_DURATION_UNIT_OVERRIDE=1ms` on the operator deployment in `deploy/operator.jsonnet` (e2e-only configuration).
   - E2e tests use small `progressDeadlineMinutes` values (e.g., 500) which are interpreted as 500ms.

6. **E2e test scenarios**
   - Scenario: COD with `progressDeadlineMinutes` set and a COS whose objects never pass assertions → verify `Progressing=False, ProgressDeadlineExceeded` after the deadline.
   - Scenario: COD with `progressDeadlineMinutes` and a COS that completes all phases → verify `Progressing=True, NewClusterObjectSetProgressed` and per-phase `completedAt` timestamps are set.
   - Scenario: COD without `progressDeadlineMinutes` and a slow COS → verify `Progressing=True, NewClusterObjectSetProgressing` (never exceeded).

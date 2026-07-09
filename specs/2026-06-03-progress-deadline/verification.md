# Verification

## Implementation Correctness

- [ ] `ProgressDeadlineMinutes` field exists on `ClusterObjectDeploymentSpec` with `Minimum=1` validation.
- [ ] `CompletedAt` field exists on `ObservedPhase` with correct JSON tag and optional semantics.
- [ ] New condition type and reason constants are defined in `types.go`.
- [ ] CRDs, deepcopy, and apply configurations regenerated (`make generate` + `hack/diff.sh generate` passes).
- [ ] COS controller sets `completedAt` on phase's first transition to Available.
- [ ] COS controller preserves existing `completedAt` values (immutability).
- [ ] COS controller does not set `completedAt` for non-Available phases.
- [ ] COD controller sets `Progressing` condition with correct status/reason for all cases: available, progressing within deadline, deadline exceeded, no deadline set, no active revisions.
- [ ] COD controller returns `RequeueAfter` with remaining deadline time when deadline is active and not exceeded.
- [ ] COD controller does not requeue when no deadline is set or deadline is already exceeded.
- [ ] Deadline evaluation uses `max(cos.creationTimestamp, max(phase.completedAt))` as the last milestone.
- [ ] `CODReconciler` uses the injected `deadlineUnit` rather than hardcoding `time.Minute`.
- [ ] `main.go` reads `ORB_DEADLINE_DURATION_UNIT_OVERRIDE`, parses it with `time.ParseDuration` when set, defaults to `time.Minute` otherwise.
- [ ] E2e operator deployment sets `ORB_DEADLINE_DURATION_UNIT_OVERRIDE=1ms`.
- [ ] Unit tests cover `preservePhaseCompletionTimes` and `evaluateProgressDeadline`.
- [ ] E2e tests cover deadline exceeded, rollout success, and no-deadline scenarios.

## Project Conventions

- [ ] No `//nolint` comments added.
- [ ] `make verify` passes (lint, generate diff, goreleaser check, build).
- [ ] `make test-unit` passes with no coverage decrease.
- [ ] `make test-e2e` passes.
- [ ] New code follows standard controller-runtime patterns (reconcile loops, conditions, requeue).
- [ ] New API fields have godoc comments consistent with existing field documentation style.
- [ ] Kubebuilder markers match existing conventions (validation, optional/required, list types).
- [ ] Commit messages use conventional commits format.

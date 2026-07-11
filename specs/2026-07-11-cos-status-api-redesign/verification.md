# Verification

## Implementation Correctness

- [ ] `PhaseStatusInvalid` is defined in `api/v1alpha1/types_clusterobjectset.go` with godoc
- [ ] CEL validation on `ObservedPhase.Status` accepts `Invalid`
- [ ] CRDs regenerated with `make generate` and committed
- [ ] Boxcutter bumped to `v0.14.1-0.20260710084406-8f7a02854da8` (or newer tagged release)
- [ ] Wrapper `revisionEngine` exists in `internal/controller/` and composes `RevisionEngine` + `PhaseEngine`
- [ ] `doReconcileActive` uses the wrapper instead of bare `RevisionEngine`
- [ ] `observedPhasesFromReconcileResult` populates `Invalid` statuses from `RevisionValidationError.Phases`
- [ ] `observedPhasesFromReconcileResult` populates `Unknown` with "Blocked by preflight errors in other phases" for non-errored phases when a revision validation error is present
- [ ] `mapSpecPhases` sets `Error` = "Waiting for earlier phases to complete" on unevaluated phases during normal gating
- [ ] Drift correction calls `PhaseEngine.Reconcile` only for phases with `completedAt` set
- [ ] Drift correction does NOT call `PhaseEngine.Reconcile` for phases that have never completed
- [ ] Drift correction short-circuits on non-nil error from `PhaseEngine.Reconcile` (no caller-side error interpretation)
- [ ] Unit tests cover all three new behaviors: Invalid mapping, Unknown disambiguation, drift correction
- [ ] E2e scenarios cover: preflight failure, normal gating messages, drift correction for completed-but-skipped phases

## Project Conventions

- [ ] Commit messages use conventional commits format (`feat:`, `test:`, `chore:`)
- [ ] No `//nolint` comments added
- [ ] `make verify` passes (lint, generate diff, goreleaser check, build)
- [ ] `make test-unit` passes with no coverage decrease
- [ ] `make test-e2e` passes (all existing + new scenarios)
- [ ] New code follows idiomatic controller-runtime patterns (per `specs/mission.md`)
- [ ] API changes are consistent with ADR-0001 (per `specs/mission.md`)
- [ ] Go tools invoked via `go tool <name>` (per `specs/tech-stack.md`)

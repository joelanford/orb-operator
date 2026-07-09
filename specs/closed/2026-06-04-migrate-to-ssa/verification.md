# Verification

## Implementation Correctness

- [x] Apply configurations generated and committed (`applyconfigurations/`)
- [x] `make generate` reproduces `applyconfigurations/` without diff
- [x] `applyCOSR` helper short-circuits via `needsApply` predicate
- [x] COS controller writes (adoption, archival, creation, field ownership reconciliation) use `cos-controller` field owner
- [x] COSR controller finalizer add uses `cosr-controller` field owner
- [x] Finalizer removal uses optimistic-lock merge patch, not SSA
- [x] `clearFinalizerFieldOwnership` removes the finalizer key from `cosr-controller` managed fields
- [x] `waitForFinalizerRemoval` polls cache after finalizer removal
- [x] COSR name validation rejects names > 128 chars or with leading/trailing whitespace

## Project Conventions

- [x] Commits use conventional commit format (`feat:`, `fix:`)
- [x] One logical change per commit
- [x] Uses idiomatic controller-runtime patterns (standard controller patterns principle)
- [x] No `//nolint` comments added
- [x] No new external dependencies introduced
- [x] Existing tests continue to pass

# Verification

## Implementation Correctness

- [ ] COS `objectCounts.total` = sum of `observedPhases[*].objectCounts.total`.
- [ ] COS `objectCounts.synced` = sum of `observedPhases[*].objectCounts.synced`.
- [ ] COS `objectCounts.available` = sum of `observedPhases[*].objectCounts.available`.
- [ ] COS `objectCounts` is zero-valued when `observedPhases` is nil or empty.
- [ ] COD `objectCounts` mirrors the latest active COS's `objectCounts`.
- [ ] COD `objectCounts` is zero-valued when no active COS exists.
- [ ] Invariant `total >= synced >= available` holds on COS and COD `objectCounts`.
- [ ] COS printer columns: NAME, GROUP, REV, AVAILABLE, SYNCED, TOTAL, LIFECYCLE, AGE.
- [ ] COD printer columns: NAME, AVAILABLE, SYNCED, TOTAL, AGE.
- [ ] Existing Available and Progressing conditions remain on COD and COS status.

## Project Conventions

- [ ] No `//nolint` comments added.
- [ ] Code formatted with gofumpt.
- [ ] `make lint` passes.
- [ ] `make verify` passes.
- [ ] Unit tests use testify assert/require.
- [ ] New status fields have godoc comments following existing conventions.
- [ ] `make test-unit` passes.
- [ ] `make test-e2e` passes.

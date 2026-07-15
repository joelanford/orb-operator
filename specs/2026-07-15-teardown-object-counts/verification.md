# Verification

## Implementation Correctness

- [ ] `ObjectCounts` has `Present int64` field between Total and Synced
- [ ] CEL validation enforces `objectCounts.present == observedPhases.sum(p, p.objectCounts.present)`
- [ ] `sumObjectCounts` sums Present alongside Total, Synced, Available
- [ ] Reconcile complete phases: present=total, synced=total, available=total, total=total
- [ ] Reconcile incomplete phases: present=count of processed objects
- [ ] Teardown complete phases: present=0, synced=0, available=0, total=total
- [ ] Tearing-down phases: present=len(waitingForDeletion), synced=0, available=0, total=total
- [ ] Gated teardown phases: cache-checked present count, synced=0, available=0, total=total
- [ ] Gated teardown phases are NOT marked Unknown - they have a definite status with real counts
- [ ] Printer columns: AVAILABLE, SYNCED, PRESENT, TOTAL (in that order)
- [ ] Existing e2e count step definitions updated to include `present`
- [ ] All ~30 existing e2e count assertions updated with correct `present` values
- [ ] Teardown scenarios assert final counts: present=0, synced=0, available=0, total=total
- [ ] Multi-phase teardown scenarios assert intermediate counts per phase state

## Project Conventions

- [ ] `make verify` passes (lint, build, generate, goreleaser check)
- [ ] `make test-unit` passes with no coverage decrease
- [ ] `make test-e2e` passes with teardown count assertions
- [ ] Commit messages use conventional format
- [ ] No `//nolint` comments added
- [ ] ADR-0001 compliance: changes are consistent with COS/COD architecture
- [ ] Standard controller patterns: cache reads follow controller-runtime idioms

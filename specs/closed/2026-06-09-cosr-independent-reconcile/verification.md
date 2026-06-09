# Verification

## Implementation Correctness

- [ ] COSR events enqueue self, not highest revision
- [ ] Managed object events enqueue owning COSR, not highest revision
- [ ] `mapToHighestRevInChain` and `managedObjectToHighestRevInChain` are removed
- [ ] Each COSR determines its role (latest, predecessor, archived, deleted) and dispatches accordingly
- [ ] Predecessors run a full boxcutter reconcile with siblings
- [ ] Predecessors pass latest active + other predecessors as siblings to boxcutter
- [ ] Predecessor `Available` condition uses reason `Superseded` with status reflecting reconcile result
- [ ] Archived COSR teardown behavior unchanged
- [ ] Deleted COSR teardown and finalizer release behavior unchanged
- [ ] Finalizer management unchanged

## Project Conventions

- [ ] No `//nolint` comments added
- [ ] `make verify` passes (lint, generate diff, build)
- [ ] `make test-unit` passes
- [ ] `make test-integration` passes
- [ ] `make test-e2e` passes
- [ ] All existing e2e scenarios pass without modification
- [ ] New e2e scenario: disjoint object sets across revisions
- [ ] New e2e scenario: predecessor object drift correction

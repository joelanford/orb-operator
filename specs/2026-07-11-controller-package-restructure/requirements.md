# Requirements

- Extract non-controller-specific logic from `internal/controller/` into independently testable packages under `internal/`
- Split the monolithic controller package into `internal/controller/cod/` and `internal/controller/cos/`
- Introduce typed errors (`ObjectResolutionError`, `InternalError`) as the contract between controller work functions and status updaters
- COS controller's `doReconcile`/`doTeardown` must be status-unaware — they return `(result, error)` with typed errors; status is applied by the caller via `cosstatus.Apply(cos, cosstatus.From*(…))`
- Shared reconcile setup (resolve → verify → engine → build revision) must be extracted into `resolveAndPrepare`, used by both `doReconcile` and `doTeardown`
- COD controller must extract `syncRevision` from the monolithic `reconcile` method
- COD status must be updated after `syncRevision` but before `archiveSuperseded`/`pruneArchived`
- Naming must be consistent: no redundant `COS`/`COD` suffixes on methods, symmetric names between active and teardown paths
- Zero e2e test changes — this is a pure internal refactor with no behavior change
- Zero changes to `test/e2e/` directory

## Acceptance Criteria

- All existing unit tests pass (migrated to new packages as appropriate)
- All existing e2e tests pass without modification — `test/e2e/` directory has zero diff
- Each new helper package has ≥70% unit test statement coverage (per `specs/conventions.md`)
- Overall project unit test coverage does not decrease from the current 24.7% baseline
- `make verify` passes (lint, generate, build)
- No import cycles between new packages
- Controller packages (`controller/cod/`, `controller/cos/`) depend on helper packages, never the reverse
- The old `internal/controller/` package is fully removed (no leftover files)

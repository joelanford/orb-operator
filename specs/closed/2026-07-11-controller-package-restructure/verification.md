# Verification

## Zero E2E Changes

- [ ] `git diff -- test/e2e/` produces no output — the e2e directory is completely untouched
- [ ] `make test-e2e` passes with no failures

## Unit Test Coverage

Current baseline: `internal/controller/` has 24.7% statement coverage.

- [ ] Each new helper package meets ≥70% statement coverage (per `specs/conventions.md`):
  - [ ] `internal/object/` ≥70%
  - [ ] `internal/status/cod/` ≥70%
  - [ ] `internal/status/cos/` ≥70%
  - [ ] `internal/revision/` ≥70%
  - [ ] `internal/cosutil/` ≥70%
  - [ ] `internal/template/` ≥70%
  - [ ] `internal/errors/` ≥70%
- [ ] Overall project unit test coverage (`make test-unit`) does not decrease from baseline
- [ ] `make test-unit` passes with no failures

## Build and Lint

- [ ] `make verify` passes (lint + generate + goreleaser check + build)
- [ ] No import cycles: `go vet ./...` passes
- [ ] No `//nolint` comments added

## Structural Correctness

- [ ] `internal/controller/` directory is fully removed (no leftover files)
- [ ] Controller packages (`controller/cod/`, `controller/cos/`) import helper packages, never the reverse
- [ ] No helper package imports from `controller/cod/` or `controller/cos/`
- [ ] `internal/errors/` has no dependencies on other internal packages

## Implementation Correctness

- [ ] COS `doReconcile` and `doTeardown` do not reference any status/condition functions — they only return `(result, error)` with typed errors
- [ ] COS status is applied in exactly one place per path via `cosstatus.Apply(cos, cosstatus.From*(…))`
- [ ] COS `resolveAndPrepare` is shared between `doReconcile` and `doTeardown`
- [ ] COD `updateStatus` is called after `syncRevision` but before `archiveSuperseded`/`pruneArchived`
- [ ] COD `archiveSuperseded`/`pruneArchived` failures do not affect status conditions
- [ ] No redundant `COS`/`COD` suffixes on method names within their respective controller packages

## Project Conventions

- [ ] Commit messages use conventional commits format (`refactor:`, `test:`, etc.)
- [ ] One logical change per commit
- [ ] No changes to public API types (`api/v1alpha1/`)
- [ ] Consistent with `specs/mission.md` design principle: "Standard controller patterns"

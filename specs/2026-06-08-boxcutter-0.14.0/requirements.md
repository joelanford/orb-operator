# Requirements

- Upgrade `pkg.package-operator.run/boxcutter` from v0.13.1 to v0.14.0 in go.mod
- Replace the removed `boxcutter.WithPreviousOwners` type with `boxcutter.WithSiblingOwners([]client.Object)`
- Rename `buildRevisionWithPreviousOwners` to `buildRevisionWithSiblings` and its `previousOwners` parameter to `siblings`
- Run `go mod tidy` to clean up go.sum

## Acceptance Criteria

- `go build ./...` passes with no compile errors
- `make verify` passes (lint, generate, build)
- `make test-unit` passes
- `make test-integration` passes
- `make test-e2e` passes — specifically all `cosr_revision_transitions.feature` scenarios that exercise predecessor handoff

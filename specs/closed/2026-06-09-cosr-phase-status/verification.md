# Verification

## Implementation Correctness

- [ ] `CompletedAt *metav1.Time` field added to `ClusterObjectSetRevisionStatus`
- [ ] `PhaseStatus` enum with `Reconciling`, `Available`, `Unknown`, `Superseded`, `TearingDown`, `TeardownComplete` values
- [ ] `ObservedPhase` type has `name`, `status`, and `incompleteObjects` fields with correct kubebuilder markers
- [ ] `ObjectStatus` type has `group`, `version`, `kind`, `namespace`, `name`, and `messages` fields with correct kubebuilder markers
- [ ] `ObservedPhases` field uses `+listType=map` and `+listMapKey=name`
- [ ] `make generate` produces updated deepcopy functions and CRD YAML
- [ ] All spec phases appear in `ObservedPhases` during active reconciliation
- [ ] Phases returned by boxcutter are `Available` or `Reconciling` based on `IsComplete()`
- [ ] Phases not returned by boxcutter are `Unknown`
- [ ] `IncompleteObjects` covers all failure modes: probe failures, collisions, creation/update errors, validation errors
- [ ] `IncompleteObjects` is empty for `Available` and `Unknown` phases
- [ ] Phases not evaluated by boxcutter are set to `Unknown`
- [ ] `ObservedPhases` is cleared for superseded COSRs
- [ ] During teardown (archival or deletion), `ObservedPhases` reports `TearingDown`/`TeardownComplete` per phase
- [ ] `ObservedPhases` is cleared once teardown is fully complete
- [ ] `updateTeardownStatus` handles both in-progress and complete teardown, and is used by both archival and deletion paths
- [ ] `CompletedAt` is set once when all phases first complete, never cleared or overwritten
- [ ] `CompletedAt` is preserved (not cleared) when a COSR is superseded or archived

## Project Conventions

- [ ] No `//nolint` comments added
- [ ] `make verify` passes (lint, generate diff, build)
- [ ] `make test-unit` passes
- [ ] `make test-integration` passes
- [ ] `make test-e2e` passes
- [ ] API type comments follow OpenShift API conventions (GoDoc on all exported types and fields)
- [ ] Field names and JSON tags follow Kubernetes API conventions (camelCase)
- [ ] ADR-0001 alignment: phase status is observational (status subresource), does not change reconciliation behavior

## Follow-up

- [ ] `theCOSRShouldNotHaveCompletedAt` e2e step does a single Get instead of polling — works because nil is the default, but should be converted to poll-based for consistency with other assertion steps

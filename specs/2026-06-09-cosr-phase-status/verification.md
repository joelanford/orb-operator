# Verification

## Implementation Correctness

- [ ] `CompletedAt *metav1.Time` field added to `ClusterObjectSetRevisionStatus`
- [ ] `PhaseStatus` enum with `Reconciling`, `Complete`, `Unknown` values
- [ ] `ObservedPhase` type has `name`, `status`, and `unavailableObjects` fields with correct kubebuilder markers
- [ ] `ObjectStatus` type has `group`, `version`, `kind`, `namespace`, `name`, and `messages` fields with correct kubebuilder markers
- [ ] `ObservedPhases` field uses `+listType=map` and `+listMapKey=name`
- [ ] `make generate` produces updated deepcopy functions and CRD YAML
- [ ] All spec phases appear in `ObservedPhases` during active reconciliation
- [ ] Phases returned by boxcutter are `Complete` or `Reconciling` based on `IsComplete()`
- [ ] Phases not returned by boxcutter are `Unknown`
- [ ] `UnavailableObjects` covers all failure modes: probe failures, collisions, creation/update errors, validation errors
- [ ] `UnavailableObjects` is empty for `Complete` and `Unknown` phases
- [ ] Phases not evaluated by boxcutter are set to `Unknown`
- [ ] `ObservedPhases` is cleared for superseded COSRs in `doReconcilePredecessor`
- [ ] `ObservedPhases` is cleared for archived COSRs in `doReconcileArchived`
- [ ] `ObservedPhases` is not set during teardown (deleted COSRs)
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

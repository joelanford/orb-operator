# Requirements

- COSR controller reconciles ClusterObjectSetRevision resources, creating/updating/deleting managed Kubernetes objects according to phases
- Phases execute sequentially: phase N+1 objects are not created until all phase N probes pass
- COSR assertions map to boxcutter probes registered as `ProgressProbeType`
- All four assertion types are supported: ConditionEqual, FieldsEqual, FieldValue, CELExpression
- Active COSRs recreate managed objects that are deleted externally
- Assertions are continuously re-evaluated; Available condition flaps as probes pass/fail
- Archiving a COSR tears down all managed objects via boxcutter
- Archived lifecycle state cannot be reverted (enforced by CRD CEL validation)
- When a newer revision in the same group becomes complete, older revisions are archived automatically
- Shared objects between revisions transfer ownership without deletion/recreation (via WithPreviousOwners)
- Objects removed in a new revision are deleted when the old revision is archived
- Default collision protection (Prevent) blocks a COSR from managing objects owned by another group
- COSR name must match `{group}-{revision}` (enforced by CRD CEL validation)
- Immutable spec fields (group, revision, phases, collisionProtection) are enforced by CRD CEL validation
- Controller is registered with the manager and started via `cmd/operator/main.go`
- Boxcutter `pkg.package-operator.run/boxcutter` is added as a dependency

## Acceptance Criteria

- All existing e2e tests pass without modification to test code
- `make verify` passes (lint, generated code check, build)
- `make test-unit` passes
- `make test-e2e` passes (all 22 scenarios across 9 feature files)
- Controller logs show reconciliation activity for COSR resources
- CRD manifests include `x-kubernetes-validations` rules for naming, immutability, and lifecycle

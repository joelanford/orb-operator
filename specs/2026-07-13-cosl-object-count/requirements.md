# Requirements

- COSL printer columns show: NAME, OBJECTS, AGE.
- COSL includes a top-level `objectCount` (int32) field.
- COSL `objectCount` is set by a MutatingAdmissionPolicy on create.
- The MutatingAdmissionPolicy is deployed alongside the operator.

## Acceptance Criteria

- `kubectl get cosl` shows OBJECTS column with correct values.
- COSL `objectCount` equals `len(objects)` after creation.
- `make verify` passes.
- `make test-unit` passes.
- `make test-e2e` passes.

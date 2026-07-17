# Requirements

- COSL printer columns show: NAME, OBJECTS, AGE.
- COSL includes a top-level `count` (int32) field.
- CEL XValidation rule enforces `count == size(objects)`.
- COSL `count` is set by a MutatingAdmissionPolicy on CREATE.
- A MutatingAdmissionPolicyBinding activates the MAP.
- Both MAP and MAPB are deployed alongside the operator via `api.libsonnet`.

## Acceptance Criteria

- `kubectl get cosl` shows OBJECTS column with correct values.
- COSL `count` equals `len(objects)` after creation.
- MAP overwrites a user-provided `count` with the correct value.
- MAP and MAPB are present in the rendered deploy manifests.
- `make verify` passes.
- `make test-unit` passes.
- `make test-e2e` passes.

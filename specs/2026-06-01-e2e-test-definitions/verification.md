# Verification

## Implementation Correctness

- [ ] `api/v1alpha1/types.go` defines ClusterObjectSetRevision with spec fields matching ADR-0001: group, revision, lifecycleState, phases
- [ ] `api/v1alpha1/groupversion_info.go` registers the scheme with group `orb.io` and version `v1alpha1`
- [ ] `zz_generated.deepcopy.go` is generated and up-to-date
- [ ] CRD manifest exists in `deploy/crds/` with group `orb.io`, version `v1alpha1`, kind `ClusterObjectSetRevision`, scope `Cluster`
- [ ] CRD has subresource status enabled
- [ ] CRD has print columns for group, revision, lifecycleState, age
- [ ] godog is a dependency in go.mod
- [ ] `test/e2e/suite_test.go` starts envtest with COSR CRD and runs godog suite
- [ ] Each scenario runs in an isolated namespace
- [ ] Feature files cover all required behaviors: object creation, phase ordering, assertion gating, four assertion types (ConditionEqual, FieldsEqual, FieldValue, CELExpression), built-in assertions, Available condition, active reconciliation, archive teardown, revision transition handoffs (superseded status, shared/removed/new objects, archival)
- [ ] All step definitions are implemented — no undefined steps, no godog.ErrPending
- [ ] All tests fail with timeout or assertion errors, not compilation or infrastructure errors

## Project Conventions

- [ ] Types use standard controller-runtime patterns (TypeMeta, ObjectMeta, Spec/Status split)
- [ ] CRD markers follow controller-gen conventions (+kubebuilder:object:root, +kubebuilder:subresource:status, +kubebuilder:resource:scope=Cluster)
- [ ] Code generation uses `go tool controller-gen` (not a standalone binary)
- [ ] Test structure follows `specs/tech-stack.md`: godog for e2e, envtest for test backend
- [ ] `make verify` passes (lint, generate diff check, build)
- [ ] No `//nolint` comments added
- [ ] Commit messages follow conventional commits format

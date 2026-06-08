# Verification

## Implementation Correctness

- [ ] `pkg.package-operator.run/boxcutter` is in go.mod as a direct dependency
- [ ] CRD YAML includes `x-kubernetes-validations` for name matching, field immutability, and lifecycle state
- [ ] `make generate` produces CRDs with validation rules and no diff
- [ ] Each assertion type (ConditionEqual, FieldsEqual, FieldValue, CELExpression) maps to the correct boxcutter probe
- [ ] Assertion probes are registered as `ProgressProbeType` to gate phase progression
- [ ] `ObjectBoundAccessManager` is created and added to the manager as a Runnable
- [ ] Controller watches COSRs and managed objects (via `accessManager.Source()` with `EnqueueWatchingObjects`)
- [ ] Controller field-indexes `spec.group` and enqueues all COSRs in the same group on changes
- [ ] Archived COSRs trigger boxcutter teardown and remove managed objects
- [ ] Latest active COSR reconciles with `WithPreviousOwners` from older revisions in the group
- [ ] Older active COSRs are set to Superseded when a newer revision exists, then Archived when the newer one completes
- [ ] Controller is registered in `cmd/operator/main.go`
- [ ] All 22 e2e scenarios pass without test code modifications

## Project Conventions

- [ ] `make lint` passes
- [ ] `make verify` passes
- [ ] `make test-unit` passes
- [ ] No `//nolint` comments added
- [ ] Import aliases match `.golangci.yml` conventions
- [ ] Controller follows standard controller-runtime patterns (Reconciler interface, SetupWithManager)
- [ ] Commit messages follow conventional commits format

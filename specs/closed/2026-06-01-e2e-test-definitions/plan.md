# Implementation Plan

1. **Define COSR API types**
   - Create `api/v1alpha1/groupversion_info.go` with `SchemeBuilder`, `GroupVersion` (`orb.operatorframework.io/v1alpha1`), and `AddToScheme`
   - Create `api/v1alpha1/types.go` with all COSR type definitions: `ClusterObjectSetRevision`, `ClusterObjectSetRevisionList`, `ClusterObjectSetRevisionSpec`, `ClusterObjectSetRevisionStatus` (conditions only), `Phase`, `PhaseObject`, `Assertion`, `ConditionEqualAssertion`, `FieldsEqualAssertion`, `FieldValueAssertion`, `CELExpressionAssertion`, `LifecycleState`
   - Add controller-gen markers: `+kubebuilder:object:root=true`, `+kubebuilder:subresource:status`, `+kubebuilder:resource:scope=Cluster`, print columns for group/revision/lifecycleState/age

2. **Generate code and CRD manifest**
   - Update `api/v1alpha1/doc.go` with `//go:generate go tool controller-gen` directive for deepcopy and CRD output
   - Run `make generate` to produce `zz_generated.deepcopy.go` and CRD YAML in `deploy/crds/`
   - Verify `make verify` passes

3. **Add godog dependency and test suite**
   - `go get github.com/cucumber/godog` to add the dependency
   - Create `test/e2e/suite_test.go`: envtest setup (start API server with COSR CRD), godog test runner, scenario initializer
   - Create `test/e2e/context.go`: per-scenario test context struct (namespace, client, COSR builder state, polling helpers)

4. **Write feature files**
   - `test/e2e/features/cosr_object_creation.feature` — inline object creation scenarios
   - `test/e2e/features/cosr_phase_progression.feature` — sequential phase execution and assertion gating
   - `test/e2e/features/cosr_assertions.feature` — ConditionEqual, FieldsEqual, FieldValue, CELExpression, and built-in assertion scenarios
   - `test/e2e/features/cosr_status.feature` — Available condition scenarios
   - `test/e2e/features/cosr_lifecycle.feature` — Active reconciliation and Archived teardown scenarios
   - `test/e2e/features/cosr_revision_transitions.feature` — multi-revision handoff, Superseded status, shared/removed/new object scenarios, archival

5. **Write step definitions**
   - `test/e2e/steps_setup.go` — "Given" steps: COSR builder (group, revision, phases, objects, assertions)
   - `test/e2e/steps_action.go` — "When" steps: create COSR, delete managed object, update lifecycleState, simulate object readiness
   - `test/e2e/steps_assert.go` — "Then" steps: poll for object existence, COSR conditions, phase completion, object absence

6. **Verify**
   - `make lint` passes
   - `make verify` passes (generate produces no diff, build succeeds)
   - `make test-e2e` runs — all scenarios fail with timeout/assertion errors, zero undefined steps

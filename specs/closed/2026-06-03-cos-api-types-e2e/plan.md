# Implementation Plan

1. **Extract `ClusterObjectSetTemplateSpec` and refactor COSR spec**
   - Create `ClusterObjectSetTemplateSpec` in `api/v1alpha1/types_clusterobjectset.go` with `CollisionProtection` and `Phases` fields
   - Refactor `ClusterObjectSetRevisionSpec` in `api/v1alpha1/types.go` to embed `ClusterObjectSetTemplateSpec` with `json:",inline"` instead of declaring `CollisionProtection` and `Phases` directly
   - Verify existing COSR controller code compiles unchanged (field access like `cosr.Spec.Phases` is the same with inline embedding)

2. **Flesh out COS API types**
   - Update `api/v1alpha1/types_clusterobjectset.go`: add `RevisionHistoryLimit`, `Template` to `ClusterObjectSetSpec`; add `Conditions` to `ClusterObjectSetStatus`
   - Add `ClusterObjectSetTemplate` and `ClusterObjectSetTemplateMetadata` types
   - Add COS-specific constants (`ReasonProgressing`)
   - Add kubebuilder markers: print columns (Available, Age), subresource status (already present)

3. **Generate code and CRD manifest**
   - Run `make generate` to produce updated deepcopy and COS CRD in `deploy/crds/`
   - Run `make verify` to confirm lint, build, and no diff

4. **Refactor test helpers for reuse**
   - Extract a `templateSpecBuilder` from `cosrBuilder` that builds `ClusterObjectSetTemplateSpec` (phases, collisionProtection). The existing phase/object/assertion builder methods move here.
   - `cosrBuilder` wraps `templateSpecBuilder` with group, revision, lifecycleState
   - Add `cosBuilder` wrapping `templateSpecBuilder` with template metadata (labels, annotations) and revisionHistoryLimit
   - Existing COSR setup steps delegate to `templateSpecBuilder` — no step definition changes, no feature file changes
   - Generalize `pollForCondition` / `pollForConditionWithReason` to work with both COS and COSR (e.g., accept a condition-extraction function or support both types)
   - Add COS tracking to `testContext` (parallel to existing `cosrs` map and `lastCreatedCOSR`)
   - Extend `teardown()` to clean up COS resources

5. **Write COS feature files**
   - `test/e2e/features/cos_stamping.feature` — initial COSR creation from COS template
   - `test/e2e/features/cos_template_metadata.feature` — label and annotation propagation
   - `test/e2e/features/cos_revision_management.feature` — revision numbering on template changes, idempotent reconciliation
   - `test/e2e/features/cos_status.feature` — COS status derived from latest COSR
   - `test/e2e/features/cos_ownership.feature` — owner references, deletion cascading
   - `test/e2e/features/cos_revision_history.feature` — revisionHistoryLimit pruning

6. **Write COS step definitions (minimal — reuse existing steps)**
   - Existing phase/object/assertion/collisionProtection setup steps work for both COS and COSR scenarios via the shared `templateSpecBuilder`
   - Add COS-specific "Given" steps: "a COS with ..." (template metadata, revisionHistoryLimit)
   - Add COS-specific "When" steps: create COS, update template spec, update template metadata, delete COS
   - Add COS-specific "Then" steps: COS condition, COSR exists with group/revision, COSR count for COS

7. **Verify**
   - `make lint` passes
   - `make verify` passes (generate produces no diff, build succeeds)
   - `make test-e2e` runs — all COS scenarios fail with timeout/assertion errors, zero undefined steps
   - Existing COSR scenarios are unaffected

# Implementation Plan

## Pass 1: ClusterObjectSet → ClusterObjectDeployment

1. **Rename Go source files**
   - `api/v1alpha1/types_clusterobjectset.go` → `types_clusterobjectdeployment.go`
   - `internal/controller/cos_controller.go` → `cod_controller.go`

2. **Rename Go identifiers in API types** (`api/v1alpha1/types_clusterobjectdeployment.go`)
   - Type names: `ClusterObjectSet` → `ClusterObjectDeployment`, `ClusterObjectSetList` → `ClusterObjectDeploymentList`, `ClusterObjectSetSpec` → `ClusterObjectDeploymentSpec`, `ClusterObjectSetStatus` → `ClusterObjectDeploymentStatus`, `ClusterObjectSetTemplate` → `ClusterObjectDeploymentTemplate`, `ClusterObjectSetTemplateMetadata` → `ClusterObjectDeploymentTemplateMetadata`, `ClusterObjectSetTemplateSpec` → `ClusterObjectDeploymentTemplateSpec`
   - `ClusterObjectSetRevisionStatusSummary` stays as-is in this pass (renamed in Pass 2)
   - Kubebuilder markers: `shortName=cos` → `shortName=cod`
   - All doc comments referencing the old names

3. **Rename Go identifiers in COSR types** (`api/v1alpha1/types_clusterobjectsetrevision.go`)
   - Update references to renamed types: `ClusterObjectSetTemplateSpec` → `ClusterObjectDeploymentTemplateSpec`
   - Update doc comments that reference "ClusterObjectSet" (the parent) to say "ClusterObjectDeployment"

4. **Update other API files**
   - `api/v1alpha1/types.go`: update any references to `ClusterObjectSet` (check for `ClusterObjectSetTemplateSpec` usage)
   - `api/v1alpha1/types_clusterobjectslice.go`: update doc comment referencing `ClusterObjectSetRevision` to say `ClusterObjectDeployment` where it refers to the parent type
   - `api/v1alpha1/groupversion_info.go`: `&ClusterObjectSet{}` → `&ClusterObjectDeployment{}`, `&ClusterObjectSetList{}` → `&ClusterObjectDeploymentList{}`

5. **Update controller code**
   - `internal/controller/cod_controller.go`: rename `COSReconciler` → `CODReconciler`, `NewCOSReconciler` → `NewCODReconciler`, all `ClusterObjectSet` type references → `ClusterObjectDeployment`
   - `internal/controller/cosr_controller.go`: update references to renamed types (e.g. `ClusterObjectSetTemplateSpec` → `ClusterObjectDeploymentTemplateSpec`)
   - `internal/controller/helpers.go`: update any COS/ClusterObjectSet references
   - `internal/controller/template_hash.go`: update `ClusterObjectSetTemplate` → `ClusterObjectDeploymentTemplate` in function signature
   - `internal/controller/template_hash_test.go`: update all `ClusterObjectSetTemplate*` type references → `ClusterObjectDeploymentTemplate*`
   - `cmd/operator/main.go`: `COSReconciler` → `CODReconciler`, `NewCOSReconciler` → `NewCODReconciler`, `cosReconciler` → `codReconciler`

6. **Update validation tests** (`api/v1alpha1/validation_test.go`)
   - Rename references to `ClusterObjectSet` types → `ClusterObjectDeployment` types

7. **Update e2e test code** (`test/e2e/`)
   - `context.go`, `steps_action.go`, `steps_assert.go`, `steps_setup.go`: rename COS-related functions, variables, and step regex patterns (e.g. `theCOSIsCreated` → `theCODIsCreated`, `aCOSNamed` → `aCODNamed`, step regex `the COS` → `the COD`)

8. **Rename and update feature files**
   - Rename `test/e2e/features/cos_*.feature` → `cod_*.feature`
   - Inside each file: replace `COS` → `COD` in feature titles, scenario names, and step text

9. **Rename and update examples**
   - `examples/sample-cos.yaml` → `sample-cod.yaml`
   - Update `kind: ClusterObjectSet` → `kind: ClusterObjectDeployment` inside the file

10. **Update apply configurations** (`applyconfigurations/`)
    - These are generated. Delete the directory contents and regenerate with `make generate`.

11. **Regenerate all generated code**
    - `make generate` (CRDs, deepcopy, apply configurations)
    - Verify CRD files are renamed: `deploy/crds/orb.operatorframework.io_clusterobjectsets.yaml` → `orb.operatorframework.io_clusterobjectdeployments.yaml`

12. **Update jsonnet deploy manifest** (`deploy/operator.jsonnet`)
    - Update CRD import path for the renamed COS CRD file
    - Rename `vapCOSNameLength` → `vapCODNameLength`, `vapCOSNameLengthBinding` → `vapCODNameLengthBinding`
    - Update `resources: ['clusterobjectsets']` → `resources: ['clusterobjectdeployments']` in the COD VAP

13. **Update project docs and specs**
    - `specs/mission.md`: replace COS references with COD, update type names
    - `specs/tech-stack.md`: update project structure section and type references
    - `specs/conventions.md`: update example commit messages and PR titles that reference COS/COSR
    - `ADR.md`: update type name references (if it contains COS/COSR names; leave ADR-0001 design language intact)
    - Open specs (`specs/2026-06-03-*`, `specs/2026-06-04-*`, `specs/2026-06-23-*`, `specs/2026-06-28-*`): update COS/ClusterObjectSet references to COD/ClusterObjectDeployment

14. **Verify Pass 1**
    - `make verify` (lint, build, generate diff check, goreleaser check)
    - `make test-unit`
    - `make test-e2e`

15. **Commit**: `refactor: rename ClusterObjectSet to ClusterObjectDeployment`

## Pass 2: ClusterObjectSetRevision → ClusterObjectSet

16. **Rename Go source files**
    - `api/v1alpha1/types_clusterobjectsetrevision.go` → `types_clusterobjectset.go`
    - `api/v1alpha1/validation_cosr_test.go` → `validation_cos_test.go`
    - `internal/controller/cosr_controller.go` → `cos_controller.go`

17. **Rename Go identifiers in API types** (`api/v1alpha1/types_clusterobjectset.go`, formerly the revision file)
    - `ClusterObjectSetRevision` → `ClusterObjectSet`, `ClusterObjectSetRevisionList` → `ClusterObjectSetList`, `ClusterObjectSetRevisionSpec` → `ClusterObjectSetSpec`, `ClusterObjectSetRevisionStatus` → `ClusterObjectSetStatus`
    - Kubebuilder markers: `shortName=cosr` → `shortName=cos`
    - All doc comments referencing "revision" where it referred to the type name

18. **Update other API files**
    - `api/v1alpha1/types_clusterobjectdeployment.go`: `ClusterObjectSetRevisionStatusSummary` → `ClusterObjectSetStatusSummary`, references to `ClusterObjectSetRevision` → `ClusterObjectSet` in doc comments
    - `api/v1alpha1/types_clusterobjectslice.go`: update remaining `ClusterObjectSetRevision` references in doc comments
    - `api/v1alpha1/groupversion_info.go`: `&ClusterObjectSetRevision{}` → `&ClusterObjectSet{}`, `&ClusterObjectSetRevisionList{}` → `&ClusterObjectSetList{}`

19. **Update controller code**
    - `internal/controller/cos_controller.go` (formerly cosr): `COSRReconciler` → `COSReconciler`, `NewCOSRReconciler` → `NewCOSReconciler`, all `ClusterObjectSetRevision` references → `ClusterObjectSet`
    - `internal/controller/cod_controller.go`: update references to `ClusterObjectSetRevision` → `ClusterObjectSet`
    - `internal/controller/helpers.go`: update COSR references
    - `cmd/operator/main.go`: `COSRReconciler` → `COSReconciler`, `NewCOSRReconciler` → `NewCOSReconciler`, `cosrReconciler` → `cosReconciler`

20. **Update validation tests** (`api/v1alpha1/validation_cos_test.go`, formerly cosr)
    - Rename `ClusterObjectSetRevision` references → `ClusterObjectSet`

21. **Update e2e test code** (`test/e2e/`)
    - `context.go`, `steps_action.go`, `steps_assert.go`, `steps_setup.go`: rename COSR-related functions, variables, and step regex patterns → COS equivalents (e.g. `theCOSRIsCreated` → `theCOSIsCreated`, `aCOSRWithGroupAndRevision` → `aCOSWithGroupAndRevision`)

22. **Rename and update feature files**
    - Rename `test/e2e/features/cosr_*.feature` → `cos_*.feature`
    - Inside each file: replace `COSR` → `COS` in feature titles, scenario names, and step text
    - Also update any remaining `ClusterObjectSetRevision` references in `cod_*.feature` files

23. **Rename and update examples**
    - `examples/sample-cosr.yaml` → `sample-cos.yaml`
    - Update `kind: ClusterObjectSetRevision` → `kind: ClusterObjectSet` inside the file

24. **Regenerate all generated code**
    - `make generate`
    - Verify CRD file: `deploy/crds/orb.operatorframework.io_clusterobjectsetrevisions.yaml` → `orb.operatorframework.io_clusterobjectsets.yaml`

25. **Update jsonnet deploy manifest** (`deploy/operator.jsonnet`)
    - Update CRD import path for the renamed COSR CRD file
    - Rename `vapCOSRName` → `vapCOSName`, `vapCOSRNameBinding` → `vapCOSNameBinding`, `vapCOSROrphanFinalizer` → `vapCOSOrphanFinalizer`, `vapCOSROrphanFinalizerBinding` → `vapCOSOrphanFinalizerBinding`
    - Update `resources: ['clusterobjectsetrevisions']` → `resources: ['clusterobjectsets']`

26. **Update project docs and specs**
    - `specs/mission.md`: replace remaining COSR references with COS
    - `specs/tech-stack.md`: update remaining COSR references
    - `specs/conventions.md`: update remaining COSR references in examples
    - `ADR.md`: update remaining COSR/ClusterObjectSetRevision references
    - Open specs: update COSR/ClusterObjectSetRevision references to COS/ClusterObjectSet

27. **Verify Pass 2**
    - `make verify`
    - `make test-unit`
    - `make test-e2e`

28. **Commit**: `refactor: rename ClusterObjectSetRevision to ClusterObjectSet`

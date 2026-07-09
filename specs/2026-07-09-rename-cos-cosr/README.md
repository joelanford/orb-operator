---
status: in-progress
---
# Rename COS to ClusterObjectDeployment, COSR to ClusterObjectSet

## Summary

Rename the two primary API types to better reflect their roles:

| Current Name               | New Name                  | Short Name |
|----------------------------|---------------------------|------------|
| ClusterObjectSet           | ClusterObjectDeployment   | cod        |
| ClusterObjectSetRevision   | ClusterObjectSet          | cos        |

ClusterObjectSlice is unchanged.

The current names are inherited from ADR-0001 and package-operator. "ClusterObjectSet" is the parent that templates revisions, and "ClusterObjectSetRevision" is the immutable snapshot that actually manages objects. The new names make the hierarchy clearer: a "deployment" declares intent and stamps out "sets" (the immutable snapshots).

## Design

### Type mapping

Every Go type, file, variable prefix, kubebuilder marker, feature file, example, and documentation reference is renamed according to this mapping:

**Pass 1 — ClusterObjectSet → ClusterObjectDeployment** (frees the "ClusterObjectSet" name):

| Current                                  | New                                          |
|------------------------------------------|----------------------------------------------|
| `ClusterObjectSet`                       | `ClusterObjectDeployment`                    |
| `ClusterObjectSetList`                   | `ClusterObjectDeploymentList`                |
| `ClusterObjectSetSpec`                   | `ClusterObjectDeploymentSpec`                |
| `ClusterObjectSetStatus`                 | `ClusterObjectDeploymentStatus`              |
| `ClusterObjectSetTemplate`               | `ClusterObjectDeploymentTemplate`            |
| `ClusterObjectSetTemplateMetadata`       | `ClusterObjectDeploymentTemplateMetadata`    |
| `ClusterObjectSetTemplateSpec`           | `ClusterObjectDeploymentTemplateSpec`        |
| shortName `cos`                          | shortName `cod`                              |

After Pass 1 the codebase has COD and COSR.

**Pass 2 — ClusterObjectSetRevision → ClusterObjectSet** (uses the now-free name):

| Current                                  | New                                          |
|------------------------------------------|----------------------------------------------|
| `ClusterObjectSetRevision`               | `ClusterObjectSet`                           |
| `ClusterObjectSetRevisionList`           | `ClusterObjectSetList`                       |
| `ClusterObjectSetRevisionSpec`           | `ClusterObjectSetSpec`                       |
| `ClusterObjectSetRevisionStatus`         | `ClusterObjectSetStatus`                     |
| `ClusterObjectSetRevisionStatusSummary`  | `ClusterObjectSetStatusSummary`              |
| shortName `cosr`                         | shortName `cos`                              |

After Pass 2 the codebase has COD and COS.

### File renames

| Current                                          | New                                                |
|--------------------------------------------------|----------------------------------------------------|
| `api/v1alpha1/types_clusterobjectset.go`         | `api/v1alpha1/types_clusterobjectdeployment.go`    |
| `api/v1alpha1/types_clusterobjectsetrevision.go` | `api/v1alpha1/types_clusterobjectset.go`           |
| `api/v1alpha1/validation_cosr_test.go`           | `api/v1alpha1/validation_cos_test.go`              |
| `internal/controller/cos_controller.go`          | `internal/controller/cod_controller.go`            |
| `internal/controller/cosr_controller.go`         | `internal/controller/cos_controller.go`            |
| `examples/sample-cos.yaml`                       | `examples/sample-cod.yaml`                         |
| `examples/sample-cosr.yaml`                      | `examples/sample-cos.yaml`                         |
| `test/e2e/features/cos_*.feature`                | `test/e2e/features/cod_*.feature`                  |
| `test/e2e/features/cosr_*.feature`               | `test/e2e/features/cos_*.feature`                  |

Apply configurations under `applyconfigurations/` are regenerated, not manually renamed.

### Abbreviation mapping

In all Go code, feature files, and documentation:

| Current | New | Meaning |
|---------|-----|---------|
| COS     | COD | ClusterObjectDeployment |
| COSR    | COS | ClusterObjectSet |
| cos     | cod | lowercase prefix (file names, variables) |
| cosr    | cos | lowercase prefix (file names, variables) |

### Scope

The rename touches:
- API type definitions and generated code (deepcopy, CRDs, apply configurations)
- Controllers and helpers
- Integration and e2e tests (Go code and `.feature` files)
- Examples
- Project context docs (`specs/mission.md`, `specs/tech-stack.md`)
- Open specs that reference COS/COSR

Closed specs under `specs/closed/` are historical and are **not** updated. A new spec covers the changes to open specs.

### Two-pass ordering

Each pass produces a compilable, testable codebase. Pass 1 is committed before Pass 2 begins. This avoids name collisions and makes the diff reviewable.

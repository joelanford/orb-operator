---
status: done
---
# COS API Types & E2E Test Definitions

## Summary

Define the ClusterObjectSet (COS) v1alpha1 API types and write godog BDD e2e tests that define COS controller behavior. API types flesh out the existing COS stub with template-based COSR stamping fields, `revisionHistoryLimit`, and status conditions. E2E tests compile and run against a kind cluster with the full operator deployed but fail because no COS controller exists yet. This is the test-first definition of COS behavior.

Scope: COS types and e2e tests only. No COS controller implementation. No ClusterObjectSlice changes. No `progressDeadlineMinutes` (deferred to a separate work item).

Prerequisite: `2026-06-03-cosr-revision-uint32` (change Revision from int32 to uint32 with min=1) should land first.

## Design

### COS API Type

The ClusterObjectSet is a cluster-scoped resource following the ADR-0001 design. It acts like a Deployment: it holds a template for COSRs and stamps out new revisions when the template changes.

Group/version: `orb.operatorframework.io/v1alpha1` (same as COSR, already registered)

Type hierarchy:

```
ClusterObjectSet
├── spec
│   ├── revisionHistoryLimit: *int32          # max archived COSRs to retain (default: 5)
│   └── template
│       ├── metadata                          # propagated to stamped COSR ObjectMeta
│       │   ├── labels: map[string]string
│       │   └── annotations: map[string]string
│       └── spec: ClusterObjectSetTemplateSpec  # user-settable COSR fields only
│           ├── collisionProtection: *CollisionProtection
│           └── phases: []Phase
└── status
    └── conditions: []Condition               # derived from latest COSR's Available condition
```

Key design decisions:

- **`template.spec` is a separate type from `ClusterObjectSetRevisionSpec`** — the COSR spec has required fields (`group`, `revision`) and a mutable field (`lifecycleState`) that don't belong in the template. The COS controller populates `group` (from COS name), `revision` (auto-incremented), and `lifecycleState` (always `Active`) when stamping. The template spec contains only user-settable fields: `phases` and `collisionProtection`.

- **COS template fields are mutable; COSR fields are immutable** — on the COS, `template.spec.phases` and `template.spec.collisionProtection` can be updated freely (each update triggers a new revision). On the COSR, these same fields are locked by CEL validation rules on `ClusterObjectSetRevisionSpec`. The immutability rules stay on the COSR spec type, not on the shared `ClusterObjectSetTemplateSpec`, so the COS template is naturally mutable.

- **Any template change (spec or metadata) creates a new revision** — this is simpler than trying to update COSRs in-place and consistent with the "revision = immutable snapshot" model.

- **`revisionHistoryLimit` defaults to 5** — controls how many Archived COSRs to retain. The active COSR is never counted toward this limit. Lowest-revision Archived COSRs are deleted first when the limit is exceeded.

- **Group is derived from COS name** — the COS controller sets `group` to the COS name on stamped COSRs. This ensures all COSRs from one COS form a single revision chain.

- **COSR naming and length constraints** — stamped COSRs are named `{group}-{revision}`. Since COSR names may appear in label values (max 63 chars), and revision is an int32 (max 10 digits), the group field (and COS name) is capped at 52 characters: 52 (group) + 1 (`-`) + 10 (revision) = 63. The existing COSR group MaxLength validation is updated from 253 to 52.

- **Status mirrors COSR state** — when a single Active COSR exists, the COS Available condition mirrors it directly (`True` or `False`). When multiple Active COSRs exist (rollout in progress), the COS shows `Available=Unknown, Reason=Progressing` since the old revision is still serving but the new one isn't confirmed ready.

- **Owner references** — the COS controller sets an owner reference on each stamped COSR pointing back to the COS. Deleting the COS cascades deletion to all owned COSRs via Kubernetes garbage collection.

### Supporting Types

```go
type ClusterObjectSetTemplate struct {
    Metadata ClusterObjectSetTemplateMetadata `json:"metadata,omitempty"`
    Spec     ClusterObjectSetTemplateSpec     `json:"spec"`
}

type ClusterObjectSetTemplateMetadata struct {
    Labels      map[string]string `json:"labels,omitempty"`
    Annotations map[string]string `json:"annotations,omitempty"`
}

type ClusterObjectSetTemplateSpec struct {
    CollisionProtection *CollisionProtection `json:"collisionProtection,omitempty"`
    Phases              []Phase              `json:"phases"`
}
```

**`ClusterObjectSetRevisionSpec` embeds `ClusterObjectSetTemplateSpec` inline** — the COSR spec gains its `phases` and `collisionProtection` fields via `json:",inline"` embedding rather than defining them directly. This means:

- The COSR CRD serialization is **unchanged** (`spec.phases`, `spec.collisionProtection` stay flat)
- The COS template and COSR spec share the same Go type for the templatable fields
- The COS controller copies `template.spec` directly into the stamped COSR's embedded struct

```go
type ClusterObjectSetRevisionSpec struct {
    Group                        string         `json:"group"`
    Revision                     int32          `json:"revision"`
    LifecycleState               LifecycleState `json:"lifecycleState,omitempty"`
    ClusterObjectSetTemplateSpec `json:",inline"`
}
```

### Status Conditions

Available condition variants:
- `Available=True, Reason=Available` — single Active COSR exists and is Available
- `Available=False, Reason=Unavailable` — single Active COSR exists and is not Available, or no COSR exists
- `Available=Unknown, Reason=Progressing` — multiple Active COSRs exist (rollout in progress, ownership handoff underway)

### CRD Print Columns

| Column | JSONPath | Type |
|--------|----------|------|
| Availability | `.status.conditions[?(@.type=="Available")].reason` | string |
| Age | `.metadata.creationTimestamp` | date |

### E2E Test Architecture

Tests extend the existing godog suite in `test/e2e/`. The test infrastructure runs against a real kind cluster with the full operator deployed via `make run` (goreleaser build, kind cluster creation, image load, jsonnet-rendered manifests applied). Tests use `ctrl.GetConfig()` to get a kubeconfig for the live cluster. This work item adds COS-specific feature files, step definitions, and builder helpers to the existing suite.

All COS tests are expected to fail (timeout waiting for controller actions) since no COS controller exists yet. The operator is deployed but has no COS reconciler registered.

### Feature File Organization

- `cos_stamping.feature` — COS creates a COSR from its template
- `cos_template_metadata.feature` — template.metadata propagation to COSRs
- `cos_revision_management.feature` — new revisions on template changes, revision numbering
- `cos_status.feature` — COS status derived from latest COSR
- `cos_ownership.feature` — owner references, deletion cascading
- `cos_revision_history.feature` — revisionHistoryLimit pruning of old archived COSRs

### Step Definition Patterns — DRY with Existing COSR Steps

COS tests should maximize reuse of existing step definitions and helpers. The key refactoring:

**Refactor `cosrBuilder` around `ClusterObjectSetTemplateSpec`:** The existing builder already manages `phases` and `collisionProtection` — exactly the fields in the shared `ClusterObjectSetTemplateSpec`. Extract a `templateSpecBuilder` that builds `ClusterObjectSetTemplateSpec`, then:
- COSR builder wraps it with `group`, `revision`, `lifecycleState`
- COS builder wraps it with `template.metadata`, `revisionHistoryLimit`

This means all existing phase/object/assertion setup steps (`a phase "X" with a ConfigMap "Y"`, `the last object has assertion ...`, `the COSR collisionProtection is "..."`, etc.) work for both COS and COSR scenarios with no duplication.

**Generalize condition polling:** `pollForCondition` and `pollForConditionWithReason` are currently hardcoded to `ClusterObjectSetRevision`. Refactor to accept any `client.Object` that has conditions (e.g., via a condition-extraction function or by supporting both COS and COSR types).

**Directly reusable (no changes needed):**
- `pollForObject` / `pollForObjectAbsence`
- Object factories: `newConfigMap`, `newConfigMapWithData`, `newCRD`, `newCR`
- All ConfigMap/CRD assertion and action steps
- UID tracking steps
- Namespace setup/teardown, utilities

**New COS-specific steps (minimal):**
1. **Setup** — "Given a COS with ..." (template metadata, revisionHistoryLimit)
2. **Action** — "When the COS is created", "When the COS template spec is updated", "When the COS template metadata is updated", "When the COS is deleted"
3. **Assert** — "Then the COS should have condition ...", "Then a COSR should exist with group X and revision N", "Then the COSR count for the COS should be N"

# API Reference

API group: `orb.operatorframework.io/v1alpha1`

---

## ClusterObjectDeployment

**Scope:** Cluster
**Short name:** `cod`

Declares a set of Kubernetes objects to apply and manage through immutable revisions.

### Spec

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `revisionHistoryLimit` | `*int32` | No | `10` | Max archived COS resources to retain. `0` disables history. |
| `progressDeadlineMinutes` | `*int32` | No | _(none)_ | Minutes to wait for rollout progress before reporting `ProgressDeadlineExceeded`. Min: `1`. |
| `template` | `ClusterObjectDeploymentTemplate` | Yes | | Template stamped into each COS revision. |

#### ClusterObjectDeploymentTemplate

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `metadata.labels` | `map[string]string` | No | Labels propagated to COS resources. Max 32 entries. |
| `metadata.annotations` | `map[string]string` | No | Annotations propagated to COS resources. Max 32 entries, values max 256 KiB. |
| `spec.collisionProtection` | `CollisionProtection` | No | Default collision protection. One of `Prevent` (default), `IfNoController`, `None`. |
| `spec.phases` | `[]Phase` | Yes | Ordered list of phases. 1–20 items. |

### Status

| Field | Type | Description |
|-------|------|-------------|
| `conditions` | `[]Condition` | `Available` and `Progressing` conditions. |
| `activeRevisions` | `[]ClusterObjectSetStatusSummary` | Non-archived COS resources with their conditions. |

### Print Columns

| Column | Source |
|--------|--------|
| Availability | `.status.conditions[?(@.type=="Available")].reason` |
| Progressing | `.status.conditions[?(@.type=="Progressing")].reason` |
| Age | `.metadata.creationTimestamp` |

---

## ClusterObjectSet

**Scope:** Cluster
**Short name:** `cos`

An immutable revision snapshot. All spec fields except `lifecycleState` are immutable after creation.

### Naming

Name **must** match `{group}-{revision}` (enforced by ValidatingAdmissionPolicy).

### Spec

| Field | Type | Required | Immutable | Description |
|-------|------|----------|-----------|-------------|
| `group` | `string` | Yes | Yes | Links related revisions. 1–52 chars, DNS-1035 label. |
| `revision` | `uint32` | Yes | Yes | Monotonically increasing sequence number. Min: `1`. |
| `lifecycleState` | `LifecycleState` | Yes | One-way | `Active` or `Archived`. Cannot revert from `Archived`. |
| `collisionProtection` | `*CollisionProtection` | No | Yes | Default collision protection mode. |
| `phases` | `[]Phase` | Yes | Yes | Ordered list of phases. 1–20 items. |

### Status

| Field | Type | Description |
|-------|------|-------------|
| `conditions` | `[]Condition` | `Available` condition. |
| `resolvedContentHash` | `string` | SHA-256 of resolved content. Immutable once set. |
| `completedAt` | `*Time` | When all phases first completed. Immutable once set. |
| `observedPhases` | `[]ObservedPhase` | Per-phase status. Max 20 items. |

#### ObservedPhase

| Field | Type | Description |
|-------|------|-------------|
| `name` | `string` | Phase name from spec. |
| `status` | `PhaseStatus` | One of: `Invalid`, `Reconciling`, `Available`, `Unknown`, `Superseded`, `TearingDown`, `TeardownComplete`. |
| `completedAt` | `*Time` | When this phase first became Available. Immutable once set. |
| `error` | `string` | Phase-level error message. Max 1024 chars. |
| `incompleteObjects` | `[]ObjectStatus` | Objects not yet complete. Max 50 items. |

#### ObjectStatus

| Field | Type | Description |
|-------|------|-------------|
| `group` | `string` | API group (empty for core). |
| `version` | `string` | API version. |
| `kind` | `string` | Kind. |
| `namespace` | `string` | Namespace (empty for cluster-scoped). |
| `name` | `string` | Object name. |
| `messages` | `[]string` | Failure reasons. Max 17 items, each max 1024 chars. |

### Print Columns

| Column | Source |
|--------|--------|
| Group | `.spec.group` |
| Revision | `.spec.revision` |
| Available | `.status.conditions[?(@.type=="Available")].status` |
| Lifecycle | `.spec.lifecycleState` |
| Age | `.metadata.creationTimestamp` |

---

## ClusterObjectSlice

**Scope:** Cluster
**Short name:** `cosl`

A pure content store. No spec or status. The `objects` field is immutable after creation.

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `objects` | `[]SliceObject` | Yes | Object manifests. 1–256 items. |

#### SliceObject

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `apiVersion` | `string` | Yes | API version (e.g., `v1`, `apps/v1`). |
| `kind` | `string` | Yes | Kind (e.g., `ConfigMap`). |
| `name` | `string` | Yes | Object name. DNS-1123 subdomain. |
| `namespace` | `string` | No | Namespace. Empty for cluster-scoped. |
| `content` | `[]byte` | Yes | Raw JSON or gzip-compressed JSON. Auto-detected by magic number. Base64-encoded on wire. |

---

## Shared Types

### Phase

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | `string` | Yes | DNS-1035 label, 1–63 chars. |
| `collisionProtection` | `*CollisionProtection` | No | Overrides spec-level setting for this phase. |
| `objects` | `[]PhaseObject` | Yes | Objects in this phase. 1–50 items. |

### PhaseObject

Exactly one of `object` or `objectRef` must be set.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `object` | `RawExtension` | No | Inline Kubernetes manifest. |
| `objectRef` | `*ObjectRef` | No | Reference to an object in a ClusterObjectSlice. |
| `collisionProtection` | `*CollisionProtection` | No | Overrides phase-level setting for this object. |
| `assertions` | `[]Assertion` | No | Availability checks. Max 16 items. |

### ObjectRef

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `sliceName` | `string` | Yes | Name of the ClusterObjectSlice. DNS-1123 subdomain. |
| `apiVersion` | `string` | Yes | API version of the referenced object. |
| `kind` | `string` | Yes | Kind of the referenced object. |
| `name` | `string` | Yes | Name of the referenced object. |
| `namespace` | `string` | No | Namespace of the referenced object. |

### Assertion

Exactly one field must be set.

| Field | Type | Description |
|-------|------|-------------|
| `conditionEqual` | `*ConditionEqualAssertion` | Check a status condition type/status. |
| `fieldsEqual` | `*FieldsEqualAssertion` | Compare two field values. |
| `fieldValue` | `*FieldValueAssertion` | Check a field against an expected value. |
| `celExpression` | `*CELExpressionAssertion` | Evaluate a CEL expression (object is `self`). |

### CollisionProtection

| Value | Description |
|-------|-------------|
| `Prevent` | Only manage objects created by this revision. **(Default)** |
| `IfNoController` | Adopt objects without a controller owner. |
| `None` | Adopt objects unconditionally. |

### LifecycleState

| Value | Description |
|-------|-------------|
| `Active` | Revision reconciles and reports availability. |
| `Archived` | Triggers reverse-order teardown. One-way transition. |

---

## Conditions

### ClusterObjectDeployment Conditions

| Type | Reasons |
|------|---------|
| `Available` | `Available`, `Unavailable`, `Progressing`, `InternalError`, `ReconcileError`, `TeardownError`, `InvalidRevision` |
| `Progressing` | `NewClusterObjectSetProgressing`, `NewClusterObjectSetProgressed`, `ProgressDeadlineExceeded`, `NoActiveRevisions` |

### ClusterObjectSet Conditions

| Type | Reasons |
|------|---------|
| `Available` | `Available`, `Unavailable`, `Progressing`, `Archived`, `Superseded`, `InternalError`, `ReconcileError`, `TeardownError`, `InvalidRevision` |

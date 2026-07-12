# ClusterObjectSet

A **ClusterObjectSet** (short name: `cos`) is an immutable revision snapshot. It contains the exact set of objects and phases that were applied to the cluster at a specific point in time.

## Analogy

A ClusterObjectSet is to a ClusterObjectDeployment what a ReplicaSet is to a Deployment. You typically don't create COS resources directly — the COD controller does it for you. However, you can create them manually for full control (see [Manual Revisions](../guides/manual-revisions.md)).

## Spec

All spec fields except `lifecycleState` are **immutable after creation**.

### group

A label-safe identifier that links related revisions together. All COS resources sharing the same `group` form an ordered sequence. The value must be lowercase alphanumeric or `-`, starting with a letter, at most 52 characters.

### revision

A monotonically increasing sequence number within the group, starting at 1.

### lifecycleState

Controls whether this revision is actively reconciling:

- **`Active`** — The revision reconciles its managed objects and reports availability
- **`Archived`** — Triggers reverse-order teardown of objects still owned by this revision. This transition is one-way — an archived revision cannot be unarchived.

### phases

The ordered list of phases and objects, identical in structure to the COD template. See [Phases & Assertions](phases-and-assertions.md).

### collisionProtection

The default collision protection mode. See [Collision Protection](collision-protection.md).

## Naming convention

COS names **must** follow the pattern `{group}-{revision}`. This is enforced by a ValidatingAdmissionPolicy at creation time. For example, a COS with `group: my-app` and `revision: 3` must be named `my-app-3`.

## Status

### Conditions

| Condition | Meaning |
|-----------|---------|
| `Available` | Whether all managed objects in this revision satisfy their assertions |

### resolvedContentHash

A SHA-256 hash of all resolved object content (inline objects and objectRef-resolved objects, in phase order). Set once on the first successful resolution and never changed. Detects content substitution — for example, if a referenced ClusterObjectSlice is deleted and recreated with different content.

### completedAt

Timestamp when all phases first completed successfully. Set once and never cleared. When `completedAt` is set but `Available` is `False`, the revision has **regressed** after a successful rollout (likely due to external drift).

### observedPhases

Per-phase status reporting. Each phase includes:

| Field | Description |
|-------|-------------|
| `name` | Phase name from the spec |
| `status` | One of: `Invalid`, `Reconciling`, `Available`, `Unknown`, `Superseded`, `TearingDown`, `TeardownComplete` |
| `completedAt` | When this phase first became Available (immutable once set) |
| `error` | Phase-level error message (validation or configuration problems) |
| `incompleteObjects` | Objects that are not yet complete, with their failure messages |

## Example

```bash
$ kubectl get cos
NAME       GROUP    REVISION   AVAILABLE   LIFECYCLE   AGE
my-app-1   my-app   1          True        Archived    10m
my-app-2   my-app   2          True        Active       5m
```

## Orphan finalizer

When you delete a COS that has the `orphan` finalizer, the managed objects are preserved on the cluster — the operator removes its owner references but does not delete the objects. This is useful when you want to hand off management of objects to another controller or to manual management.

!!! note
    The `orphan` finalizer cannot be removed while the `orb.operatorframework.io/cos-finalizer` is still present. This ordering is enforced by a ValidatingAdmissionPolicy.

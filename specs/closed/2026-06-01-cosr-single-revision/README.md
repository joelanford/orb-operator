---
status: done
---
# COSR Controller — Single Revision Lifecycle

## Summary

Implement the ClusterObjectSetRevision controller backed by boxcutter's RevisionEngine. After this work item, all existing e2e tests pass: object creation across phases, assertion-gated phase progression, continuous status evaluation, active/archived lifecycle, multi-revision handoffs, collision protection, naming enforcement, and spec immutability.

## Design

### Architecture

The controller is group-aware: reconciling any COSR triggers reconciliation logic for all COSRs in the same group. This enables the controller to coordinate revision transitions (supersede, ownership transfer, archival) without external coordination.

```
COSR Reconcile(cosr)
├── List all COSRs with same spec.group
├── Sort by spec.revision (ascending)
├── Determine role: latest active, older active, or archived
│
├── If archived:
│   └── Teardown via boxcutter → set Available=False/Archived
│
├── If older active and latest is complete:
│   └── Set lifecycleState=Archived (triggers re-reconcile → teardown)
│
├── If older active and latest is NOT complete:
│   └── Set Available=False/Superseded
│
└── If latest active:
    ├── Build boxcutter Revision from phases
    ├── Map COSR assertions → boxcutter probes (ProgressProbeType)
    ├── WithPreviousOwners from older revisions
    ├── WithCollisionProtection from spec
    ├── Reconcile via RevisionEngine
    └── Map RevisionResult → status conditions
        ├── IsComplete() → Available=True
        └── !IsComplete() → Available=False/Unavailable
```

### Dynamic cache via managedcache

The controller uses boxcutter's `managedcache.ObjectBoundAccessManager` to create per-COSR caches. Each COSR gets its own `Accessor` (cached reader + writer) scoped to the GVKs it manages. Benefits:

- Informers are automatically created for managed object types and stopped when no longer needed
- The `EnqueueWatchingObjects` handler provides reverse-lookup watches — when a managed object changes, the owning COSR is re-enqueued
- Cache lifecycle is tied to COSR lifecycle — caches are freed on COSR deletion

The `ObjectBoundAccessManager` is added to the controller-runtime manager as a `Runnable` and its `Source()` is used for controller watches.

The `Accessor` from the managed cache is passed to boxcutter's `RevisionEngineOptions` as the `Reader` and `Writer`, so boxcutter reads/writes through the per-COSR scoped cache.

### Boxcutter integration

The COSR controller creates a `boxcutter.RevisionEngine` (via `boxcutter.NewRevisionEngine`) at setup time. Each reconcile:

1. Converts COSR phases → boxcutter `Phase` objects, with each `PhaseObject.Object` as an unstructured Kubernetes object
2. Converts COSR assertions → boxcutter probes registered as `ProgressProbeType` to gate phase progression:
   - `ConditionEqual` → `probing.ConditionProbe`
   - `FieldsEqual` → `probing.FieldsEqualProbe`
   - `FieldValue` → `probing.FieldValueProbe`
   - `CELExpression` → `probing.CELProbe`
3. Builds a `boxcutter.Revision` with `boxcutter.NewRevisionWithOwner`
4. Calls `engine.Reconcile(ctx, revision, opts...)` or `engine.Teardown(ctx, revision, opts...)` depending on lifecycle state
5. Maps the `RevisionResult` to COSR status conditions

### CRD CEL validation rules

Validation is enforced at the CRD level via `x-kubernetes-validations` CEL rules (no webhook needed):

| Rule | CEL expression |
|---|---|
| Name must match `{group}-{revision}` | `self.metadata.name == self.spec.group + '-' + string(self.spec.revision)` |
| `group` is immutable | `self.group == oldSelf.group` (on spec) |
| `revision` is immutable | `self.revision == oldSelf.revision` (on spec) |
| `phases` is immutable | `self.phases == oldSelf.phases` (on spec) |
| `collisionProtection` is immutable | `self.collisionProtection == oldSelf.collisionProtection` (on spec) |
| `lifecycleState` one-way Active→Archived | `oldSelf.lifecycleState != 'Archived' \|\| self.lifecycleState == 'Archived'` (on spec) |

Duplicate group+revision is prevented naturally by the naming rule: two COSRs with the same group and revision would have the same name, which the API server rejects.

### Status conditions

The controller sets a single `Available` condition:

| Status | Reason | When |
|---|---|---|
| `True` | — | All phases complete, all probes pass |
| `False` | `Unavailable` | Phases incomplete, probes failing, or objects missing |
| `False` | `Superseded` | A newer revision exists in the same group |
| `False` | `Archived` | lifecycleState is Archived |

### Controller watches

- Primary: ClusterObjectSetRevision
- Secondary: managed objects via `ObjectBoundAccessManager.Source()` with `EnqueueWatchingObjects` handler (re-enqueues owning COSR when managed objects change, enabling assertion re-evaluation and object recreation)
- Group-based fan-out: when any COSR changes, enqueue all COSRs in the same group (via field index on `spec.group`)

### Collision protection

Default collision protection is `Prevent` (from boxcutter). When a COSR tries to manage an object already owned by another COSR in a different group, boxcutter returns an `ObjectResultCollision` and the object is not adopted. The COSR stays `Available=False`.

Within a group, `WithPreviousOwners` allows the newer revision to take ownership from the older one — this bypasses collision detection for the expected handoff case.

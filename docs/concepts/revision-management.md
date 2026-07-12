# Revision Management

Revisions are the core mechanism for safe upgrades. Each revision is an immutable snapshot of the desired state, and the operator coordinates transitions between revisions to prevent gaps or conflicts.

## Revision lifecycle

```
COD template updated
  └── New COS created (Active)
        └── Phases applied sequentially
              └── All assertions pass → Available
                    └── Previous COS archived
                          └── Archived COS torn down (reverse order)
                                └── Eventually garbage collected
```

### 1. Creation

When you create or update a ClusterObjectDeployment, the controller creates a new ClusterObjectSet with an incremented revision number. The COS is named `{cod-name}-{revision}`.

### 2. Rollout

The COS controller applies objects phase by phase. Each phase must become available (all assertions pass) before the next phase begins.

### 3. Transition

During a revision transition, **both the old and new revision are active simultaneously**. The new (higher-numbered) revision takes ownership of objects that exist in both revisions. Objects unique to the old revision remain under the old revision's ownership.

This means:

- Shared objects (e.g., a Namespace that exists in both revisions) are transferred to the new revision without being deleted and recreated
- Objects removed in the new revision stay under the old revision until it's archived
- There is no gap in reconciliation — at least one revision is always managing each object

### 4. Archival

Once the new revision becomes fully available, the COD controller archives the old revision by setting `lifecycleState: Archived`. This triggers reverse-order teardown of objects still owned by the archived revision (objects that were not carried forward to the new revision).

### 5. Garbage collection

The `revisionHistoryLimit` setting controls how many archived revisions are retained. Older archived revisions beyond this limit are garbage collected. The default limit is 10.

## Revision chain

All COS resources sharing the same `group` form a **revision chain**. The COS controller coordinates ownership handoffs within the group based on revision numbers. Only the highest-numbered active revision reconciles shared objects.

```
my-app-1 (revision 1, Archived)
my-app-2 (revision 2, Archived)
my-app-3 (revision 3, Active)     ← manages all shared objects
```

## Template hashing

The COD controller computes a hash of the template content and stores it in a label on each COS:

```
orb.operatorframework.io/template-hash: 569a9089
```

If the template hasn't changed (same hash), no new revision is created. This prevents unnecessary revision churn when a COD is reconciled without changes.

## Progress deadline

When `progressDeadlineMinutes` is set, the controller watches for forward progress during a rollout. Progress is defined as any phase completing successfully. If no phase completes within the deadline, the COD reports:

```
Progressing: False (ProgressDeadlineExceeded)
```

This helps detect stuck rollouts — for example, when an assertion can never pass because of a misconfigured object.

## Drift detection

After initial rollout, completed phases are continuously re-reconciled. If a managed object is modified externally (drift), the operator restores it to the desired state defined in the COS. The `completedAt` field in the COS status records when all phases first completed. If `completedAt` is set but `Available` is `False`, the revision has regressed — likely due to external drift that the operator is in the process of correcting.

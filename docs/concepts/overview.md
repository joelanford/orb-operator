# Concepts Overview

orb-operator manages Kubernetes objects through a layered system of deployments, revisions, and phases.

## Architecture

```
ClusterObjectDeployment (COD)
  └── stamps out → ClusterObjectSet (COS) revisions
                     └── each COS manages → Kubernetes objects
                           └── objects may be stored in → ClusterObjectSlice (COSL)
```

### ClusterObjectDeployment

A [ClusterObjectDeployment](cluster-object-deployment.md) is the mutable resource you create and update. It works like a Kubernetes Deployment: you declare the desired state in a template, and the controller creates immutable snapshots (revisions) whenever the template changes.

### ClusterObjectSet

A [ClusterObjectSet](cluster-object-set.md) is an immutable revision. Each time you update a COD's template, a new COS is created with an incremented revision number. The COS controller manages the actual Kubernetes objects, handles ownership transfers between revisions, and reports per-phase availability.

### ClusterObjectSlice

A [ClusterObjectSlice](cluster-object-slice.md) is an external content store. When your bundle of objects is too large to fit in a single COS resource (Kubernetes has a ~1.5 MiB etcd size limit), you store objects in ClusterObjectSlice resources and reference them from the COS phases.

## Key mechanisms

### Phased rollout

Objects are organized into [phases](phases-and-assertions.md). Phases are applied sequentially — the controller does not advance to the next phase until all objects in the current phase pass their assertions. This ensures dependencies are ready before dependents are created.

### Assertions

Each object can have [assertions](phases-and-assertions.md) — conditions that define what "ready" means. Assertions can check status conditions, compare fields, match field values, or evaluate arbitrary CEL expressions. Common types like CRDs and Deployments have built-in assertions that are applied automatically when no explicit assertions are specified.

### Revision transitions

When a new revision is created, the operator [safely transfers ownership](revision-management.md) of objects shared between revisions. Both the old and new revision are active simultaneously during the transition. Once the new revision is fully available, the old one is archived.

### Collision protection

[Collision protection](collision-protection.md) controls what happens when a revision tries to manage an object that already exists on the cluster. The default mode (`Prevent`) only manages objects the revision created. Other modes allow adopting existing objects.

### Drift detection

After initial rollout, the operator continuously re-reconciles completed phases to detect and correct configuration drift. If a managed object is modified externally, the operator restores it to the desired state.

# ADR-0001: Extension Object Management Architecture

- **Status:** Accepted
- **Date:** 2026-06-01

## Context

OLM v1 manages the lifecycle of Kubernetes extensions — installing, upgrading, and removing sets of Kubernetes resources (CRDs, Deployments, RBAC, etc.) on behalf of cluster administrators. This requires:

- **Phased rollout** — resources must be applied in dependency order (e.g. CRDs before Deployments that use them), with readiness gates between phases.
- **Safe revision transitions** — during upgrades, object ownership must transfer from the old revision to the new one without gaps or conflicts. Both revisions remain active until the new one fully rolls out.
- **Immutable revision records** — each deployed revision is an auditable, point-in-time snapshot of exactly what was applied.
- **Large bundle support** — extension bundles can exceed etcd's 1.5 MiB object size limit, so object manifests need an external storage mechanism.
- **Separation of concerns** — the object management layer should be usable independently of the extension resolution layer, by other controllers or directly by users.

## Decision

The extension object management architecture uses five cooperating API resources organized into three layers: resolution, orchestration, and content.

### Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│ ClusterCatalog    ClusterCatalog    ClusterCatalog       │
└──────────────────────────┬──────────────────────────────┘
                           │ resolves from
                    ┌──────▼──────┐
                    │             │
                    │ ClusterExt  │
                    │   ension    │
                    │             │
                    └──────┬──────┘
                           │ manages
                    ┌──────▼──────┐
                    │             │
                    │ ClusterObj  │
                    │   ectSet    │
                    │             │
                    └──┬───────┬──┘
             manages   │       │   manages
          ┌────────────▼─┐   ┌─▼────────────┐
          │ COS         │   │ COS         │
          │ rev: 1       │   │ rev: 2       │
          │ state:Active │   │ state:Active │
          └──┬───────────┘   └───────────┬──┘
        refs │                           │ refs
    ┌────────▼────────┐       ┌──────────▼──────┐
    │ ClusterObject   │       │ ClusterObject   │
    │    Slice        │       │    Slice        │
    └────────┬────────┘       └──────────┬──────┘
      embeds │                           │ embeds
    ┌ ─ ─ ─ ▼─ ─ ─ ─ ┐       ┌ ─ ─ ─ ─ ▼─ ─ ─ ┐
    │ CRD             │╌╌╌╌╌╌╌│ CRD             │
    │ Deployment      │╌╌╌╌╌╌╌│ Deployment      │
    │ Service         │╌╌╌╌╌╌╌│ Service         │
    │ ClusterRole     │╌╌╌╌╌╌╌│ ClusterRole     │
    │ ClusterRole     │       │ ClusterRole     │
    │   Binding       │╌╌╌╌╌╌╌│   Binding       │
    │ Validating      │       │ Validating      │
    │   Webhook       │       │   Admission     │
    │   Config        │       │   Policy        │
    └ ─ ─ ─ ─ ─ ─ ─ ─┘       └ ─ ─ ─ ─ ─ ─ ─ ─┘

    ╌╌╌ = ownership transfer during rollout
```

### Layer 1: Resolution — ClusterCatalog and ClusterExtension

**ClusterCatalog** declares a source of extension metadata (File-Based Catalog images). Multiple ClusterCatalogs can exist in a cluster. catalogd unpacks them and serves the metadata over HTTP.

**ClusterExtension** is the user-facing API. It declares intent to install a package by specifying constraints (package name, version range, channel, upgrade policy) and a ServiceAccount for RBAC scoping. The ClusterExtension controller resolves these constraints against the available ClusterCatalogs and manages a ClusterObjectDeployment to deploy the resolved content.

The ClusterExtension controller is the only component that interacts with catalogs and performs resolution. Everything below this layer is catalog-agnostic.

### Layer 2: Orchestration — ClusterObjectDeployment and ClusterObjectSet

**ClusterObjectDeployment (COD)** is a mutable, cluster-scoped resource analogous to a Deployment. It declares a template for ClusterObjectSets. When its spec changes, the COD controller stamps out a new COS from the template.

Spec fields:
- `progressDeadlineMinutes` — deadline for rollout progress
- `template.metadata` — labels and annotations propagated to stamped-out COSs (analogous to `Deployment.spec.template.metadata`). Callers attach arbitrary metadata here (package name, bundle version, service account info) without the COD API needing to know about those concerns.
- `template.spec` — the COS spec template (phases, collisionProtection, per-object assertions)

**ClusterObjectSet (COS)** is a point-in-time snapshot analogous to a ReplicaSet. COSs with the same `group` form a revision chain ordered by `revision`. The COS controller manages object ownership handoffs within a group.

Immutable spec fields:
- `group` — identifies the revision chain
- `revision` — monotonically increasing integer within the group
- `phases` — ordered list of phases with objects and per-object inline assertions
- `collisionProtection` — collision protection strategy

Mutable spec field:
- `lifecycleState` — `Active` or `Archived` (one-way transition)

COSs can be created by the COD controller or directly by users and other controllers. The `group` and `revision` fields give the COS controller everything it needs to manage handoffs regardless of who created the COS.

### Layer 3: Content — ClusterObjectSlice

**ClusterObjectSlice** is a cluster-scoped resource that carries embedded Kubernetes object definitions. It replaces the use of Secrets as an external storage mechanism for object manifests.

COSs reference ClusterObjectSlices rather than embedding all object manifests directly. This solves the etcd size limit problem: a single COS stays small (just references), while the actual manifests are distributed across one or more ClusterObjectSlice resources.

The caller (e.g. the ClusterExtension controller) creates ClusterObjectSlices. The COD and COS controllers only resolve refs — they never create, own, or manage ClusterObjectSlices.

### Object Lifecycle During Revision Transitions

When transitioning from revision N to revision N+1 within a group:

1. A new COS is created with `revision: N+1` and `lifecycleState: Active`.
2. Both COSs are active simultaneously. The COS controller begins transferring object ownership from revision N to revision N+1 as objects in the new revision become ready.
3. Phases roll out sequentially — each phase waits for all its objects to pass their inline assertions before the next phase begins.
4. Once the new revision succeeds, the old revision's `lifecycleState` is set to `Archived`.
5. Archival removes the old revision from the owner list of all managed objects. Objects that did not transition to the new revision are deleted.

Objects common to both revisions (shown as dashed lines in the diagram) transfer ownership seamlessly. Objects removed in the new revision are cleaned up during archival. Objects added in the new revision are created fresh.

### Per-Object Assertions

Readiness checks are defined as inline `assertions` on each object entry within a phase. The selector is implicit — assertions apply to the object they are colocated with. This replaces spec-level `progressionProbes` with selectors.

Assertion types:
- **ConditionEqual** — checks that an object has a condition of specified type and status
- **FieldsEqual** — checks that values at two field paths match
- **FieldValue** — checks that a field has a specific value

Several resource kinds have built-in assertions (e.g. CRD checks `Established=True`, Deployment checks `updatedReplicas == replicas`). Inline assertions are for custom resources or non-standard readiness criteria.

### Collision Protection

Collision protection controls whether a COS can adopt pre-existing objects. It is configured at three levels, with the most specific taking precedence: **object > phase > spec**.

- **Prevent** — only manages objects the revision created itself
- **IfNoController** — can adopt pre-existing objects not owned by another controller
- **None** — can adopt any pre-existing object, even if owned by another controller

### Labels and Annotations

The COD and COS APIs define no domain-specific labels. Caller-specific metadata (package name, bundle version, service account info) is passed through `template.metadata` on the COD, which propagates it to COSs. This keeps the orchestration layer decoupled from the resolution layer.

## Consequences

- **COD and COS are independently useful.** Any controller or user can create COSs directly (without a COD) to manage phased rollouts of arbitrary Kubernetes resources. The `group` and `revision` fields are sufficient for the COS controller to manage handoffs.
- **The ClusterExtension controller simplifies.** It resolves a bundle, creates ClusterObjectSlices with the content, and creates or updates a COD. It no longer directly manages revision numbers, Secret ownership, or object application.
- **ClusterObjectSlice replaces Secrets for content storage.** Object manifests are stored in purpose-built ClusterObjectSlice resources instead of being packed into Secrets. This provides a clearer API contract and avoids overloading the Secret type.
- **Template metadata is the extension point.** The COD `template.metadata` field is the seam between the resolution layer (which knows about packages and bundles) and the orchestration layer (which does not). Any future metadata needs from callers are satisfied by adding labels/annotations to the template, not by extending the COD/COS spec.
- **Single ownership is enforced.** Each Kubernetes resource is managed by at most one COS at a time within a group. Ownership transfers are coordinated by the COS controller during revision transitions.

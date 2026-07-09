---
status: done
---
# Migrate Controller Writes to Server-Side Apply (SSA)

## Summary

Migrated COS and COSR controller writes from a mix of `client.Create`, `client.Update`, and `client.Patch(MergePatch)` to Server-Side Apply (SSA). SSA gives field-level ownership tracking and built-in conflict detection for COSR metadata managed by the COS controller (owner references, lifecycle state, labels) and COSR finalizers managed by the COSR controller.

## Design

### Apply configuration generation

`controller-gen applyconfiguration` generates typed apply configurations for all API types under `api/`. A `//go:generate` directive in `generate.go` wires this into `make generate`. The generated code lives in `applyconfigurations/` and provides builder-style apply configs (e.g. `ClusterObjectSetRevisionApplyConfiguration`) with a per-type `Extract*` function that reads existing managed fields for a given field owner.

### Shared `applyCOSR` helper

All SSA writes to COSRs go through a single helper in `helpers.go`:

```go
func applyCOSR(ctx, client, cosr, fieldOwner, needsApply, mutate) (bool, error)
```

It follows extract-mutate-apply: extract the current apply config for the field owner, call the `mutate` callback to set desired fields, then `client.Apply` with `ForceOwnership`. The `needsApply` predicate short-circuits when no change is needed, avoiding unnecessary API calls.

### Field owner identities

Two field owners partition COSR metadata ownership:

| Field owner | Controller | Owns |
|---|---|---|
| `cos-controller` | COS reconciler | Owner reference, labels, `spec.lifecycleState` |
| `cosr-controller` | COSR reconciler | Finalizer |

Boxcutter's managed-object field owner is `cosr-group/<group>`, scoped per revision group, so managed object ownership transfers between revisions in the same group without conflicts.

### COS controller: COSR creation (two-step)

New COSRs use a two-step create-then-apply pattern:

1. `client.Create` with an unstructured object to create the resource (SSA cannot create objects with server-defaulted fields like `metadata.uid` on the first call because the apply config lacks them).
2. `client.Apply` immediately after to establish `cos-controller` field ownership over labels, owner reference, and spec fields.

### COS controller: field ownership reconciliation

On every reconcile, the COS controller extracts its current apply config for the latest COSR and compares it with the desired state. If they diverge (e.g. after manual edits or a controller restart before the follow-up apply), it re-applies to fix field ownership. This ensures the COS controller always owns the fields it manages, even for COSRs created before this migration.

### COS controller: adoption and archival

- **Adoption**: orphaned COSRs (no controller owner ref) are adopted via `applyCOSR` — the COS controller applies an owner reference.
- **Archival**: when the latest COSR is available, older COSRs get `spec.lifecycleState: Archived` via `applyCOSR`.

Both use the `cos-controller` field owner and the shared `applyCOSR` helper.

### COSR controller: finalizer via SSA

Finalizer addition uses `applyCOSR` with the `cosr-controller` field owner. The `needsApply` predicate checks `ContainsFinalizer` to skip when already present.

### COSR controller: finalizer removal via patch

Finalizer removal does **not** use SSA. Instead it uses `MergeFromWithOptimisticLock` patch because:

1. The finalizer must be removed atomically in a single API call.
2. The patch also clears the `cosr-controller` field ownership entry for the finalizer from `managedFields`, preventing a stale ownership record from surviving after the finalizer is gone.

`clearFinalizerFieldOwnership` directly manipulates the `managedFields` JSON to remove the finalizer key from the field owner's entry.

### COSR controller: cache sync after finalizer removal

After patching the finalizer off, `waitForFinalizerRemoval` polls the informer cache (50ms interval, 5s timeout) until the COSR either lacks the finalizer or is NotFound. This prevents a race where controller-runtime enqueues a reconcile from a watch event (e.g. managed object deletion) before the cache reflects the finalizer removal — without the wait, the stale cache read would re-enter the teardown path for a COSR that no longer needs it.

### COSR name validation

A validation was added requiring COSR names to be valid Kubernetes field owner strings (<=128 characters, no leading/trailing whitespace), since COSR names are used as part of boxcutter's field owner identity.

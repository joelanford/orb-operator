---
status: done
---
# Lightweight Teardown

## Summary

Replace full object resolution in the COS teardown path with lightweight
identity extraction. Today, `doTeardown` calls the same `resolveAndPrepare`
that `doReconcile` uses, which fetches every referenced ClusterObjectSlice
from the API server, JSON-unmarshals every object manifest into a full
`*unstructured.Unstructured`, computes a SHA-256 content hash, and verifies
it against `status.resolvedContentHash`. None of that work is needed for
teardown - boxcutter only uses each object's GVK, name, and namespace to
look it up on the cluster and delete it.

Beyond the unnecessary cost, the slice-fetch dependency creates a
correctness problem: if a ClusterObjectSlice is deleted before the COS is
torn down, the current path fails with a NotFound error and teardown stalls.
Lightweight identity extraction eliminates this failure mode since it reads
identity from the COS spec itself, not from the referenced slice.

## Design

### Identity-only resolution

Add a standalone `ResolveIdentities` function in `internal/object/` that
extracts object identity from `[]orbv1alpha1.Phase` without any API calls,
full JSON unmarshalling, or hash computation:

- **objectRef objects:** The `ObjectRef` embeds `ObjectKey`, which already
  carries `apiVersion`, `kind`, `name`, and `namespace`. Extract identity
  directly from the struct fields - no slice fetch needed.

- **Inline objects:** Partial-unmarshal `po.Object.Raw` into a small struct
  that captures only `apiVersion`, `kind`, `metadata.name`, and
  `metadata.namespace`. Skip the full `unstructured.UnmarshalJSON` that
  parses the entire manifest (spec, status, data, etc.).

The function returns an `*object.Result` (same type the full resolver
returns) with identity-only `*unstructured.Unstructured` objects. This
reuses the existing `ManagedObjects()` method and is compatible with
`revision.Build()` without changes to either.

### Teardown path changes

`doTeardown` stops calling `resolveAndPrepare` and instead:

1. Calls `object.ResolveIdentities(cos.Spec.Phases)` (no context, no reader)
2. Calls `r.newEngine(ctx, cos, resolved)` as before
3. Calls `revision.Build(cos, resolved, nil, r.ownerStrategy)` as before
4. Calls `engine.Teardown(ctx, rev, ...)` as before

Steps 2-4 are unchanged. The only difference is how the `*object.Result`
is produced.

### What stays the same

- `revision.Build` is unchanged. It receives identity-only objects but its
  callers (boxcutter's teardown engine) only read GVK + name + namespace.
  Assertion probes are nil since the identity result carries no assertions.
- `newEngine` is unchanged. `ManagedObjects()` still returns one object per
  unique GVK for the access manager's cache watches.
- `resolveAndPrepare` is unchanged. The reconcile path still uses full
  resolution with hash verification.
- boxcutter's `RevisionEngine.Teardown` is unchanged. It calls
  `phase.GetObjects()`, reads GVK + name/namespace, looks up the real
  object on cluster, and deletes it.

### What is eliminated for teardown

| Work | Where | Cost |
|---|---|---|
| Slice fetching | resolver.go `resolveRaw` | 1 API GET per referenced slice |
| Gzip decompression | resolver.go `resolveRaw` | CPU per compressed slice entry |
| Full JSON unmarshal | resolver.go `unmarshalUnstructured` | CPU + alloc per object |
| SHA-256 hash computation | resolver.go `Resolve` | CPU over all raw bytes |
| Hash verification | controller.go `resolveAndPrepare` | comparison (cheap, but conceptually wrong for teardown) |

# Requirements

- ClusterObjectSlice must have a root-level `objects` field (no `spec` wrapper) holding a list of `SliceObject` entries, each with explicit key fields (apiVersion, kind, name, namespace) and a `content` field (raw JSON or gzip-compressed JSON)
- The objects list must use `listType=map` with composite listMapKeys (apiVersion, kind, name, namespace) to enforce uniqueness at admission time
- `ClusterObjectSlice` must have a root-level `ObjectMap map[ObjectKey][]byte` field tagged `json:"-"` (following the `runtime.RawExtension` dual-representation pattern) — no custom MarshalJSON/UnmarshalJSON
- The COS controller must register an informer cache transform on ClusterObjectSlice that reindexes `Objects` into `ObjectMap` and nils `Objects` (cache is read-only, wire format not needed)
- Gzip decompression must happen lazily in the reconciler, only for objects actually referenced by a COS objectRef (auto-detect via magic number `0x1f 0x8b`)
- ObjectRef resolution must be an O(1) map lookup against `ObjectMap`, not a linear scan with deserialization
- PhaseObject must support an `objectRef` field mutually exclusive with the existing `object` field — exactly one must be set
- ObjectRef must identify a target object by sliceName, apiVersion, kind, name, and namespace (defaults to "" for cluster-scoped resources)
- The COS controller must resolve objectRef entries by fetching the named ClusterObjectSlice and matching the object by its identity fields
- ClusterObjectSlice `objects` field must be immutable after creation (CRD CEL validation on root type)
- Every COS reconcile must begin by resolving all phase objects (inline + objectRef); resolution failure short-circuits with an error condition
- On the first successful resolution, the COS controller must compute a hash of all resolved objects and store it in `status.resolvedContentHash` (set-once, immutable)
- On subsequent reconciles, the controller must verify the resolved content hash matches the stored value; mismatch short-circuits with an error condition
- The COS controller must watch ClusterObjectSlice resources so that COS reconciliation re-triggers when a referenced slice is created (a COS may pre-exist the slices it references)
- The COS controller must not create, own, or set owner references on ClusterObjectSlice resources
- Assertions and collision protection remain on PhaseObject — they apply regardless of whether the object is inline or referenced
- The COD controller requires no changes — objectRef flows through the template naturally
- All objectRef fields are immutable after COS creation (covered by existing phases immutability validation)

## Acceptance Criteria

- A COS with PhaseObjects using objectRef resolves them from the referenced ClusterObjectSlice and reconciles the objects identically to inline objects
- A COS can mix inline `object` and `objectRef` entries within the same phase
- A COS whose objectRef points to a nonexistent slice gets `Available=False/InvalidRevision` if never resolved before, or `Available=Unknown/InvalidRevision` if previously resolved
- A COS whose objectRef points to an object not present in the slice gets the same condition logic based on whether resolution previously succeeded
- A COS whose resolved content hash changes (e.g. slice was deleted and recreated with different content) gets `Available=Unknown/InvalidRevision`
- Validation rejects a PhaseObject where both `object` and `objectRef` are set
- Validation rejects a PhaseObject where neither `object` nor `objectRef` is set
- Existing COS behavior with inline objects is unchanged (no regressions)
- CRDs, deepcopy, and apply configurations are regenerated and consistent
- E2e scenarios cover the golden path (objectRef resolves and reconciles) and error paths (missing slice, missing object in slice)
- Unit tests cover objectRef resolution logic

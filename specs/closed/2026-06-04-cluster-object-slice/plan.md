# Implementation Plan

1. **Add ClusterObjectSlice fields at root level**
   - Add `SliceObject` struct with key fields (`APIVersion`, `Kind`, `Name`, `Namespace`) and `Content []byte` to `api/v1alpha1/types_clusterobjectslice.go`
   - Add `Objects []SliceObject` and `ObjectMap map[ObjectKey][]byte` (json:"-") directly on the `ClusterObjectSlice` type (no `spec` wrapper — pure content store like ConfigMap)
   - Remove the existing placeholder type and replace with the full definition
   - Add `+listType=map` with `+listMapKey=apiVersion,kind,name,namespace` to enforce unique keys at admission
   - Add immutability CEL on root type: `self.objects == oldSelf.objects`
   - Add kubebuilder validation markers (MinItems=1, MaxItems=256, field length constraints)
   - Update godoc to reflect the content store design

2. **Add ObjectRef type and update PhaseObject**
   - Add `ObjectRef` struct to `api/v1alpha1/types.go` with fields: `SliceName`, `APIVersion`, `Kind`, `Name`, `Namespace`
   - Add kubebuilder validation markers on each field (required/optional, length constraints)
   - Add `ObjectRef *ObjectRef` field to `PhaseObject`
   - Change `PhaseObject.Object` from required to optional
   - Add `ExactlyOneOf` kubebuilder validation to enforce mutual exclusivity of `object` and `objectRef`

3. **Regenerate code**
   - Run `make generate` to regenerate CRDs, deepcopy, and apply configurations
   - Run `make verify` to confirm generated code is consistent

4. **Add resolved content hash to COS status**
   - Add `resolvedContentHash` field (string) to `ClusterObjectSetStatus`
   - Add CRD CEL immutability rule: once set, cannot be changed (same pattern as `completedAt`)

5. **Add informer transform**
   - Register an informer cache transform on ClusterObjectSlice (`cache.Options.ByObject`)
   - Transform: reindex `Objects` into `ObjectMap` (no decompression), nil `Objects` (cache is read-only)

6. **Update COS controller: object resolution and hash verification**
   - Add a resolve phase at the start of every reconcile that resolves all phase objects (inline objects used directly, objectRef entries resolved via `slice.ObjectMap` O(1) lookup)
   - Lazily decompress gzip content at point-of-use (only for objects actually referenced)
   - Fetch slices via `client.Get` (served from the informer cache — no extra caching layer needed)
   - After successful resolution, compute a deterministic hash of all resolved objects
   - If `status.resolvedContentHash` is empty, set it to the computed hash (first successful resolution)
   - If `status.resolvedContentHash` is already set, verify it matches the computed hash; mismatch short-circuits with an error condition
   - Only after resolution + hash verification, proceed to build the boxcutter revision and reconcile/teardown
   - Update `managedObjectsForCOS` to handle PhaseObjects with `objectRef`

7. **Add ClusterObjectSlice watch to COS controller**
   - Add a watch on ClusterObjectSlice resources to the COS controller setup (a COS may be created before the slices it references exist)
   - Map slice create events to COS reconcile requests (enqueue COSs that reference the new slice)
   - Add RBAC markers for ClusterObjectSlice read access

8. **Error handling**
   - Resolution failure (slice not found, object not found in slice): set error condition in status, short-circuit reconcile
   - Hash mismatch: set error condition in status, short-circuit reconcile
   - Follow existing patterns (e.g., `ReasonInvalidRevision`) for surfacing errors in COS conditions

9. **Add e2e scenarios**
   - Golden path: COS with objectRef entries resolves from a ClusterObjectSlice and reconciles to Available
   - Mixed: COS with both inline objects and objectRef entries in the same phase
   - Error: COS with objectRef pointing to nonexistent slice — verify error condition
   - Error: COS with objectRef pointing to object not present in slice — verify error condition
   - Hash mismatch: delete and recreate a slice with different content — verify error condition

10. **Add unit tests**
    - Test the slice resolution helper: correct match, no match, missing slice
    - Test hash computation: deterministic, changes when content changes
    - Test hash verification: first resolution sets hash, subsequent match required, mismatch detected

# Implementation Plan

1. **Add `ResolveIdentities` in `internal/object/identity.go`**

   Define a small unexported struct for partial JSON unmarshal of inline
   objects (apiVersion, kind, metadata.name, metadata.namespace). Implement
   `ResolveIdentities(phases []orbv1alpha1.Phase) (*Result, error)` that:

   - Iterates phases and phase objects
   - For objectRef: builds an identity-only `*unstructured.Unstructured`
     from `ObjectRef.ObjectKey` fields
   - For inline: partial-unmarshals `po.Object.Raw` into the identity
     struct, then builds the `*unstructured.Unstructured`
   - Copies per-object and per-phase `CollisionProtection` into the result
     (same as the full resolver does)
   - Sets `Result.Hash` to empty string (not computed)

2. **Add unit tests in `internal/object/identity_test.go`**

   Test cases:
   - objectRef phase object produces correct GVK + name + namespace
   - Inline phase object produces correct GVK + name + namespace
   - Mixed phase with both objectRef and inline objects
   - Multiple phases with overlapping GVKs (verify `ManagedObjects` dedup)
   - Cluster-scoped inline object (no namespace)
   - Malformed inline JSON returns error
   - Phase object with neither object nor objectRef returns error

3. **Add e2e scenario for teardown after slice deletion**

   Add a godog scenario that:
   - Creates a COS with phases referencing a ClusterObjectSlice
   - Waits for the COS to become Available (objects applied)
   - Deletes the ClusterObjectSlice
   - Archives the COS (triggers teardown)
   - Verifies teardown succeeds and the managed objects are deleted

   This is the motivating case: with full resolution, this scenario would
   fail because the slice fetch returns NotFound. With lightweight identity
   extraction, teardown reads identity from the ObjectRef and succeeds.

4. **Modify `doTeardown` in `internal/controller/cos/controller.go`**

   Replace the `resolveAndPrepare` call with:
   ```
   resolved, err := object.ResolveIdentities(cos.Spec.Phases)
   ```
   Keep the rest of the method unchanged (`newEngine`, `revision.Build`,
   `engine.Teardown`).

5. **Run verification**

   - `make test-unit` (new tests + existing tests pass)
   - `make test-e2e` (teardown scenarios pass)
   - `make verify` (lint, build, generate all clean)

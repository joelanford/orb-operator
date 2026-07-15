# Requirements

- Teardown must succeed even when referenced ClusterObjectSlices have
  already been deleted.
- Teardown must not fetch ClusterObjectSlices from the API server.
- Teardown must not fully unmarshal inline object manifests.
- Teardown must not compute or verify content hashes.
- The reconcile path must be completely unaffected.
- `ResolveIdentities` must be a standalone function with no I/O dependencies
  (no `client.Reader`, no context parameter).
- The returned `*object.Result` must be compatible with the existing
  `ManagedObjects()`, `revision.Build()`, and `newEngine()` call sites
  without changes to any of them.

## Acceptance Criteria

- `ResolveIdentities` correctly extracts identity from objectRef phase
  objects using the embedded `ObjectKey` fields.
- `ResolveIdentities` correctly extracts identity from inline phase objects
  via partial JSON unmarshal of `apiVersion`, `kind`, `metadata.name`, and
  `metadata.namespace`.
- `ResolveIdentities` returns an error for inline objects with malformed
  JSON (matching current behavior where `unmarshalUnstructured` would fail).
- `ManagedObjects()` on the identity-only result returns the correct set of
  unique GVKs.
- All existing e2e teardown scenarios pass without modification.
- A new e2e scenario verifies teardown succeeds for a COS that references a
  ClusterObjectSlice when the slice is deleted before teardown begins.
- Unit test coverage for `ResolveIdentities` covers: objectRef-only phases,
  inline-only phases, mixed phases, multiple phases, and error cases.

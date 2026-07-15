# Verification

## Implementation Correctness

- [ ] `ResolveIdentities` is a standalone function (no receiver, no
      `context.Context`, no `client.Reader`)
- [ ] objectRef identity comes from `ObjectRef.ObjectKey` fields, not from
      a slice fetch
- [ ] Inline identity comes from partial JSON unmarshal, not full
      `unstructured.UnmarshalJSON`
- [ ] The partial unmarshal struct only captures `apiVersion`, `kind`,
      `metadata.name`, `metadata.namespace` (no spec, status, data, etc.)
- [ ] `Result.Hash` is empty string in the identity result
- [ ] `CollisionProtection` is propagated at both phase and object level
- [ ] `doTeardown` no longer calls `resolveAndPrepare`
- [ ] `doTeardown` no longer passes through hash verification
- [ ] The reconcile path (`doReconcile` / `resolveAndPrepare`) is unchanged
- [ ] e2e scenario confirms teardown succeeds after the referenced
      ClusterObjectSlice has been deleted

## Project Conventions

- [ ] No `//nolint` comments added
- [ ] `make lint` passes
- [ ] `make test-unit` passes with new and existing tests
- [ ] `make test-e2e` passes (teardown scenarios exercise the new path)
- [ ] `make verify` passes (lint + generate + build)
- [ ] New code has at least 70% statement coverage
- [ ] Commit message follows conventional commits format

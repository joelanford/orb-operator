# Verification

## Implementation Correctness

- [ ] ClusterObjectSlice CRD has root-level `objects` as a `listType=map` of `SliceObject` with composite keys (no `spec` wrapper)
- [ ] SliceObject has explicit key fields (apiVersion, kind, name, namespace) and `content` field (`[]byte`, raw JSON or gzip)
- [ ] Duplicate keys in a slice are rejected at admission time
- [ ] `ClusterObjectSlice` has root-level `ObjectMap map[ObjectKey][]byte` field tagged `json:"-"` — no custom JSON methods on the API type
- [ ] Informer cache transform reindexes `Objects` into `ObjectMap` and nils `Objects` (no decompression in transform)
- [ ] Gzip decompression happens lazily in the reconciler, only for objects actually referenced by a COS objectRef
- [ ] ObjectRef resolution uses O(1) `ObjectMap` lookup, not linear scan with deserialization
- [ ] PhaseObject enforces exactly-one-of `object` / `objectRef` via kubebuilder validation
- [ ] ObjectRef has all required fields: sliceName, apiVersion, kind, name, namespace (defaults to "" for cluster-scoped)
- [ ] COS controller resolves objectRef entries by fetching the named slice and matching the object identity
- [ ] `managedObjectsForCOS` correctly discovers GVKs from both inline objects and objectRef entries
- [ ] `buildRevisionWithSiblings` builds boxcutter phases from both inline objects and objectRef entries
- [ ] ClusterObjectSlice `objects` is immutable after creation (CRD CEL: `self.objects == oldSelf.objects`)
- [ ] COS status has `resolvedContentHash` field, set-once and immutable (CRD CEL)
- [ ] Every reconcile starts with object resolution; failure short-circuits with error condition
- [ ] First successful resolution computes and stores hash in `status.resolvedContentHash`
- [ ] Subsequent reconciles verify resolved content matches stored hash; mismatch short-circuits with error condition
- [ ] COS controller watches ClusterObjectSlice resources and re-reconciles affected COSs when a referenced slice is created
- [ ] COS controller does not create, own, or set owner references on ClusterObjectSlice resources
- [ ] RBAC markers grant the COS controller read access to ClusterObjectSlice resources
- [ ] First-time resolution failure sets `Available=False/InvalidRevision` (resolvedContentHash empty)
- [ ] Post-resolution failure sets `Available=Unknown/InvalidRevision` (resolvedContentHash already set)
- [ ] Hash mismatch sets `Available=Unknown/InvalidRevision`
- [ ] No panics on any resolution failure path
- [ ] Existing inline-object COS behavior is unchanged (all existing e2e tests pass)
- [ ] CRDs, deepcopy, and apply configurations are regenerated and consistent (`make verify` passes)

## Project Conventions

- [ ] No `//nolint` comments added (per `specs/conventions.md`)
- [ ] ADR-0001 compliance: slice is content-only, COS/COD controllers only resolve refs
- [ ] ADR-0001 compliance: COS remains an immutable snapshot (objectRef fields are immutable)
- [ ] Standard controller patterns: idiomatic reconcile loop, watches, no custom frameworks (per `specs/mission.md`)
- [ ] Test coverage: e2e godog scenarios for golden path and error paths (per `specs/mission.md`)
- [ ] Test coverage: unit tests for slice resolution logic with testify (per `specs/tech-stack.md`)
- [ ] `make lint` passes
- [ ] `make test-unit` passes
- [ ] `make test-e2e` passes
- [ ] `make verify` passes (lint + generate diff + goreleaser check + build)

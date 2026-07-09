# Implementation Plan

1. **Generate apply configurations** — add `//go:generate` directive for `controller-gen applyconfiguration` targeting `api/` types; add group version marker to `doc.go`; run `make generate` to produce `applyconfigurations/`.

2. **Add COSR name validation** — validate COSR names fit Kubernetes field owner constraints (<=128 chars, no leading/trailing whitespace) since they become part of the boxcutter field owner string.

3. **Introduce shared `applyCOSR` helper** — extract-mutate-apply pattern in `helpers.go` used by both controllers.

4. **Migrate COSR finalizer add to SSA** — `ensureFinalizer` uses `applyCOSR` with `cosr-controller` field owner.

5. **Migrate COSR archival to SSA** — `archiveOlderRevisions` uses `applyCOSR` with `cos-controller` field owner to set `spec.lifecycleState: Archived`.

6. **Migrate COSR adoption to SSA** — `adoptCOSR` uses `applyCOSR` with `cos-controller` field owner to apply the owner reference.

7. **Migrate COSR creation to two-step create-then-apply** — `client.Create` with unstructured to create the object, then `client.Apply` to establish field ownership.

8. **Add field ownership reconciliation on every COS reconcile** — extract the COS controller's current apply config for the latest COSR, compare with desired, re-apply if divergent.

9. **Migrate finalizer removal to optimistic-lock patch with field ownership cleanup** — `removeFinalizer` uses `MergeFromWithOptimisticLock` patch; `clearFinalizerFieldOwnership` strips the finalizer key from the `cosr-controller` managed fields entry.

10. **Add cache sync wait after finalizer removal** — `waitForFinalizerRemoval` polls the informer cache until the finalizer is gone or the COSR is deleted.

# Implementation Plan

1. **Add boxcutter dependency**
   - `go get pkg.package-operator.run/boxcutter`
   - `go mod tidy`
   - Confirm `make build` succeeds

2. **CRD CEL validation rules**
   - Add `+kubebuilder:validation:XValidation` markers to the root COSR type for name enforcement
   - Add `+kubebuilder:validation:XValidation` markers to `ClusterObjectSetRevisionSpec` for:
     - `group` immutability (transition rule with `oldSelf`)
     - `revision` immutability
     - `phases` immutability
     - `collisionProtection` immutability
     - `lifecycleState` one-way Active→Archived
   - Run `make generate` to regenerate CRDs
   - Confirm validation rules appear in the CRD YAML
   - Confirm `make verify` passes

3. **Assertion → probe mapping** (`internal/assertions/`)
   - Create `internal/assertions/probes.go`
   - Function `ProbesForAssertions(assertions []v1alpha1.Assertion) ([]boxcutter.Prober, error)` that converts:
     - `ConditionEqual` → `probing.ConditionProbe{Type, Status}`
     - `FieldsEqual` → `probing.FieldsEqualProbe{FieldA, FieldB}`
     - `FieldValue` → `probing.FieldValueProbe{FieldPath, Value}`
     - `CELExpression` → `probing.NewCELProbe(expression, message)`
   - Unit tests for each assertion type mapping

4. **COSR reconciler** (`internal/controller/cosr_controller.go`)
   - Implement `Reconciler` struct with `boxcutter.RevisionEngine`, `managedcache.ObjectBoundAccessManager`, `client.Client`
   - `SetupWithManager(mgr)`:
     - Create `ObjectBoundAccessManager[*v1alpha1.ClusterObjectSetRevision]` for per-COSR caches
     - Add the access manager to the controller-runtime manager as a Runnable
     - Create boxcutter RevisionEngine using the manager's scheme/REST mapper (Reader/Writer come from per-COSR accessor at reconcile time)
     - Field index on `spec.group` for group-based queries
     - Watch COSRs (primary)
     - Watch managed objects via `accessManager.Source()` with `EnqueueWatchingObjects` handler
     - Map COSR changes to all COSRs in the same group (fan-out)
   - `Reconcile(ctx, req)`:
     - Get the COSR
     - Get or create the per-COSR `Accessor` via `accessManager.Get(ctx, cosr)`
     - Add finalizer (for teardown on deletion)
     - List all COSRs in the same group, sort by revision
     - Determine role: archived, superseded (older active with newer complete), or latest active
     - **Archived**: teardown via `engine.Teardown`, set Available=False/Archived, remove finalizer if teardown complete
     - **Superseded**: set Available=False/Superseded, set lifecycleState=Archived
     - **Latest active**: build boxcutter Revision from phases, map assertions to probes, reconcile via `engine.Reconcile` with `WithPreviousOwners` from older revisions, map result → status conditions
   - Status update: set the `Available` condition based on the reconciliation outcome

5. **Wire up in main.go**
   - Import `internal/controller`
   - Call controller's `SetupWithManager(mgr)` after manager creation
   - Confirm `make build` succeeds

6. **Integration verification**
   - Run `make verify` (lint, generated code, build)
   - Run `make test-e2e` and confirm all 22 scenarios pass
   - If any test fails, debug and fix the controller (not the tests) unless test code has a demonstrable bug

# Implementation Plan

1. **COSL API changes**
   - Add `count` (int32) field to `ClusterObjectSlice` in
     `api/v1alpha1/types_clusterobjectslice.go`.
   - Add `+kubebuilder:printcolumn` annotation for OBJECTS column
     with JSONPath `.count`.
   - Add type-level `+kubebuilder:validation:XValidation` rule:
     `self.count == self.objects.size()`.
   - Run `make generate` to regenerate CRDs and deepcopy.

2. **MAP and MAPB in jsonnet**
   - Add `mapCOSLCount` and `mapCOSLCountBinding` locals to
     `deploy/lib/api.libsonnet`, following the existing VAP/VAPB pattern.
   - Include both in the `generate()` output array.

3. **Tests**
   - Envtest: verify MAP sets `count` when not provided.
   - Envtest: verify MAP overwrites incorrect `count`.

4. **Envtest infrastructure**
   - Update `InstallAPI` to wait for MAP dispatcher sync via dry-run
     CREATE after CRD establishment. Without this, the MAP dispatcher
     returns ServiceUnavailable for CRD resources it targets until its
     type resolver syncs with newly-created CRDs.

5. **Verify**
   - `make verify`
   - `make test-unit`
   - `make test-e2e`

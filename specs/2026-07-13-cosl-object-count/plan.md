# Implementation Plan

1. **COSL API changes**
   - Add `objectCount` (int32) to `ClusterObjectSlice`.
   - Add OBJECTS printer column annotation.
   - Run `make generate`.

2. **MutatingAdmissionPolicy**
   - Create the MutatingAdmissionPolicy and MutatingAdmissionPolicyBinding manifests.
   - Add to the operator's deployment jsonnet.

3. **Tests**
   - E2e test: create a COSL and verify `objectCount` is set.

4. **Verify**
   - `make verify`.
   - `make test-unit`.
   - `make test-e2e`.

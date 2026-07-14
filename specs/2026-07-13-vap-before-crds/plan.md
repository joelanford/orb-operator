# Implementation Plan

1. **Create `deploy/lib/api.libsonnet`**
   - Move the three VAP definitions (cos-name-must-match-group-revision,
     cos-orphan-finalizer-ordering, cod-name-max-length) and their
     bindings from `main.jsonnet` into this library.
   - Import the three CRD files (same `importstr` pattern).
   - Export a function or value returning the ordered list: VAPs, VAPBs,
     then CRDs.

2. **Create `deploy/lib/controller.libsonnet`**
   - Move Namespace, ServiceAccount, ClusterRoleBinding, Deployment
     (with e2e profile logic), and Service into this library.
   - Accept parameters: `image`, `namespace`, `profiles`.

3. **Rename `deploy/operator.jsonnet` → `deploy/main.jsonnet`**
   - Rename the file.
   - Update the Makefile reference.
   - Import `api.libsonnet` and `controller.libsonnet`.
   - Concatenate `api + controller` into the final List.
   - Verify output is identical to current (modulo ordering).

4. **Create shared envtest helper**
   - Create `internal/testutil/envtest.go` (or similar) with a function
     that renders `api.libsonnet` via `go tool jsonnet`, parses the
     output, and returns the VAP/VAPB objects as unstructured.

   Envtest startup sequence in each suite's `TestMain`:
   1. `testEnv.Start()` with no `CRDDirectoryPaths`.
   2. Create client with `apiutil.NewDynamicRESTMapper` for lazy
      discovery.
   3. Directly create all objects from the API manifests (including
      CRDs), then wait for CRDs to be established.
   4. Dynamic mapper discovers CRD types on first use in tests.

5. **Update envtest TestMain in `api/v1alpha1/`**
   - Use the shared helper to apply VAP/VAPBs after environment start.

6. **Update envtest TestMain in `internal/cosutil/`**
   - Same approach as step 5.

7. **Verify**
   - `make verify` passes.
   - `make test-unit` passes.
   - `make test-e2e` passes.

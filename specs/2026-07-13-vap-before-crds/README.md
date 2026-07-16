---
status: in-progress
---
# VAP/VAPB Before CRDs

## Summary

Restructure operator deployment manifests so ValidatingAdmissionPolicies
and their bindings are installed before CRDs. This ensures admission
policies are active from the moment CRD-based resources can be created.
The same ordering applies to both `make run` (kind cluster) and envtest
integration tests.

## Design

### Jsonnet library restructure

Split `deploy/main.jsonnet` into shared libraries:

- **`deploy/lib/api.libsonnet`** — exports an ordered list: VAP/VAPBs
  first, then CRDs. These are the "API surface" objects that must exist
  before any orb resources are created.
- **`deploy/lib/controller.libsonnet`** — exports everything else:
  Namespace, ServiceAccount, ClusterRoleBinding, Deployment, Service.
  Accepts parameters (image, namespace, profiles).

`deploy/main.jsonnet` (renamed from `deploy/operator.jsonnet`) becomes
a thin orchestrator that imports both libraries and concatenates
`api + controller` into the final List.

### Envtest integration

The two envtest suites (`api/v1alpha1/`, `internal/cosutil/`) currently
install only CRDs via `envtest.Environment.CRDDirectoryPaths`. To add
VAP/VAPB support with strict ordering:

1. `testEnv.Start()` with no `CRDDirectoryPaths` — starts the API
   server and etcd only.
2. Create a single `client.New()` using `apiutil.NewDynamicRESTMapper`
   so it lazily discovers resource types at runtime.
3. A shared helper (`internal/testutil/`) shells out to
   `go tool jsonnet deploy/lib/api.libsonnet`, parses the output,
   and directly creates all objects (including CRDs) via the client,
   then waits for CRDs to be established.
4. The dynamic REST mapper discovers CRD types on first use - the
   same client works for both VAPs and orb resources.

VAP is GA in Kubernetes 1.30+ (envtest uses k8s 1.36), so no feature
gate configuration is needed. The VAPs are enforced during tests.

### `make run` ordering

The Makefile's `make run` currently renders `main.jsonnet` and
pipes everything through `kubectl apply -f -` in one shot. Since
main.jsonnet now emits `api + workload` in order, and `kubectl
apply` processes items sequentially within a List, the VAPs and VAPBs
are naturally created before the Deployment starts creating resources.
No Makefile changes needed beyond the jsonnet restructure.

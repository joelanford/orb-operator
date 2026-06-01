# Requirements

- Empty Go types for ClusterObjectSet, ClusterObjectSetRevision, and ClusterObjectSlice in `api/v1alpha1/` with correct controller-gen markers (cluster-scoped, subresource:status, short names)
- `go generate ./...` produces CRD YAML in `deploy/crds/` and `zz_generated.deepcopy.go`
- The v1alpha1 scheme is registered with the controller-runtime manager
- The metrics endpoint serves on `:8443` over HTTPS with authn/authz via `filters.WithAuthenticationAndAuthorization`
- `deploy/operator.jsonnet` renders Namespace, ServiceAccount, ClusterRoleBinding (to cluster-admin), Deployment, and Service as a JSON List
- Jsonnet is parameterized by `image` and `namespace` external variables
- `make image` builds the container image via goreleaser snapshot
- `make kind-cluster` creates a kind cluster; `make kind-cluster-delete` destroys it
- `make kind-load` loads the built image into the kind cluster
- `make deploy` applies CRDs and rendered jsonnet manifests; `make undeploy` removes them
- `make verify` continues to pass (existing gate)
- `make lint` continues to pass (existing gate)

## Acceptance Criteria

- `make generate` produces three CRD YAML files in `deploy/crds/` (one per type) and deepcopy code compiles
- `make verify` passes (generated code is up to date, lint clean, build succeeds)
- `make image` produces a loadable container image
- `make kind-cluster image kind-load deploy` results in a running operator pod in `orb-operator-system`
- `kubectl get cos,cosr,cosl` returns empty lists (CRDs are registered)
- The operator pod logs show the manager starting and the metrics server listening on `:8443`
- `make undeploy kind-cluster-delete` cleanly tears everything down

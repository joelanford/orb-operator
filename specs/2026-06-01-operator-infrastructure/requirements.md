# Requirements

- Empty Go types for ClusterObjectSet, ClusterObjectSetRevision, and ClusterObjectSlice in `api/v1alpha1/` with correct controller-gen markers (cluster-scoped, subresource:status, short names)
- `go generate ./...` produces CRD YAML in `deploy/crds/` and `zz_generated.deepcopy.go`
- The v1alpha1 scheme is registered with the controller-runtime manager
- The metrics endpoint serves on `:8443` over HTTPS with authn/authz via `filters.WithAuthenticationAndAuthorization`
- `deploy/operator.jsonnet` renders Namespace, ServiceAccount, ClusterRoleBinding (to cluster-admin), Deployment, and Service as a JSON List
- Jsonnet is parameterized by `image` and `namespace` external variables
- `make run` builds the image, creates a kind cluster (if needed), loads the image, and applies manifests
- `make verify` continues to pass (existing gate)
- `make lint` continues to pass (existing gate)

## Acceptance Criteria

- `make generate` produces three CRD YAML files in `deploy/crds/` (one per type) and deepcopy code compiles
- `make verify` passes (generated code is up to date, lint clean, build succeeds)
- `make run` results in a running operator pod in `orb-operator-system`
- `kubectl get cos,cosr,cosl` returns empty lists (CRDs are registered)
- The operator pod logs show the manager starting and the metrics server listening on `:8443`

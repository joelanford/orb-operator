# Verification

## Implementation Correctness

- [ ] `make generate` produces three CRD files in `deploy/crds/` (`orb.operatorframework.io_clusterobjectsets.yaml`, `orb.operatorframework.io_clusterobjectsetrevisions.yaml`, `orb.operatorframework.io_clusterobjectslices.yaml`)
- [ ] `zz_generated.deepcopy.go` is generated and compiles
- [ ] All three CRDs are cluster-scoped with `subresource:status` (COS and COSR) or without (ClusterObjectSlice)
- [ ] Short names work: `cos`, `cosr`, `cosl`
- [ ] Manager registers the v1alpha1 scheme
- [ ] Metrics server binds to `:8443` with `SecureServing: true` and `FilterProvider` set
- [ ] `deploy/operator.jsonnet` renders valid JSON containing all five resource kinds
- [ ] Jsonnet external variables `image` and `namespace` control the output
- [ ] `make image` produces a container image via goreleaser snapshot
- [ ] `make kind-cluster image kind-load deploy` results in a running pod
- [ ] `kubectl get cos,cosr,cosl` returns empty lists
- [ ] `make undeploy kind-cluster-delete` cleans up without errors

## Project Conventions

- [ ] Commit messages use conventional commits format
- [ ] No `//nolint` comments added
- [ ] `make lint` passes
- [ ] `make verify` passes (generated code up to date, goreleaser check, build)
- [ ] `make test-unit` passes
- [ ] Go types follow standard controller-runtime patterns (TypeMeta, ObjectMeta, Spec, Status, List)
- [ ] Import aliases match `.golangci.yml` conventions (metav1, utilruntime, ctrl, etc.)
- [ ] API group is `orb.operatorframework.io` as declared in existing `doc.go`

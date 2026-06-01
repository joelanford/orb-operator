# Implementation Plan

1. **Goreleaser snapshot tag**
   - Update `.goreleaser.yml` `snapshot.version_template` from `{{ .ShortCommit }}` to `dev` so `make image` produces the `ghcr.io/joelanford/orb-operator:dev` tag by default

2. **Tool dependencies**
   - Add `github.com/google/go-jsonnet/cmd/jsonnet` as a `tool` directive in `go.mod` (for `go tool jsonnet`)
   - Add `sigs.k8s.io/kind` as a `tool` directive in `go.mod` (for `go tool kind`)
   - Run `go mod tidy`

3. **API type stubs and generation**
   - Create `api/v1alpha1/groupversion_info.go` with `SchemeBuilder`, `GroupVersion`, `AddToScheme`
   - Create `api/v1alpha1/types_clusterobjectset.go` — `ClusterObjectSet`, `ClusterObjectSetSpec`, `ClusterObjectSetStatus`, `ClusterObjectSetList` with controller-gen markers (`+kubebuilder:object:root`, `+kubebuilder:subresource:status`, `+kubebuilder:resource:scope=Cluster,shortName=cos`)
   - Create `api/v1alpha1/types_clusterobjectsetrevision.go` — same pattern, short name `cosr`
   - Create `api/v1alpha1/types_clusterobjectslice.go` — same pattern, short name `cosl` (no subresource:status — slices are pure content)
   - Update `api/v1alpha1/doc.go` with `go:generate` directive for controller-gen: `controller-gen crd output:crd:dir=../../deploy/crds paths=./... && controller-gen object paths=./...`
   - Run `make generate` and confirm CRD YAML files appear in `deploy/crds/` and `zz_generated.deepcopy.go` compiles

4. **Manager configuration**
   - Update `cmd/operator/main.go`:
     - Import `api/v1alpha1` and register scheme via `utilruntime.Must(v1alpha1.AddToScheme(scheme))`
     - Configure `ctrl.Options` with `Metrics: metricsserver.Options{BindAddress: ":8443", SecureServing: true, FilterProvider: filters.WithAuthenticationAndAuthorization}`
   - Confirm `make build` succeeds

5. **Jsonnet operator manifests**
   - Replace the placeholder `deploy/operator.jsonnet` with the full manifest rendering:
     - Namespace, ServiceAccount, ClusterRoleBinding (to cluster-admin), Deployment (single replica, port 8443, non-root), Service (metrics on 8443)
     - `image` is a required external variable (no default — jsonnet errors if not provided); `namespace` defaults to `orb-operator-system`
   - Remove `deploy/crds/.gitkeep` (replaced by generated CRD files)

6. **Makefile targets**
   - Add `IMAGE` variable defaulting to `ghcr.io/joelanford/orb-operator:dev`
   - Add `NAMESPACE` variable defaulting to `orb-operator-system`
   - Add `run` target: builds image, creates kind cluster (if needed), loads image, applies manifests
   - Confirm `make verify` still passes

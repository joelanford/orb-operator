---
status: in-progress
---
# Code Coverage Collection

## Summary

Add coverage profiling to all test targets so we can measure which lines of operator code are exercised by unit tests, e2e tests, and both combined. `test-unit` and `test-e2e` each produce coverage as a side effect. A new `test-coverage` target merges them into a single report, and `test-all` calls it automatically.

## Design

### Unit coverage (`test-unit`)

`go test -coverprofile -coverpkg=./internal/...,./api/...` writes a text coverage profile directly. The `-coverpkg` flag ensures coverage is measured across all operator packages, not just the package under test. The profile is written to `_output/unit/coverage.out`. The `KUBEBUILDER_ASSETS` setup remains unchanged.

### E2E coverage (`test-e2e`)

Go's `-cover` flag on `go build` instruments the binary to write coverage counter files. The `.goreleaser.yml` config includes a templated `flags` entry that passes `{{ .Env.GO_BUILD_FLAGS }}` to `go build`. For e2e, `GO_BUILD_FLAGS='-cover -tags=cover -covermode=atomic'`.

The flow:

1. Build with goreleaser: `GO_BUILD_FLAGS='-cover -tags=cover -covermode=atomic' go tool goreleaser release --snapshot --clean`
2. Deploy to Kind with the `e2e` profile (adds `GOCOVERDIR=/coverage`, emptyDir volume, `imagePullPolicy: Never`)
3. Run e2e tests
4. Signal the operator to flush coverage: `docker exec <kind-node> pkill -USR1 -f '/orb-operator$'` (distroless has no `kill` binary, so signal is sent from the Kind node)
5. Copy coverage files from the Kind node's kubelet volume path: `docker cp <kind-node>:/var/lib/kubelet/pods/<pod-uid>/volumes/kubernetes.io~empty-dir/coverage/. _output/e2e/covdata/` (distroless has no `tar`, so `kubectl cp` doesn't work either)
6. Convert to text format with `go tool covdata textfmt` → `_output/e2e/coverage.out`

#### SIGUSR1 coverage flush (`cmd/operator/covflush.go`)

A `covflush.go` file with `//go:build cover` registers a SIGUSR1 handler in an `init()` function. On signal, it calls `runtime/coverage.WriteCountersDir` and `runtime/coverage.WriteMetaDir` to flush coverage data to `GOCOVERDIR` without terminating the process.

This requires `-covermode=atomic` (not the default `set`) because `WriteCountersDir` only supports atomic mode. The `-tags cover` flag activates the build-tagged file.

The operator pod stays running after coverage collection regardless of test outcome, so pod logs remain available for debugging.

#### Counter file validation

Before running `covdata textfmt`, check that `covcounters.*` files exist in the collected directory. Fail loudly if they don't, to avoid silently producing a valid-but-empty 0.0% coverage profile.

### Merged coverage (`test-coverage`)

Concatenates the two text profiles from `_output/unit/coverage.out` and `_output/e2e/coverage.out` into `_output/merged/coverage.out` with a single `mode: set` header. Prints a summary with `go tool cover -func`.

### `test-all` integration

`test-all` depends on `test-coverage` (which depends on `test-unit` and `test-e2e`), so running `make test-all` produces the merged report automatically.

### Goreleaser changes (`.goreleaser.yml`)

Add a templated `flags` entry to the build config and a corresponding `env` entry with a default:

```yaml
env:
  - CGO_ENABLED=0
  - GOFLAGS={{ if index .Env "GO_BUILD_FLAGS" }}{{ .Env.GO_BUILD_FLAGS }}{{ end }}
```

The `GO_BUILD_FLAGS` env var is mapped to `GOFLAGS` in goreleaser's build subprocess environment. Go handles parsing the flags natively. When `GO_BUILD_FLAGS` is unset, `GOFLAGS` resolves to empty and goreleaser builds normally.

Note: flags with values must use `=` syntax (e.g. `-tags=cover` not `-tags cover`) because `GOFLAGS` splits on spaces.

### Jsonnet changes (`deploy/operator.jsonnet`)

Add a `profiles` external variable (array of strings, default `[]`). The Makefile passes `--ext-code profiles='["e2e"]'` when deploying for e2e tests, or omits it for default deployment.

When `"e2e"` is in the profiles array, `applyE2eProfile()` patches the deployment:
- Set `imagePullPolicy: Never` (image is loaded via `kind load`)
- Add `GOCOVERDIR: "/coverage"` env var to the operator container
- Add an emptyDir volume mounted at `/coverage` in the container
- Set `terminationGracePeriodSeconds: 120` to ensure clean shutdown

The coverage volume and env are always present in e2e mode. They are harmless when the binary isn't built with `-cover` — nothing writes to the directory. This avoids a separate coverage flag; the Makefile's only decision is whether it's deploying for e2e.

### Merging strategy

`go test -coverprofile` produces text profiles. `go build -cover` + `GOCOVERDIR` produces covdata files, which are converted to text with `go tool covdata textfmt`. Both end up as text profiles in the same format (`mode: set` header + `file:line.col,line.col stmts count` lines).

Merging is a simple concatenation with a single `mode: set` header. `go tool cover` handles duplicate blocks correctly by taking the max count per block. No covdata-level merge (`go tool covdata merge -pcombine`) is needed because there is no text→covdata converter — the text format is the common denominator.

### Output layout

```
_output/
├── unit/
│   └── coverage.out          # text profile from go test -coverprofile
├── e2e/
│   ├── covdata/              # raw covmeta.* + covcounters.* files
│   └── coverage.out          # text profile converted from covdata
└── merged/
    └── coverage.out          # concatenated text profile
```

### Historical context

Coverage analysis (2026-06-23) showed integration tests (envtest) covered 318/755 statements (42.1%), while e2e covered 546/755 (72.3%). Merged was 547/755 (72.5%) — only 1 statement was exclusively covered by integration tests. The envtest integration tests were removed as redundant.

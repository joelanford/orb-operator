# Implementation Plan

1. **`.gitignore`** — add `/_output/`

2. **`.goreleaser.yml`** — add templated build flags
   - Add `GOFLAGS={{ if index .Env "GO_BUILD_FLAGS" }}{{ .Env.GO_BUILD_FLAGS }}{{ end }}` to build `env`
   - `GO_BUILD_FLAGS` is mapped to `GOFLAGS` so Go handles flag parsing natively
   - Verify normal build (no GO_BUILD_FLAGS) still works
   - Verify `GO_BUILD_FLAGS='-cover -tags=cover -covermode=atomic'` produces instrumented binary

3. **`cmd/operator/covflush.go`** — SIGUSR1 coverage flush handler
   - Build-tagged with `//go:build cover`
   - `init()` starts a goroutine: listen for SIGUSR1, call `runtime/coverage.WriteCountersDir` and `WriteMetaDir` using `GOCOVERDIR` env var
   - No-op in normal builds (file excluded by build tag)

4. **`deploy/operator.jsonnet`** — add profiles-based e2e support
   - Add `profiles` external variable (array, default `[]`): `std.extVar('profiles')` with `--ext-code` (not `--ext-str`, since it's an array)
   - When `std.member(profiles, 'e2e')`: add `GOCOVERDIR` env, hostPath volume (`/tmp/e2e-coverage` → `/coverage`), `imagePullPolicy: Never`, `terminationGracePeriodSeconds: 120`
   - Verify default rendering (empty profiles) is unchanged

5. **`Makefile` — update `run` target**
   - Add `PROFILES ?= []` and `GO_BUILD_FLAGS ?=` variables
   - Pass `GO_BUILD_FLAGS=$(GO_BUILD_FLAGS)` as env prefix to the goreleaser invocation
   - Pass `--ext-code profiles='$(PROFILES)'` to the jsonnet invocation
   - Existing `run` behavior unchanged (default empty profiles, no GO_BUILD_FLAGS)

6. **`Makefile` — update `test-unit`**
   - Create `_output/unit/` directory
   - Add `-coverprofile=_output/unit/coverage.out -coverpkg=./internal/...,./api/...` to the `go test` invocation
   - `-coverpkg` ensures coverage is measured across all operator packages, not just the package under test

7. **`Makefile` — update `test-e2e`**
   - Set `PROFILES = ["e2e"]` and `GO_BUILD_FLAGS = -cover -tags=cover -covermode=atomic` (overrides for the `run` dependency)
   - The `run` target builds via goreleaser (which picks up `GO_BUILD_FLAGS`), deploys with the e2e profile
   - Run e2e tests
   - Signal coverage flush: `kubectl exec` the operator pod and send SIGUSR1
   - Copy covdata files from Kind node: `docker cp` from the kind container's `/tmp/e2e-coverage` to `_output/e2e/covdata/`
   - Validate `covcounters.*` files exist, fail if missing
   - Convert: `go tool covdata textfmt -i=_output/e2e/covdata -o=_output/e2e/coverage.out`

8. **`Makefile` — add `test-coverage` target**
   - Depend on `test-unit` and `test-e2e`
   - Merge the two text profiles: concatenate with a single `mode: set` header into `_output/merged/coverage.out` (go's coverage tools handle duplicate blocks correctly, taking the max count)
   - Print summary: `go tool cover -func=_output/merged/coverage.out`

9. **`Makefile` — update `test-all`**
   - Change dependency from `test-unit test-e2e` to `test-coverage`

### Notes

**Why text-based merge instead of covdata merge:** `go test -coverprofile` produces text profiles, not covdata files. There is no `go tool covdata` command to convert text→covdata, only covdata→text. Since both outputs can be in text profile format, concatenating them (with a single mode header) is the simplest and correct approach — `go tool cover` handles duplicate blocks by taking the max count per block.

**Goreleaser handles everything:** The `GO_BUILD_FLAGS` env var is mapped to `GOFLAGS` in goreleaser's build subprocess environment. Go handles flag parsing natively. Goreleaser produces both the binary and docker image, so no separate Dockerfile or build path is needed for coverage.

**Why `-covermode=atomic`:** `runtime/coverage.WriteCountersDir` only supports atomic mode. The default mode (`set`) causes a warning. Atomic mode also makes the counters goroutine-safe, which is correct for a concurrent controller.

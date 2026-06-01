# Implementation Plan

1. **Initialize Go module and tool dependencies**
   - `go mod init github.com/joelanford/orb-operator`
   - Add core dependencies: controller-runtime, cobra, pflag, klog
   - Add tool dependencies: golangci-lint, gofumpt, controller-gen, goreleaser
   - `go mod tidy`

2. **Create cobra entrypoint**
   - `cmd/operator/main.go` with root command
   - Wire up klog flags, controller-runtime manager creation, manager.Start()
   - Verify it compiles with `go build ./cmd/operator/`

3. **Create placeholder packages**
   - `api/v1alpha1/doc.go` (package declaration + groupversion comment)
   - `internal/controller/doc.go`
   - `internal/handler/doc.go`
   - `internal/assertions/doc.go`
   - `test/integration/doc.go`
   - `test/e2e/doc.go`
   - `deploy/operator.jsonnet` (skeleton that renders an empty manifest set)
   - `deploy/lib/` (empty, placeholder for shared jsonnet libraries)
   - `deploy/crds/` (empty, placeholder for controller-gen CRD output)

4. **Create Makefile**
   - Targets: check, lint, lint-fix, test, build, tidy, generate, verify
   - `check` depends on `lint verify test build`
   - `verify` calls `./hack/diff.sh generate`

5. **Create hack/diff.sh**
   - Copy from library-olm (jj-aware verify script)
   - `chmod +x`

6. **Create .golangci.yml**
   - Formatters: gci (standard, default, project prefix), gofmt
   - Linters: errcheck, govet, importas, ineffassign, misspell, staticcheck, unused
   - importas alias for `api/v1alpha1`

7. **Create .goreleaser.yml**
   - Build config for `cmd/operator/`
   - Docker image config for `ghcr.io/joelanford/orb-operator`
   - Snapshot mode settings

8. **Create Dockerfile**
   - Multi-stage: Go builder image + `gcr.io/distroless/static:nonroot` runtime
   - Copy binary from goreleaser build context

9. **Create GitHub Actions CI workflows**
   - `.github/workflows/unit.yml` — `make test-unit` (PR + push to main)
   - `.github/workflows/integration.yml` — `make test-integration` (PR + push to main)
   - `.github/workflows/e2e.yml` — `make test-e2e` (PR + push to main)
   - `.github/workflows/verify.yml` — `make lint`, `make verify`, `make build` (PR + push to main)
   - `.github/workflows/image.yml` — `go tool goreleaser release --snapshot` (push to main only)

10. **Verify everything works**
    - `make check` exits 0
    - `go tool goreleaser check` validates config
    - `go tool goreleaser build --snapshot --clean` produces a binary

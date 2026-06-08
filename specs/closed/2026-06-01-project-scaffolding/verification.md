# Verification

## Implementation Correctness

- [ ] `go mod tidy` produces no changes (module is clean)
- [ ] `make check` exits 0 (lint + verify + test + build all pass)
- [ ] `make lint` exits 0 with no warnings
- [ ] `make test-unit` exits 0
- [ ] `make test-integration` exits 0
- [ ] `make test-e2e` exits 0
- [ ] `make test-all` exits 0
- [ ] `make build` exits 0
- [ ] `make verify` exits 0 (generated code is up to date)
- [ ] `go tool goreleaser check` validates the goreleaser config
- [ ] `go tool goreleaser build --snapshot --clean` produces a binary
- [ ] `go build ./cmd/operator/` compiles the entrypoint
- [ ] `go run ./cmd/operator/ --help` prints usage without errors
- [ ] All placeholder packages compile (`go build ./...`)
- [ ] hack/diff.sh is executable and works with jj

## Project Conventions

- [ ] Makefile targets match specs/tech-stack.md build commands table
- [ ] Directory structure matches specs/tech-stack.md project structure
- [ ] .golangci.yml includes all linters listed in specs/tech-stack.md
- [ ] Go module path matches specs/tech-stack.md
- [ ] No `//nolint` comments (specs/conventions.md)
- [ ] Code follows standard controller-runtime patterns (specs/mission.md design principles)

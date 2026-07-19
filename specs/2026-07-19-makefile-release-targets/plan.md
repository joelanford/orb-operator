# Implementation Plan

1. **Add `GORELEASER_ARGS` and `MANIFEST` variables to the Makefile**
   - `GORELEASER_ARGS ?= --snapshot --clean`
   - `MANIFEST ?= _output/install.json`

2. **Add `release` target**
   - Depends on `generate`.
   - Runs `go tool goreleaser release $(GORELEASER_ARGS)` with `GO_BUILD_FLAGS` forwarded.
   - Add to `.PHONY`.

3. **Add `manifest` target**
   - Creates the output directory and runs `go tool jsonnet` with `IMAGE`, `NAMESPACE`, and `PROFILES`, writing to `$(MANIFEST)`.
   - Add to `.PHONY`.

4. **Update `run` target**
   - Change dependency from `generate` to `release`.
   - Remove the inline goreleaser call.
   - Everything else stays the same.

5. **Verify**
   - `make run` works end-to-end (kind cluster + operator running).
   - `make manifest` produces the expected output.
   - `make verify` passes.

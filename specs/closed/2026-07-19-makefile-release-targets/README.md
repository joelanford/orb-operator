---
status: done
---
# Makefile: split out release and manifest targets

## Summary

The `run` target currently inlines goreleaser, jsonnet rendering, kind cluster setup, and image loading all in one recipe. Extract `release` and `manifest` into standalone targets so they can be used independently (by CI workflows, local testing, etc.), then rewrite `run` to call them.

## Design

### `release` target

Wraps `go tool goreleaser release` and forwards `GO_BUILD_FLAGS` and `GORELEASER_ARGS`:

```makefile
GORELEASER_ARGS ?= --snapshot --clean

release: generate
	GO_BUILD_FLAGS="$(GO_BUILD_FLAGS)" go tool goreleaser release $(GORELEASER_ARGS)
```

Default `GORELEASER_ARGS` is `--snapshot --clean` so `make release` behaves like the current local build. CI overrides it.

### `manifest` target

Renders `deploy/main.jsonnet` to a file:

```makefile
MANIFEST ?= _output/install.json

manifest:
	@mkdir -p $(dir $(MANIFEST))
	go tool jsonnet --ext-str image=$(IMAGE) --ext-str namespace=$(NAMESPACE) --ext-code profiles='$(PROFILES)' deploy/main.jsonnet -o $(MANIFEST)
```

### Updated `run` target

Calls `release`, then sets up kind and applies the manifest:

```makefile
run: release
	go tool kind delete cluster --name $(KIND_CLUSTER) || true
	go tool kind create cluster --name $(KIND_CLUSTER)
	go tool kind load docker-image $(IMAGE)-$$(go env GOARCH) --name $(KIND_CLUSTER)
	go tool jsonnet --ext-str image=$(IMAGE)-$$(go env GOARCH) --ext-str namespace=$(NAMESPACE) --ext-code profiles='$(PROFILES)' deploy/main.jsonnet | kubectl apply -f -
	kubectl -n $(NAMESPACE) rollout status deployment/orb-operator --timeout=60s
```

Note: `run` still inlines the jsonnet call (with the arch-suffixed image) rather than calling `manifest`, because the local kind workflow needs the `-$(go env GOARCH)` suffix that goreleaser snapshot appends. The `manifest` target is for producing the canonical release artifact with a clean image ref.

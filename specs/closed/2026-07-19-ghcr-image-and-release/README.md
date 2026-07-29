---
status: done
---
# GHCR Image Push and GitHub Releases

## Summary

Replace the current `image.yml` workflow with a single `release.yml` workflow that handles all ref types: PRs (snapshot build), main branch pushes (push `main`-tagged image), and version tag pushes (push versioned image + create GitHub release with install manifest).

## Design

### Makefile variable changes

Split `IMAGE` into derived components:

```makefile
IMAGE_REPO ?= ghcr.io/joelanford/orb-operator
IMAGE_TAG ?= dev
IMAGE = $(IMAGE_REPO):$(IMAGE_TAG)
```

Pass `IMAGE_REPO` and `IMAGE_TAG` to goreleaser so it can reference them in its config. CI sets `IMAGE_TAG` and everything else flows.

### Goreleaser configuration changes

- `dockers_v2.images`: use `{{ .Env.IMAGE_REPO }}` instead of hardcoded `ghcr.io/joelanford/orb-operator`.
- `dockers_v2.tags`: use `{{ .Env.IMAGE_TAG }}` instead of `{{ .Version }}`.
- `release.disable`: `'{{ ne .Env.ENABLE_RELEASE_PIPELINE "true" }}'` instead of `true`.
- `changelog.disable`: `'{{ ne .Env.ENABLE_RELEASE_PIPELINE "true" }}'` instead of `true`.
- Add `extra_files` to attach `_output/install.json` to the GitHub release.
- Add a release header template with a copy/pasteable `kubectl apply -f` command using the full GitHub release download URL.

### Single workflow, env-var-driven

A single `release.yml` workflow replaces `image.yml`. Triggered on PRs, pushes to `main`, and `vX.Y.Z` tag pushes.

Permissions: `packages: write` + `contents: write`. GitHub automatically downgrades fork PR tokens to read-only.

A setup step sets env vars based on the ref type:

| Ref | `IMAGE_TAG` | `GORELEASER_ARGS` | `ENABLE_RELEASE_PIPELINE` |
|-----|-------------|-------------------|--------------------------|
| PR | `pr-<number>` | `--snapshot --clean` | (unset) |
| `main` | `main` | `--clean --skip=validate` | (unset) |
| `vX.Y.Z` tag | `<tag>` | `--clean` | `true` |

Steps:
1. Checkout (with `fetch-depth: 0`)
2. Setup Go
3. Docker login to GHCR (skip on PRs)
4. Set env vars based on ref
5. Run `make manifest release` (manifest renders the install artifact, release runs goreleaser)

### Release manifest

For tag pushes, `make manifest` renders `deploy/main.jsonnet` with the tagged image, producing `_output/install.json`. Goreleaser attaches it to the GitHub release via `extra_files`. The manifest is a plain Kubernetes List (~16 objects, under 100KB).

### Local dev

`make run` is unaffected. Default `IMAGE_TAG=dev` + `GORELEASER_ARGS=--snapshot --clean` = same behavior as before. The `run` target's `IMAGE := $(IMAGE)-$(shell go env GOARCH)` appends the arch suffix that goreleaser snapshot adds.

### Nice to have (future)

Bootstrap the installation into a COD so upgrades are handled via COD spec patches. Deferred because the COD CRD does not exist prior to first install.

---
status: idea
---
# GHCR Image Push and GitHub Releases

Push container images to GHCR and create GitHub releases on version tags. Split into separate work items:

1. **[Makefile refactor](../2026-07-19-makefile-release-targets/)** - add `manifest` and `release` targets, update `run` to use them.
2. **Push images** - `release.yml` workflow + goreleaser config to push multi-arch images on main and tag pushes.
3. **Release artifacts** - render and attach install manifest to GitHub release, release notes with `kubectl apply` command.

## Design notes

### Single workflow, env-var-driven

Following the pattern from [operator-controller](https://github.com/operator-framework/operator-controller/blob/main/.github/workflows/release.yaml), a single workflow handles all ref types. A setup step sets env vars based on the ref:

- `IMAGE_TAG`: `main` for main branch, the tag name for `vX.Y.Z` tags.
- `GORELEASER_ARGS`: controls flags passed to goreleaser.
- `ENABLE_RELEASE_PIPELINE`: `true` only for tag pushes, used in goreleaser config to conditionally enable GitHub release creation and changelog.

### Goreleaser configuration

The existing `.goreleaser.yml` uses `dockers_v2`, which natively builds and pushes multi-arch images via `docker buildx build --push` during a non-snapshot release. Stop using `--snapshot` for main and tag pushes so goreleaser pushes the image.

- Make `release.disable` and `changelog.disable` conditional on an env var instead of hardcoded `true`.
- Use `--skip=validate` for main branch pushes (no git tag to validate against).
- The image tag comes from an `IMAGE_TAG` env var referenced in the `dockers_v2.tags` config.

### Release manifest

A plain Kubernetes List rendered from `deploy/main.jsonnet`. ~16 objects totaling under 100KB (CRDs are ~80KB), so no COSLs are needed. Attached to the GitHub release via goreleaser `extra_files`.

### Nice to have (future)

Bootstrap the installation into a COD so upgrades are handled via COD spec patches. Deferred because the COD CRD does not exist prior to first install.

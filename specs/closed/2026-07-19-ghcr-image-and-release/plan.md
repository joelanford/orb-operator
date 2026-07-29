# Implementation Plan

1. **Update Makefile variables**
   - Replace `IMAGE ?= ghcr.io/joelanford/orb-operator:dev` with `IMAGE_REPO`, `IMAGE_TAG`, and derived `IMAGE`.
   - Pass `IMAGE_REPO` and `IMAGE_TAG` as env vars to goreleaser in the `release` target.

2. **Update `.goreleaser.yml`**
   - Change `dockers_v2.images` from hardcoded `ghcr.io/joelanford/orb-operator` to `{{ .Env.IMAGE_REPO }}`.
   - Change `dockers_v2.tags` from `{{ .Version }}` to `{{ .Env.IMAGE_TAG }}`.
   - Change `release.disable` from `true` to `'{{ ne .Env.ENABLE_RELEASE_PIPELINE "true" }}'`.
   - Change `changelog.disable` from `true` to `'{{ ne .Env.ENABLE_RELEASE_PIPELINE "true" }}'`.
   - Add `extra_files` for `_output/install.json`.
   - Add a `release.header` template with the `kubectl apply -f` command using the templated download URL.

3. **Replace `.github/workflows/image.yml` with `release.yml`**
   - Triggered on `pull_request`, `push` to `main`, and `push` of `v*.*.*` tags.
   - Permissions: `packages: write`, `contents: write`.
   - Setup step sets `IMAGE_TAG`, `GORELEASER_ARGS`, and `ENABLE_RELEASE_PIPELINE` based on ref type.
   - Docker login to GHCR (skipped on PRs).
   - Run `make manifest release` with `GITHUB_TOKEN` in env.

4. **Verify**
   - `make verify` passes.
   - `make release` runs a local snapshot build.
   - `make manifest` produces correct output.
   - `make run` works end-to-end.

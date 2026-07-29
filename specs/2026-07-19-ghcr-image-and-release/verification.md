# Verification

## Implementation Correctness

- [ ] `make verify` passes (includes `goreleaser check`).
- [ ] `make release` runs a local snapshot build without pushing.
- [ ] `make manifest` produces valid JSON at `_output/install.json`.
- [ ] `make manifest IMAGE_TAG=v1.0.0` produces a manifest with `ghcr.io/joelanford/orb-operator:v1.0.0` as the image ref.
- [ ] `make run` works end-to-end (kind cluster + operator running).
- [ ] `image.yml` is deleted and replaced by `release.yml`.
- [ ] `release.yml` sets correct env vars for each ref type (PR, main, tag).
- [ ] `release.yml` skips docker login on PRs.
- [ ] `release.yml` runs `make manifest release` as a single step.
- [ ] Goreleaser config conditionally enables release and changelog based on `ENABLE_RELEASE_PIPELINE`.
- [ ] Goreleaser `extra_files` references `_output/install.json`.
- [ ] Release header template includes a `kubectl apply -f` command with the correct download URL.

## Project Conventions

- [ ] Makefile variable naming follows existing conventions (`UPPER_SNAKE_CASE`, `?=` defaults).
- [ ] Workflow follows patterns from other `.github/workflows/` files.
- [ ] No new tool dependencies introduced.
- [ ] `specs/tech-stack.md` updated to reflect the new workflow.

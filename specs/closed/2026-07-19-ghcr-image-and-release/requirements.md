# Requirements

- On push to `main`, push a multi-arch image to `ghcr.io/joelanford/orb-operator:main`.
- On push of a `vX.Y.Z` tag, push a multi-arch image to `ghcr.io/joelanford/orb-operator:<tag>`.
- On push of a `vX.Y.Z` tag, create a GitHub release with auto-generated changelog.
- The GitHub release includes `_output/install.json` as a release artifact.
- The release notes include a copy/pasteable `kubectl apply -f <url>` command.
- On PRs, run a snapshot build (no push, no release).
- `make run` and `make test-e2e` continue to work unchanged.
- `make verify` continues to pass.

## Acceptance Criteria

- Pushing to `main` results in a multi-arch image at `ghcr.io/joelanford/orb-operator:main`.
- Pushing a `vX.Y.Z` tag results in a multi-arch image at `ghcr.io/joelanford/orb-operator:vX.Y.Z`.
- Pushing a `vX.Y.Z` tag creates a GitHub release with changelog, install manifest, and `kubectl apply` command.
- PR builds run goreleaser in snapshot mode without pushing or creating releases.
- `make release` locally still runs a snapshot build (no push).
- `make manifest IMAGE_TAG=v1.0.0` produces a manifest with the correct image reference.
- `make run` works end-to-end.
- `make verify` passes.

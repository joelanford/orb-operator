# Requirements

- A `release` target wraps goreleaser with configurable args (defaulting to `--snapshot --clean`).
- A `manifest` target renders `deploy/main.jsonnet` to a file with configurable image, namespace, and profiles.
- The `run` target calls `release` instead of invoking goreleaser directly.
- `make run` (including `make test-e2e` which depends on `run`) continues to work identically.
- `make verify` continues to pass.

## Acceptance Criteria

- `make release` builds local snapshot images (same as current behavior).
- `make release GORELEASER_ARGS="--clean"` runs a non-snapshot release (for CI use).
- `make manifest` produces a valid JSON file at `_output/install.json` containing all objects from `deploy/main.jsonnet`.
- `make manifest IMAGE=ghcr.io/joelanford/orb-operator:v1.0.0` produces a manifest with the correct image reference.
- `make run` and `make test-e2e` work unchanged.

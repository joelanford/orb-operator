# Verification

## Implementation Correctness

- [ ] `make release` builds local snapshot images without pushing.
- [ ] `make manifest` writes a valid JSON file to `_output/install.json`.
- [ ] `make manifest IMAGE=ghcr.io/joelanford/orb-operator:v1.0.0` uses the correct image ref in the output.
- [ ] `make run` works end-to-end (kind cluster created, operator deployed and healthy).
- [ ] `make test-e2e` still passes (it depends on `run`).
- [ ] `make verify` passes.

## Project Conventions

- [ ] New targets are declared in `.PHONY`.
- [ ] Variable naming follows existing conventions (`UPPER_SNAKE_CASE`, `?=` defaults).
- [ ] No new tool dependencies introduced.

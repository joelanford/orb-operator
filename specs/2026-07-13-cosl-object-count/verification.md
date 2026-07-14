# Verification

## Implementation Correctness

- [ ] COSL `objectCount` equals `len(objects)` after creation.
- [ ] MutatingAdmissionPolicy is deployed with the operator.
- [ ] COSL printer columns: NAME, OBJECTS, AGE.

## Project Conventions

- [ ] No `//nolint` comments added.
- [ ] Code formatted with gofumpt.
- [ ] `make lint` passes.
- [ ] `make verify` passes.
- [ ] Unit tests use testify assert/require.
- [ ] New fields have godoc comments following existing conventions.
- [ ] `make test-unit` passes.
- [ ] `make test-e2e` passes.

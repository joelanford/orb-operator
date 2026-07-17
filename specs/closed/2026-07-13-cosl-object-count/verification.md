# Verification

## Implementation Correctness

- [x] COSL `count` field exists as int32 on `ClusterObjectSlice`.
- [x] CEL XValidation rule enforces `count == size(objects)` on the CRD.
- [x] MAP sets `count` automatically when not provided by caller.
- [x] MAP overwrites incorrect `count` with the correct value.
- [x] MAP (`cosl-set-count`) is defined in `api.libsonnet`.
- [x] MAPB (`cosl-set-count`) is defined in `api.libsonnet`.
- [x] Both MAP and MAPB appear in rendered deploy output.
- [x] COSL printer columns: NAME, OBJECTS, AGE.
- [x] `InstallAPI` waits for MAP dispatcher sync via dry-run CREATE.

## Project Conventions

- [x] No `//nolint` comments added.
- [x] Code formatted with gofumpt.
- [x] `make lint` passes.
- [x] `make verify` passes.
- [x] New fields have godoc comments following existing conventions.
- [x] `make test-unit` passes.
- [x] `make test-e2e` passes.

# Requirements

- Every occurrence of `ClusterObjectSet` (the parent type) is renamed to `ClusterObjectDeployment` across Go source, generated code, markers, tests, examples, and docs.
- Every occurrence of `ClusterObjectSetRevision` is renamed to `ClusterObjectSet` across the same surfaces.
- The `ClusterObjectSlice` type and all references to it are unchanged.
- Kubebuilder short names change: `cos` → `cod`, `cosr` → `cos`.
- File names, directory names, and variable prefixes follow the abbreviation mapping (`cos_` → `cod_`, `cosr_` → `cos_`).
- Generated artifacts (deepcopy, CRDs, apply configurations) are regenerated after each pass.
- `specs/mission.md`, `specs/tech-stack.md`, and open (non-closed) specs are updated to use the new names.
- Closed specs under `specs/closed/` are left untouched.
- Each pass (COS→COD, then COSR→COS) produces a compilable codebase that passes `make verify` and all tests.

## Acceptance Criteria

- `make verify` passes after each commit.
- `make test-all` passes after each commit.
- `grep -r 'ClusterObjectSetRevision' --include='*.go'` returns zero hits after both passes.
- `grep -r 'shortName=cosr' --include='*.go'` returns zero hits after both passes.
- The only remaining `ClusterObjectSet` references in Go code are for the new type (formerly ClusterObjectSetRevision).
- CRD YAML files under `deploy/crds/` reflect the new type names.
- No `ClusterObjectSetRevision` references remain in `.feature` files, examples, or project docs.

# Verification

## Implementation Correctness

### After Pass 1 (COS → COD)

- [ ] `make verify` passes (lint, build, generate diff, goreleaser check)
- [ ] `make test-unit` passes
- [ ] `make test-e2e` passes
- [ ] `grep -rn 'type ClusterObjectSet ' --include='*.go' api/` returns zero hits (the type no longer exists under that name)
- [ ] `grep -rn 'shortName=cos' --include='*.go'` returns zero hits
- [ ] `grep -rn 'COSReconciler' --include='*.go'` returns zero hits (renamed to CODReconciler)
- [ ] No file named `cos_controller.go` or `types_clusterobjectset.go` exists (those names are freed for Pass 2)
- [ ] CRD file `deploy/crds/orb.operatorframework.io_clusterobjectdeployments.yaml` exists
- [ ] Old CRD file `deploy/crds/orb.operatorframework.io_clusterobjectsets.yaml` does not exist
- [ ] Feature files are renamed from `cos_*` to `cod_*`
- [ ] Feature file content uses `COD` not `COS` for the parent type

### After Pass 2 (COSR → COS)

- [ ] `make verify` passes
- [ ] `make test-unit` passes
- [ ] `make test-e2e` passes
- [ ] `grep -rn 'ClusterObjectSetRevision' --include='*.go'` returns zero hits
- [ ] `grep -rn 'shortName=cosr' --include='*.go'` returns zero hits
- [ ] `grep -rn 'COSRReconciler' --include='*.go'` returns zero hits
- [ ] No file named `cosr_controller.go` or `types_clusterobjectsetrevision.go` exists
- [ ] CRD file `deploy/crds/orb.operatorframework.io_clusterobjectsets.yaml` exists (now for the former COSR type)
- [ ] Old CRD file `deploy/crds/orb.operatorframework.io_clusterobjectsetrevisions.yaml` does not exist
- [ ] Feature files are renamed from `cosr_*` to `cos_*`
- [ ] `grep -rn 'COSR' test/e2e/features/` returns zero hits
- [ ] `grep -rn 'ClusterObjectSetRevision' deploy/ specs/mission.md specs/tech-stack.md` returns zero hits

## Project Conventions

- [ ] Commit messages follow conventional commits: `refactor: rename ...`
- [ ] One logical change per commit (Pass 1 and Pass 2 are separate commits)
- [ ] No `//nolint` comments added
- [ ] Generated code is regenerated, not hand-edited
- [ ] `specs/mission.md` accurately reflects the new type names
- [ ] `specs/tech-stack.md` project structure section uses new file names
- [ ] `ClusterObjectSlice` references are unchanged throughout

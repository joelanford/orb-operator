# Verification

## Implementation Correctness

- [ ] `templateHash` function exists with unit tests for stability and sensitivity
- [ ] COSR controller uses `Watches(mapToLatestInChain)` — no `For(&COSR{})`, no `mapToGroupMembers`
- [ ] COSR `reconcile()` partitions group members by controller owner name and reconciles the full chain in one pass
- [ ] COSR controller never writes to `spec.lifecycleState` — grep confirms zero writes
- [ ] Every error return in the COSR controller sets a status condition before returning
- [ ] COS controller creates COSRs with `orb.operatorframework.io/template-hash` label
- [ ] COS uses hash comparison, not deep comparison, for template change detection
- [ ] COS adopts unowned COSRs by setting controller ownerRef
- [ ] COS sets `lifecycleState: Archived` on older Active owned COSRs when latest is Available
- [ ] COS pruning only deletes Archived COSRs with no finalizer
- [ ] COS revision numbering uses max across ALL group COSRs, not just owned
- [ ] COS status derived from Active owned revisions only (0→Unavailable, 1→mirror, >1→Progressing)

## Test Coverage

- [ ] Standalone COSR scenarios expect `Superseded` (not `Archived`) when a newer revision exists
- [ ] Archival+cleanup scenario moved to COS tests (not in COSR tests)
- [ ] Superseded COSR retention scenario passes — objects not in the latest revision remain with ownerRef to predecessor
- [ ] COS lifecycle scenario passes — template change → new revision → archival → teardown deletes old objects
- [ ] COS adoption scenario passes — unowned COSR gets adopted

## Project Conventions

- [ ] `make verify` passes (lint, build, generate check, goreleaser check)
- [ ] `make test-unit` passes
- [ ] `make test-e2e` passes
- [ ] No `//nolint` comments added
- [ ] Code follows standard controller-runtime patterns (per specs/mission.md)
- [ ] No new dependencies introduced beyond what's in specs/tech-stack.md
- [ ] Hash function uses only stdlib (`crypto/sha256`, `encoding/json`, `fmt`)

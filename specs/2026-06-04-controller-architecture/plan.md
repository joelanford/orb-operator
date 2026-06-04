# Implementation Plan

## 1. Add template hash utility

Add a `templateHash(ClusterObjectSetTemplate) string` function (in `internal/controller/`) that JSON-serializes the template and returns the first 8 hex characters of its SHA-256 hash. Add the label constant `orb.operatorframework.io/template-hash`. Write a unit test confirming stability (same input → same output) and sensitivity (different input → different output).

## 2. Refactor COSR controller to chain-based reconciliation

### 2a. Replace event mapping

Remove the `For(&COSR{})` registration and the `Watches(&COSR{}, mapToGroupMembers)` mapper. Replace with a single `Watches(&COSR{}, mapToLatestInChain)` that:
- Lists all COSRs in the group (field index)
- Partitions by controller owner name
- Returns a single reconcile request for the latest (highest revision) Active COSR in the same chain as the triggering COSR

Keep the existing `WatchesRawSource` for the boxcutter access manager.

### 2b. Restructure reconcile to chain-level

Rewrite `reconcile()` to:
1. List all COSRs in the group
2. Partition by `controllerOwnerName(cosr)` — extract controller owner name from the COSR's owner references
3. Find chain members for the reconciled COSR
4. For the latest active COSR: call `engine.Reconcile` with all chain predecessors as previous owners
5. For each non-latest active predecessor: set `Available=False, Reason=Superseded` and call `Status().Update` on that predecessor directly (not via the top-level DeepEqual guard, which only covers the reconciled COSR)
6. For each archived member: run `engine.Teardown`, set `Available=False, Reason=Archived`, call `Status().Update` on that member, and remove finalizer when complete

### 2c. Remove spec mutation

Delete all code paths where the COSR controller sets `spec.lifecycleState`. The COSR controller only reads lifecycle state, never writes it.

### 2d. Add error conditions on all paths

Audit every error return in the reconciler. Before returning an error, set a condition with the error details. The existing DeepEqual-guarded `Status().Update` in `Reconcile` handles persistence.

## 3. Refactor COS controller

### 3a. Template hash comparison

Replace `templateEqual()` with hash-based comparison:
- In `buildCOSRFromTemplate`, add the computed template hash as a label on the new COSR
- In the reconcile loop, compare the hash label on the latest owned COSR against the hash of the current template

### 3b. Adoption logic

After listing COSRs in the group, iterate over unowned COSRs (no controller ownerRef) and set a controller ownerRef pointing to this COS. Use `controllerutil.SetControllerReference`. Re-list or append to the owned set after adoption.

### 3c. COS-driven archival

When the latest owned COSR is Available, iterate over all older Active owned COSRs and set `spec.lifecycleState = Archived`. Update each via `r.client.Update`.

### 3d. Fix pruning

Only delete Archived COSRs that have completed teardown (no finalizer present). Log delete errors, don't return them.

### 3e. Fix revision numbering

Change revision number computation from `latestCOSR.Spec.Revision + 1` to `max(revision across ALL COSRs in the group) + 1`. This uses the full group list (not just owned).

### 3f. Fix status derivation

Status uses Active (non-Archived) owned revisions only. Remove the current code that looks at the latest COSR regardless of lifecycle state. Implement the three-branch logic: 0 active → Unavailable, 1 active → mirror, >1 active → Progressing.

## 4. Update e2e tests

### 4a. Fix standalone COSR expectations

In `cosr_revision_transitions.feature`:
- "Non-contiguous revision numbers work correctly": change expected reason from `"Archived"` to `"Superseded"`
- "Old revision is archived after new revision succeeds": change expected reason from `"Archived"` to `"Superseded"` and rename the scenario

### 4b. Remove moved scenario

Remove "Revision transition deletes old objects, updates shared objects, and creates new objects" from `cosr_revision_transitions.feature`.

### 4c. Add superseded retention test

Add a scenario to `cosr_revision_transitions.feature` verifying that a superseded standalone COSR retains objects not adopted by the latest revision. This may require new step definitions for checking owner references on specific objects.

### 4d. Add COS lifecycle test

Add a scenario to `cos_revision_management.feature` testing the full COS-driven lifecycle: template change creates a new revision, old revision gets archived by COS, teardown cleans up old objects.

### 4e. Add COS adoption test

Add a scenario to `cos_ownership.feature` (or a new feature file) verifying that a COS adopts an unowned COSR in its group.

## 5. Verify

Run `make verify`, `make test-unit`, and `make test-e2e`. Fix any failures.

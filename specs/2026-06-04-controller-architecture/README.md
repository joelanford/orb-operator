---
status: in-progress
---
# Controller Architecture — Chain Model & COS-Driven Lifecycle

## Summary

Refactor the COS and COSR controllers around a chain-based reconciliation model. A chain is defined by `(group, controllerOwnerName)`. The COSR controller reconciles entire chains top-down from the latest revision. The COS controller handles COSR lifecycle (creation, adoption, archival, pruning) and derives status from active owned revisions. Template change detection uses a deterministic hash label instead of deep comparison.

This spec supersedes the `2026-06-01-cos-controller` idea and revises controller behavior established in `2026-06-01-cosr-revision-transitions`.

## Motivation

A code review of the initial COS controller and supporting COSR controller changes identified several architectural issues:

1. **Group and ownerRef compete.** The COSR controller uses `group` for supersession (all COSRs in the group compete) while the COS controller uses `ownerRef` for management (only owned COSRs). When these disagree — e.g., an unowned COSR in a COS-managed group — controllers fight.

2. **COSR controller mutates its own spec.** `reconcileSuperseded` sets `spec.lifecycleState` to Archived on the COSR being reconciled. This is a Kubernetes anti-pattern (reconcilers should update status, not spec).

3. **Template comparison is fragile.** Deep-comparing labels/annotations between the COS template and the COSR means any external metadata addition (admission webhook, another controller) triggers infinite revision creation.

4. **Status derived from wrong set of COSRs.** `setStatus` picks the highest-revision COSR including Archived ones, misreporting the COS as Unavailable when a lower-revision Active COSR is Available.

5. **Per-COSR reconciliation causes races.** Each COSR independently determines its role (latest vs. superseded), leading to concurrent reconciliations of the same chain with no coordination.

6. **Missing error conditions.** Several error paths return errors from `Reconcile` without setting status conditions, so failures are invisible in COSR status.

## Design

### Chain model

A **chain** is defined by `(group, controllerOwnerName)`. All COSRs sharing the same group and the same controller owner name form a single chain. `nil` controller owner gives an empty owner name — all standalone COSRs (no controller ownerRef) in a group form one chain together.

Within a chain, the COSR with the highest revision is the **latest**; all others are **predecessors**.

Using owner name (not UID) means a recreated COS with the same name inherits the existing chain, which is the desired behavior for re-adoption.

### COSR controller changes

The COSR controller reconciles chains, not individual COSRs.

**Event mapping:** Replace the current `For(&COSR{})` + `Watches(mapToGroupMembers)` setup. Use `Watches(&COSR{}, mapToLatestInChain)` where all COSR events map to the latest COSR in the chain. The workqueue deduplicates by key — one reconciliation per chain regardless of how many member events fired. Keep the existing `WatchesRawSource` for the boxcutter access manager.

**Chain reconciliation:** On each `Reconcile(latest)`:

1. List all COSRs in the group (field index).
2. Partition by controller owner name to find the chain members for the reconciled COSR's chain.
3. **Latest** (highest revision, Active): call `engine.Reconcile` with all predecessors in the chain as previous owners for resource handoff.
4. **Predecessors** (non-latest, Active): set `Available=False, Reason=Superseded`. These COSRs retain managed objects not adopted by the latest — objects still exist with ownerRefs pointing to the predecessor.
5. **Archived members**: run `engine.Teardown` (delete managed objects, remove finalizer when complete). Set `Available=False, Reason=Archived`.

**The COSR controller never mutates `lifecycleState`.** It reconciles based on the current state but doesn't transition between states. Lifecycle transitions are the responsibility of the parent (COS controller for managed COSRs, user for standalone COSRs).

**Error conditions.** Every error path must set a condition reflecting the failure before returning. The existing centralized status-write in `Reconcile` (DeepEqual check before `Status().Update`) persists the condition.

### COS controller changes

**Template hash comparison:** Compute a deterministic hash of `ClusterObjectSetTemplate` (both metadata and spec), store as a label on the COSR (e.g., `orb.operatorframework.io/template-hash`). Template change detection becomes a string comparison on the latest owned COSR's hash label vs. the computed hash. This eliminates fragility to external label/annotation additions.

The hash function: JSON-serialize the `ClusterObjectSetTemplate` struct, SHA-256 hash, truncate to 8 hex characters. Use `encoding/json` with sorted keys (Go maps sort by default in `encoding/json`).

**Adoption:** When listing COSRs in the group, check `metav1.GetControllerOf()` for each:
- `nil` → adopt (set controller ownerRef pointing to this COS)
- This COS's UID → already owned
- Different UID → skip (belongs to another controller)

This follows the same adoption logic as the Deployment controller with ReplicaSets.

**COS-driven archival:** When the latest owned COSR is Available, set `spec.lifecycleState = Archived` on all older Active owned COSRs. `Owns()` guarantees the informer cache is at least as fresh as the triggering event. Stale cache only causes delayed archival (safe direction), never premature archival.

**Pruning:** Delete Archived COSRs beyond `revisionHistoryLimit` (lowest revision first). Must wait until teardown is complete before deleting (check that the COSR has no finalizer). Log delete errors but don't return them (pruning is best-effort).

**Revision numbering:** `max(revision across ALL COSRs in the group) + 1`. Uses all COSRs in the group (not just the COS's own chain) to avoid COSR name collisions (`{group}-{revision}`) across chains sharing a group.

**Status:** Derived from Active (non-Archived) owned revisions only:
- `len == 0` → `Available=False, Reason=Unavailable`
- `len == 1` → mirror the single active revision's Available condition
- `len > 1` → `Available=Unknown, Reason=Progressing`

### Cross-chain conflicts

Two chains in the same group (different controller owners) don't interfere with each other's supersession logic (scoped by controller owner). If their COSRs manage overlapping resources, collision protection handles the resource-level conflict. Within a chain, collision protection is bypassed for predecessors, since the latest legitimately adopts from them.

### Standalone COSRs

Standalone COSRs (no COS, no controller owner) form chains scoped by `(group, "")`. The COSR controller reconciles them the same way — latest reconciles, others yield. But since there's no COS controller to manage lifecycle, standalone COSRs are never automatically archived. The user must manually set `lifecycleState: Archived` to trigger teardown.

## Test changes

### Move archival + cleanup scenario to COS tests

The COSR `cosr_revision_transitions.feature` scenario "Revision transition deletes old objects, updates shared objects, and creates new objects" tests the full lifecycle (supersede → archive → teardown deletes old objects). With COS-driven archival, this only works through a COS. Rewrite as a COS scenario in `cos_revision_management.feature`.

### Change standalone COSR expectations from Archived to Superseded

In `cosr_revision_transitions.feature`:
- "Non-contiguous revision numbers work correctly" — change `reason "Archived"` to `reason "Superseded"`
- "Old revision is archived after new revision succeeds" — change `reason "Archived"` to `reason "Superseded"` (and rename the scenario to match new behavior)

### Add test: superseded COSR retains non-transitioned objects

New scenario in `cosr_revision_transitions.feature` verifying that a superseded standalone COSR retains objects not adopted by the latest revision.

### Add COS test: full revision transition with archival and cleanup

New scenario in `cos_revision_management.feature` testing COS-driven lifecycle: template change → new revision → archival → teardown of old objects.

### Add COS test: adoption of unowned COSR

New scenario in `cos_ownership.feature` verifying that a COS adopts an unowned COSR in its group by adding a controller ownerRef.

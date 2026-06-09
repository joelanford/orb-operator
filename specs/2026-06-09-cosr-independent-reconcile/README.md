---
status: ready
---
# Independent COSR Reconciliation

## Summary

Refactor the COSR controller so each ClusterObjectSetRevision reconciles itself independently, rather than funneling all events through the highest revision in the chain. This ensures predecessors actively manage objects they uniquely own during revision transitions, and is a prerequisite for the COSR phase status work item.

## Use Cases

### 1. Predecessor objects stay managed during transitions

**Problem:** Today, predecessors are marked `Superseded` with no boxcutter reconciliation. Objects unique to the predecessor (not shared with the new revision) are not actively managed during the transition period. If they drift or fail probes, nothing detects or fixes it.

**Need:** Predecessors must actively reconcile objects they own, with sibling awareness so shared objects are handled correctly by the highest revision.

### 2. Foundation for phase status

**Problem:** The COSR phase status work item needs meaningful boxcutter results from all active COSRs to report per-phase status. Today, only the highest revision produces boxcutter results.

**Need:** Every active COSR produces boxcutter reconcile results that can be mapped to phase status.

## Design

### Event mapping changes

**Current:**
- COSR event → `mapToHighestRevInChain` → enqueue highest revision
- Managed object event → `managedObjectToHighestRevInChain` → enqueue highest revision

**New:**
- COSR event → enqueue self
- Managed object event → enqueue owning COSR (via controller owner ref)

### Reconciliation changes

Each COSR determines its role by listing chain members (same as today), then reconciles based on its own role:

| Role | Current behavior | New behavior |
|---|---|---|
| **Latest active** | Full boxcutter reconcile with predecessors as siblings | Same |
| **Predecessor** | Set `Superseded` condition, no boxcutter call | Full boxcutter reconcile with latest + other predecessors as siblings |
| **Archived** | Teardown | Same |
| **Deleted** | Teardown + release finalizer | Same |

The only behavioral change is predecessors getting a real boxcutter reconcile with sibling awareness.

### Predecessor reconciliation details

When a predecessor reconciles:
1. List chain members, identify self as a predecessor (a higher active revision exists)
2. Build a boxcutter revision with siblings (the latest active + other predecessors)
3. Call `engine.Reconcile()` — boxcutter handles shared objects via sibling ownership (hands them off to the highest revision) and actively reconciles objects unique to this predecessor
4. Set the `Available` condition based on the reconcile result. The condition reason remains `Superseded` but status reflects actual object health.

### What stays the same

- Chain membership logic (`buildChain`, `listGroupMembers`, `filterByControllerOwner`)
- Finalizer management (`ensureFinalizers`)
- Archived/deleted COSR handling
- COS controller behavior (archival decisions, status mirroring)
- The COS controller continues to decide when to archive predecessors

### Known edge case

When the new revision has a completely disjoint set of objects from the predecessor, nothing triggers the predecessor to reconcile after the new revision is created (no shared objects means no ownership transfer events). The predecessor will reconcile on its next natural event (e.g., a managed object watch event, or controller resync). An e2e test must cover this case. If the delay proves problematic, the event mapping can be extended to also enqueue lower-revision active siblings.

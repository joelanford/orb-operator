# Requirements

- Each COSR must reconcile itself independently when enqueued, not delegate to the highest revision.
- COSR events must enqueue the COSR itself, not the highest revision in the chain.
- Managed object events must enqueue the owning COSR directly (via controller owner ref).
- Predecessors must run a full boxcutter reconcile with sibling awareness, actively managing objects they own.
- Predecessors must pass the latest active COSR and other predecessors as siblings to boxcutter.
- The `Available` condition reason for predecessors remains `Superseded`.
- Archived COSR teardown behavior must not change.
- Deleted COSR teardown and finalizer release behavior must not change.
- The COS controller must not change — it continues to decide when to archive predecessors.
- Chain membership logic must not change — each COSR still lists its chain to determine its role.

## Acceptance Criteria

- A predecessor COSR actively reconciles objects it uniquely owns (drift is corrected).
- A predecessor COSR with shared objects defers those objects to the highest revision via boxcutter's sibling mechanism.
- A predecessor COSR's `Available` condition reflects actual object health, not just `Superseded=False`.
- Existing e2e scenarios continue to pass with no changes.
- New e2e scenario: two revisions with fully disjoint object sets — predecessor's unique objects remain managed and healthy during the transition.
- New e2e scenario: predecessor with a unique object that drifts — drift is corrected by the predecessor's reconcile.

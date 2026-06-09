# Implementation Plan

1. **Simplify event mapping** — replace `mapToHighestRevInChain` with a direct self-enqueue (return the COSR's own name as the reconcile request). Replace `managedObjectToHighestRevInChain` with a direct lookup of the owning COSR via controller owner ref (no chain traversal).

2. **Refactor reconcileChain into per-role dispatch** — the `Reconcile` entry point lists chain members, determines the reconciled COSR's role (latest, predecessor, archived, deleted), and dispatches to the appropriate handler. Remove `reconcileActiveMembers` which orchestrated the chain from the latest's perspective.

3. **Implement predecessor reconciliation** — replace `doReconcilePredecessor` (which just sets `Superseded`) with a full boxcutter reconcile. Build the revision with siblings (latest active + other predecessors). Set the `Available` condition with reason `Superseded` and status reflecting boxcutter's result.

4. **Update existing tests** — adjust integration tests that assert on reconciliation ordering or event routing. The behavioral change is minimal (predecessors are now actively reconciled), so most tests should pass without changes.

5. **Add e2e scenarios** — add scenarios for: (a) disjoint object sets across revisions — predecessor's unique objects remain managed during transition, (b) predecessor object drift — drift is corrected by the predecessor's reconcile.

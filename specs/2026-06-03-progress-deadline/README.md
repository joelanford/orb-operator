---
status: idea
---
# COD Progress Deadline

Add `progressDeadlineMinutes` to the ClusterObjectDeployment spec. This field sets a deadline for rollout progress — if a COS doesn't make progress within this window, the COD controller marks the rollout as failed. Deferred from the initial COD API types work item to keep the first iteration focused on core stamping behavior.

## Prerequisites

- **COS phase status** — progress deadline semantics should mirror Kubernetes Deployments, which track incremental progress (replica count changes) via a `LastUpdateTime` on the `Progressing` condition. Since `metav1.Condition` lacks `LastUpdateTime`, the COD needs per-phase progress information in the COS status to serve as the progress signal. Without it, the only timer baseline is COS `creationTimestamp`, which doesn't capture incremental progress and can't support Deployment-style deadline resets.

## Design Considerations

- **COD-level phase status aggregation** — each COS reports `observedPhases` only for the objects it owns. During partial supersession (a newer COS adopts some but not all objects from a predecessor), the predecessor's phase status is accurate for its remaining objects but doesn't show the full picture. The COD should union `observedPhases` across all active COSs to provide a complete view of every object's health in the group. This aggregated view is also the natural place for progress deadline evaluation, since the COD is the entity that understands the full revision chain.

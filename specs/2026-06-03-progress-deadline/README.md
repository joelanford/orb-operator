---
status: idea
---
# COS Progress Deadline

Add `progressDeadlineMinutes` to the ClusterObjectSet spec. This field sets a deadline for rollout progress — if a COSR doesn't make progress within this window, the COS controller marks the rollout as failed. Deferred from the initial COS API types work item to keep the first iteration focused on core stamping behavior.

## Prerequisites

- **COSR phase status** — progress deadline semantics should mirror Kubernetes Deployments, which track incremental progress (replica count changes) via a `LastUpdateTime` on the `Progressing` condition. Since `metav1.Condition` lacks `LastUpdateTime`, the COS needs per-phase progress information in the COSR status to serve as the progress signal. Without it, the only timer baseline is COSR `creationTimestamp`, which doesn't capture incremental progress and can't support Deployment-style deadline resets.

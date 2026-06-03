---
status: idea
---
# COS Progress Deadline

Add `progressDeadlineMinutes` to the ClusterObjectSet spec. This field sets a deadline for rollout progress — if a COSR doesn't make progress within this window, the COS controller marks the rollout as failed. Deferred from the initial COS API types work item to keep the first iteration focused on core stamping behavior.

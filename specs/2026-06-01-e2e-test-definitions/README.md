---
status: idea
---
# E2E Test Definitions

Define the full intended behavior of the operator as godog BDD feature files. Scenarios cover: single-revision COSR lifecycle, multi-revision ownership handoffs, phased rollout with per-object assertions, collision protection modes (Prevent, IfNoController, None), COS template stamping and revision management, and ClusterObjectSlice content resolution. Tests compile but fail — no controllers exist yet. This is the test-first definition of project intent.

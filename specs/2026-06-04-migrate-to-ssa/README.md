---
status: idea
---
# Migrate Controller Writes to Server-Side Apply (SSA)

The codebase currently uses a mix of `client.Create`, `client.Update`, and `client.Patch` with `MergePatch` for controller writes (COSR creation, status updates, archival mutations, finalizer removal, COSR adoption). SSA would provide field-level ownership tracking, built-in conflict detection, and declarative semantics. This would replace ~9 write call sites across `cos_controller.go` and `cosr_controller.go`.

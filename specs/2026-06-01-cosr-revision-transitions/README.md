---
status: idea
---
# COSR Controller — Revision Transitions

Extend the COSR controller to handle multi-revision scenarios within a group. Implement ownership handoffs: when a new revision (N+1) is created in the same group, transfer object ownership from revision N as objects in N+1 become ready. Implement the Active → Archived lifecycle transition: archival removes the old revision from owner lists and deletes objects that didn't carry over to the new revision.

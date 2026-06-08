---
status: superseded
superseded_by: 2026-06-04-controller-architecture
---
# COS Controller

Implement the ClusterObjectSet controller. COS acts like a Deployment: it holds a template for COSRs and stamps out new revisions when the template spec changes. The controller manages revision numbering within a group, propagates template.metadata (labels/annotations) to stamped COSRs, and derives its own status from the latest COSR's status.

**Superseded**: This idea is now part of the `2026-06-04-controller-architecture` spec, which defines both the COS and COSR controller architecture together.

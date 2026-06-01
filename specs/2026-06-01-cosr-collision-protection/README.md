---
status: idea
---
# COSR Controller — Collision Protection

Implement the three collision protection modes for the COSR controller: Prevent (only manage objects the revision created), IfNoController (adopt pre-existing objects not owned by another controller), and None (adopt any pre-existing object). Collision protection is configured at three levels with most-specific-wins precedence: object > phase > spec.

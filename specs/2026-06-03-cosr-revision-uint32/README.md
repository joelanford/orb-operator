---
status: done
---
# COSR Revision Field Cleanup

Change `ClusterObjectSetRevisionSpec.Revision` from `int32` to `uint32` and add `+kubebuilder:validation:Minimum=1`. Revision numbers are always positive integers starting at 1. This also affects the COSR controller code that references the field and the CRD print column type.

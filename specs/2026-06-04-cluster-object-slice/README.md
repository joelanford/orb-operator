---
status: idea
---
# Implement ClusterObjectSlice

ClusterObjectSlice is currently a bare stub with no spec fields. The ADR specifies that COSs should reference ClusterObjectSlices rather than embedding all object manifests directly, to avoid hitting etcd's 1.5 MiB object size limit for large bundles. This requires adding spec fields to ClusterObjectSlice (to hold object manifests), updating COSs to reference slices instead of (or in addition to) embedding objects inline, and updating the COS controller to resolve slice references when building boxcutter phases.

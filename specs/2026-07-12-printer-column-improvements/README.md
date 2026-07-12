---
status: idea
---
# Printer Column Improvements

## Summary

Improve COD, COS, and COSL printer columns to provide useful at-a-glance
information, mirroring Deployment and ReplicaSet column patterns where
the analogy fits. All count columns use the same noun (objects) with
different adjectives, matching how Deployment columns all count pods.

New integer status fields on COD and COS, and an `objectCount` field on
COSL, back the printer columns via JSONPath. The COSL field is populated
by a MutatingAdmissionPolicy.

A future UP-TO-DATE column on COD depends on the COS three-pass
reconcile work item (`specs/2026-07-12-cos-three-pass-reconcile/`).

## Design

### COD Columns

```
NAME           OBJECTS   AVAILABLE   AGE
my-extension   15        15          5m
```

New fields on `ClusterObjectDeploymentStatus`:
- `totalObjects int32` — total objects across all phases of the current
  (latest active) revision.
- `availableObjects int32` — objects in the current revision whose
  assertions pass.

The invariant `totalObjects >= availableObjects` always holds.

These replace the existing `Availability` (reason) and `Progressing`
(reason) printer columns. The Available and Progressing conditions
remain on the status — they just stop being printer columns.

### COS Columns

```
NAME              GROUP    REV   OBJECTS   AVAILABLE   LIFECYCLE   AGE
my-ext-abc-1      my-ext   1     12        12          Active      5m
```

New fields on `ClusterObjectSetStatus`:
- `totalObjects int32` — total objects across all phases.
- `availableObjects int32` — objects whose assertions pass (phases with
  status Available have all their objects available; phases with status
  Reconciling contribute their non-incomplete objects).

GROUP, REV, LIFECYCLE, and AGE are existing columns (REV was previously
named "Revision"). The existing `Available` (True/False from condition
status) column is replaced by the `availableObjects` integer.

### COSL Columns

```
NAME              OBJECTS   AGE
my-ext-abc-1-0    42        5m
```

New top-level field on `ClusterObjectSlice`:
- `objectCount int32` — number of entries in the `objects` list.

Populated by a MutatingAdmissionPolicy on create. Since COSL is
immutable, the policy only needs to run once.

### Computing COD counts

The COD controller already has access to all owned COS resources in
`updateStatus`. The new counts are derived from the COS status fields:

- `totalObjects`: `totalObjects` from the latest active COS that matches
  the current template hash.
- `availableObjects`: `availableObjects` from the latest active COS that
  matches the current template hash.

### Computing COS counts

The COS controller already builds `observedPhases` from the boxcutter
reconcile result. The new counts are derived in the same place:

- `totalObjects`: sum of object counts across all spec phases (counting
  inline objects and objectRefs).
- `availableObjects`: for each observed phase, count objects that are
  complete (total objects in that phase minus `len(incompleteObjects)`).

### Future: UP-TO-DATE column

An UP-TO-DATE column on COD requires the COS three-pass reconcile
(`specs/2026-07-12-cos-three-pass-reconcile/`), which adds up-to-date
detection via a paused boxcutter pass over unevaluated phases. Once
that lands, the COS can report `upToDateObjects` and the COD can
expose it as a printer column.

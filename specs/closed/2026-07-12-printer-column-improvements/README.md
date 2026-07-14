---
status: done
---
# Printer Column Improvements

## Summary

Improve COD and COS printer columns to provide useful at-a-glance
information, mirroring Deployment and ReplicaSet column patterns where
the analogy fits. All count columns use the same noun (objects) with
different adjectives, matching how Deployment columns all count pods.

New integer status fields on COD and COS back the printer columns via
JSONPath. The per-phase `ObjectCounts` added by the three-pass reconcile
work item provide the source data — the new top-level fields are sums.

## Design

### COD Columns

```
NAME           AVAILABLE   SYNCED   TOTAL   AGE
my-extension   15          15       15      5m
```

New field on `ClusterObjectDeploymentStatus`:
- `objectCounts ObjectCounts` — reuses the existing `ObjectCounts`
  struct (total, synced, available as int64). Values are copied from
  the latest active revision's status.

The invariant `total >= synced >= available` holds.

These replace the existing `Availability` (reason) and `Progressing`
(reason) printer columns. The Available and Progressing conditions
remain on the status — they just stop being printer columns.

### COS Columns

```
NAME              GROUP    REV   AVAILABLE   SYNCED   TOTAL   LIFECYCLE   AGE
my-ext-abc-1      my-ext   1     12          12       12      Active      5m
```

New field on `ClusterObjectSetStatus`:
- `objectCounts ObjectCounts` — reuses the existing `ObjectCounts`
  struct. Values are sums across all observed phases.

The existing `Available` (True/False from condition status) column is
replaced by the `availableObjects` integer. GROUP, REV, LIFECYCLE, and
AGE remain.

### Computing COD counts

The COD controller already has access to all owned COS resources in
`updateStatus`. The new counts are derived from the latest active COS
status fields:

- `objectCounts`: copied from the latest active COS's `objectCounts`.

"Latest active" means the non-archived, non-deleting COS with the
highest revision number.

### Computing COS counts

The COS controller already builds `observedPhases` with per-phase
`ObjectCounts` from the three-pass reconcile. The new top-level fields
are computed by summing across all observed phases:

- `objectCounts.total`: `sum(observedPhases[*].objectCounts.total)`
- `objectCounts.synced`: `sum(observedPhases[*].objectCounts.synced)`
- `objectCounts.available`: `sum(observedPhases[*].objectCounts.available)`

These are computed in `internal/status/cos/status.go` alongside the
existing `ObservedPhases` construction.

### Known limitation: SYNCED during revision transitions

During a revision transition, objects that are unchanged in content
will still show as "not synced" because their ownership metadata
(owner references, revision annotations) needs to change. This means
SYNCED may temporarily undercount during upgrades. A future
improvement could filter out ownership-related diffs to show a
content-only synced count.

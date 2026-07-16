---
status: done
---
# Accurate Object Counts During Teardown

## Summary

An archived COS with 14 objects shows `14/14/14` (available/synced/total) even
though all objects have been deleted. This work item adds a `present` field to
`ObjectCounts`, fixes teardown count semantics, and adds read-only presence
checks for read-only teardown phases so counts are always accurate.

## Problem

The root cause is that `mapSpecPhases` treats "complete" identically for reconcile
and teardown. A reconcile-complete phase correctly reports synced=total,
available=total. But a teardown-complete phase uses the same logic, even though
"complete" means all objects are gone.

A simple fix (set synced/available to 0 for teardown-complete phases) works for
the final state, but exposes deeper problems:

- **Tearing-down phases** report 0/0/0: `tearingDownPhase` never sets
  `ObjectCounts` at all (not even `Total`).
- **Read-only phases** (not yet reached by teardown) are marked Unknown with
  0/0/total. No object-level evaluation happens for these phases.

## Design

### New `present` field on ObjectCounts

Add a `Present` field that tracks how many objects exist on the cluster:

```go
type ObjectCounts struct {
    Total     int64 `json:"total"`
    Present   int64 `json:"present"`
    Synced    int64 `json:"synced"`
    Available int64 `json:"available"`
}
```

Printer columns for COS: AVAILABLE, SYNCED, PRESENT, TOTAL (in that order).

### Lifecycle-aware count semantics

Each lifecycle populates only the fields that are meaningful for it.

**Active (reconciling)** - all four fields are populated:

| Phase state | present | synced | available | total |
|---|---|---|---|---|
| Complete | total | total | total | total |
| Incomplete | objects processed by boxcutter | objects matching spec | objects passing assertions | total |
| Unevaluated | 0 | 0 | 0 | total |

**Tearing down** - only `present` and `total` are meaningful:

| Phase state | present | synced | available | total |
|---|---|---|---|---|
| TearingDown | objects waiting for deletion | 0 | 0 | total |
| Read-only (not yet reached) | objects found in cache | 0 | 0 | total |

**Teardown complete** - all four fields are explicitly set:

| Phase state | present | synced | available | total |
|---|---|---|---|---|
| TeardownComplete | 0 | 0 | 0 | total |

This applies at both the per-phase and aggregate COS levels.

### Read-only presence check for read-only teardown phases

During reconcile, phases that can't be actively reconciled still get a read-only
evaluation pass. Teardown needs the same pattern: phases that teardown hasn't
reached yet (read-only on a later phase still tearing down) should do a read-only
presence check against the cache rather than being left as Unknown.

This means teardown never produces Unknown/unevaluated phases. Every phase is
either TeardownComplete, TearingDown, or read-only with an accurate `present` count
derived from the cache.

Boxcutter has no dry-run teardown mode (`WithPaused` is reconcile-only), so the
presence check is done directly: for each object identity in a read-only phase, do a
cache lookup (Get by GVK + namespace + name) and count how many exist. This
bypasses boxcutter entirely for read-only phases.

### Differentiating complete-phase counts in mapSpecPhases

`mapSpecPhases` currently hardcodes `ObjectCounts{Total: total, Synced: total,
Available: total}` for all complete phases. To differentiate reconcile-complete
from teardown-complete, add a callback parameter (e.g.,
`buildCompleteObjectCounts func(total int64) ObjectCounts`) that each lifecycle
provides. Reconcile returns `{total, total, total, total}`. Teardown returns
`{total, 0, 0, 0}`.

### Validation

Extend the existing CEL validation rules on `ClusterObjectSetStatus`:
- `objectCounts.present` must equal the sum of per-phase present values
  (same pattern as synced, available, and total).

---
status: done
---
# Bump boxcutter to 0.14.0

## Summary

Upgrade `pkg.package-operator.run/boxcutter` from 0.13.1 to 0.14.0. The breaking change is the removal of the `WithPreviousOwners` type (a `[]client.Object` that implemented `RevisionReconcileOption`) in favor of `WithSiblingOwners([]client.Object)`, a constructor function that returns a `WithSiblingOwnerClassifier`.

The semantic meaning is the same: tell boxcutter which other owners are "friendly" so it can adopt objects from them without treating it as a collision. The rename from "previous" to "sibling" reflects that the relationship isn't necessarily temporal — sibling owners in the same chain share ownership regardless of order.

## Design

### API migration

**Old (0.13.1):**
```go
prevOwners := make(boxcutter.WithPreviousOwners, 0, len(previousOwners))
for _, po := range previousOwners {
    prevOwners = append(prevOwners, po)
}
reconcileOpts = append(reconcileOpts, prevOwners)
```

**New (0.14.0):**
```go
siblings := make([]client.Object, 0, len(predecessors))
for _, p := range predecessors {
    siblings = append(siblings, p)
}
reconcileOpts = append(reconcileOpts, boxcutter.WithSiblingOwners(siblings))
```

### Scope

The only compile error after bumping is at `internal/controller/cosr_controller.go:544` — the single reference to `boxcutter.WithPreviousOwners`. No other boxcutter API used by this project changed between 0.13.1 and 0.14.0. `NewRevisionWithOwner`, `NewPhaseWithOwner`, `RevisionEngine`, `WithCollisionProtection`, `WithProbe`, `WithObjectReconcileOptions`, and `ProbeFunc` are all unchanged.

### Naming

Rename `buildRevisionWithPreviousOwners` → `buildRevisionWithSiblings` and its `previousOwners` parameter → `siblings` to align with the new boxcutter vocabulary. The callers at lines 307 and 481 pass `predecessors` and `nil` respectively — no caller changes needed beyond the function name.

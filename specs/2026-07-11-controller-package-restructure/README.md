---
status: in-progress
---
# Controller Package Restructure

Extract non-controller-specific logic from `internal/controller/` into well-organized, independently testable packages with clean abstractions. Split the monolithic controller package into separate controller packages for COD and COS, with shared logic in sibling packages under `internal/`.

## Target Structure

```
internal/
├── assertions/                       # (existing, unchanged)
├── controller/
│   ├── cod/                          # COD reconciler
│   │   ├── controller.go
│   │   └── controller_test.go
│   └── cos/                          # COS reconciler
│       ├── controller.go
│       └── controller_test.go
├── object/                           # Object resolution + slice transform
│   ├── resolver.go
│   ├── resolver_test.go
│   ├── transform.go
│   └── transform_test.go
├── errors/                           # Typed reconcile outcome errors
│   └── errors.go
├── status/
│   ├── cod/                          # COD status evaluation (flat)
│   │   ├── availability.go
│   │   ├── availability_test.go
│   │   ├── progress.go
│   │   └── progress_test.go
│   └── cos/                          # COS status updater (flat)
│       ├── status.go
│       └── status_test.go
├── revision/                         # Revision lifecycle (flat package)
│   ├── chain.go
│   ├── chain_test.go
│   ├── engine.go
│   └── builder.go
├── cosutil/
│   └── apply.go
└── template/
    ├── template.go
    └── template_test.go
```

## Packages

### `internal/object/` — Object Resolution
Resolver struct embedding `client.Reader`. Provides `Resolve(ctx, []Phase) → (Result, error)` with transparent handling of inline JSON, slice references, and gzip decompression. Result exposes `ManagedObjects()` for cache setup and `VerifyHash()` for content integrity. Also contains `TransformClusterObjectSlice` cache transform.

### `internal/status/cod/` — COD Status Evaluation
- `availability.go` — `EvaluateAvailability(generation, activeRevisions) → Condition`, `IsAvailable(cos) → bool`
- `progress.go` — `EvaluateDeadline(cod, latestCOS, now, deadlineUnit) → (Condition, Result)`

### `internal/errors/` — Typed Reconcile Outcome Errors
Shared vocabulary between the "do work" layer and the "report status" layer. Both the controller and status packages import these types.

```go
// ObjectResolutionError indicates object resolution failed (slice not found,
// object missing from slice, content hash mismatch).
type ObjectResolutionError struct{ Err error }
func (e *ObjectResolutionError) Error() string { return e.Err.Error() }
func (e *ObjectResolutionError) Unwrap() error { return e.Err }

// InternalError indicates an internal controller error (engine setup, etc.)
type InternalError struct{ Err error }
func (e *InternalError) Error() string { return e.Err.Error() }
func (e *InternalError) Unwrap() error { return e.Err }
```

### `internal/status/cos/` — COS Status Updater
Declarative status updater. The COS controller returns `(result, error)` from reconcile/teardown; the status package owns the full mapping from outcomes to status fields.

```go
// Update captures all status mutations for a single reconcile/teardown cycle.
type Update struct {
    Condition      metav1.Condition
    ObservedPhases *[]orbv1alpha1.ObservedPhase // non-nil → replace
    CompletedAt    *metav1.Time                 // non-nil → set if not already set
}

// Apply writes the update to the COS status, setting ObservedGeneration from cos.Generation.
func Apply(cos *orbv1alpha1.ClusterObjectSet, u Update)

// FromReconcile maps a reconcile outcome to a status update.
// Uses errors.As on err to distinguish:
//   - *ObjectResolutionError → False (no hash) or Unknown (hash set), ReasonInvalidRevision
//   - *InternalError         → clear phases, Unknown/ReasonInternalError
//   - plain error            → compute phases from result, Unknown/ReasonReconcileError
//   - nil                    → compute phases, then: validation error → InvalidRevision,
//                              progressed → Superseded, complete → Available + CompletedAt,
//                              default → Unavailable
func FromReconcile(cos *orbv1alpha1.ClusterObjectSet, result RevisionResult, err error, now time.Time) Update

// FromTeardown maps a teardown outcome to a status update.
// Same error type assertions as FromReconcile, plus teardown-specific conditions.
func FromTeardown(cos *orbv1alpha1.ClusterObjectSet, result TeardownResult, err error, now time.Time) Update
```

### `internal/revision/` — Revision Lifecycle (flat)
- `chain.go` — `Build(members) → Chain`, `Chain.SiblingsOf(cos)`, `FilterByOwner(members, cos)`. Categorizes COS group members into latest-active, predecessors, archived, deleted.
- `engine.go` — Wraps boxcutter's RevisionEngine with drift correction for already-completed phases. `New(opts, existingPhases) → Engine`, `Engine.Reconcile()`, `Engine.Teardown()`.
- `builder.go` — `Build(cos, resolveResult, siblings, ownerStrategy) → boxcutter.Revision`, `MapCollisionProtection()`.

### `internal/cosutil/` — COS SSA + Finalizer Operations
`Apply()`, `RemoveFinalizer()`, `WaitForFinalizerRemoval()`, `ClearFinalizerFieldOwnership()`.

### `internal/template/` — COD Template Manager
`Hash(template) → (string, error)`, `BuildCOS(cod, revision, hash) → ApplyConfiguration`, `SetControllerReference(cod, cos)`.

### `internal/controller/cod/` — COD Reconciler
CODReconciler struct and all COD-specific reconcile methods.

`syncRevision` extracts the template hash check + create-or-fixup block from `reconcile`. Status is updated after `syncRevision` (the only mutation that affects availability/progress) but before archive/prune (pure housekeeping). Archive/prune failures don't affect status conditions — they just return errors for requeue, matching upstream Deployment controller behavior.

### `internal/controller/cos/` — COS Reconciler
COSReconciler struct, all COS-specific reconcile methods, and SetupIndexes.

`doReconcile` and `doTeardown` return `(result, error)` using typed errors from `internal/errors/` — they are completely status-unaware. The caller passes the result and error to `cosstatus.Apply(cos, cosstatus.FromReconcile(...))` which owns all status logic.

Shared setup (resolve → verify → engine → build revision) is extracted into `resolveAndPrepare`, called by both `doReconcile` and `doTeardown`.

## Controller Call Trees

### COS Controller

```
Reconcile(ctx, req)
├── client.Get(ctx, req, cos)
└── reconcile(ctx, log, cos)                              # pure router
    │
    ├── [deleted/archived] → teardown(ctx, log, cos)
    │   ├── [orphan finalizer] → release(ctx, cos)
    │   ├── doTeardown(ctx, cos) → (TeardownResult, error)
    │   │   ├── resolveAndPrepare(ctx, cos, nil)
    │   │   │   ├── resolver.Resolve(ctx, cos.Spec.Phases)     → *ObjectResolutionError
    │   │   │   ├── resolved.VerifyHash(…)                     → *ObjectResolutionError
    │   │   │   ├── newEngine(ctx, cos, resolved)               → *InternalError
    │   │   │   └── revision.Build(cos, resolved, nil, …)
    │   │   └── engine.Teardown(ctx, rev, …)                   → plain error
    │   ├── cosstatus.Apply(cos, cosstatus.FromTeardown(cos, result, err, now))
    │   ├── client.Status().Update(…)
    │   └── [complete] → release(ctx, cos)
    │       ├── accessManager.FreeWithUser(…)
    │       ├── cosutil.RemoveFinalizer(…)
    │       └── cosutil.WaitForFinalizerRemoval(…)
    │
    └── [active] → reconcileActive(ctx, log, cos)
        ├── resolveSiblings(ctx, cos) → []*COS
        │   ├── listGroupMembers(ctx, group)
        │   ├── revision.FilterByOwner(members, cos)
        │   ├── revision.Build(members) → Chain
        │   └── chain.SiblingsOf(cos)
        ├── ensureFinalizer(ctx, cos)
        │   └── cosutil.Apply(…)
        ├── doReconcile(ctx, cos, siblings) → (RevisionResult, error)
        │   ├── resolveAndPrepare(ctx, cos, siblings)
        │   │   ├── resolver.Resolve(ctx, cos.Spec.Phases)     → *ObjectResolutionError
        │   │   ├── resolved.VerifyHash(…)                     → *ObjectResolutionError
        │   │   ├── newEngine(ctx, cos, resolved)               → *InternalError
        │   │   └── revision.Build(cos, resolved, siblings, …)
        │   └── engine.Reconcile(ctx, rev, …)                  → plain error
        ├── cosstatus.Apply(cos, cosstatus.FromReconcile(cos, result, err, now))
        └── client.Status().Update(…)
```

### COD Controller

```
Reconcile(ctx, req)
├── client.Get(ctx, req, cod)
└── reconcile(ctx, cod)
    ├── listGroupRevisions(ctx, group) → allCOSs
    ├── adoptOrphans(ctx, cod, allCOSs) → ownedCOSs
    │   └── [no controller owner] → adopt(ctx, cod, cos)
    │       └── cosutil.Apply(…)                               # sets owner ref
    ├── syncRevision(ctx, cod, allCOSs, &ownedCOSs)
    │   ├── template.Hash(cod.Spec.Template)
    │   ├── [changed] → createRevision(ctx, cod, allCOSs, hash)
    │   │   ├── nextRevisionNumber(allCOSs)
    │   │   └── template.BuildCOS(cod, revision, hash)
    │   └── [same] → ensureFieldOwnership(ctx, latest, hash)
    ├── updateStatus(cod, ownedCOSs, now) → requeueAfter
    │   ├── codstatus.ActiveRevisionSummaries(ownedCOSs)
    │   ├── codstatus.EvaluateAvailability(…)
    │   └── codstatus.EvaluateDeadline(…)
    ├── client.Status().Update(…)
    ├── archiveSuperseded(ctx, &ownedCOSs)
    │   └── cosutil.Apply(…)                                   # set lifecycle=Archived
    └── pruneArchived(ctx, cod, &ownedCOSs)
        └── client.Delete(…)
```

Status is updated after `syncRevision` (reflects current active state including any newly created revision) but before `archiveSuperseded`/`pruneArchived` (housekeeping that doesn't affect availability/progress conditions). This matches upstream Deployment controller behavior where cleanup failures are invisible to status — they just cause a silent requeue.

## Deliverables

- [ ] Create `internal/object/` package with Resolver abstraction and slice transform
- [ ] Create `internal/status/cod/` package (availability, progress)
- [ ] Create `internal/errors/` package (ObjectResolutionError, InternalError)
- [ ] Create `internal/status/cos/` package (Update, Apply, FromReconcile, FromTeardown)
- [ ] Create `internal/revision/` package (chain, engine, builder)
- [ ] Create `internal/cosutil/` package
- [ ] Create `internal/template/` package
- [ ] Split `internal/controller/` into `internal/controller/cod/` and `internal/controller/cos/`
- [ ] Update all imports (cmd/, test/, etc.)
- [ ] All existing tests pass with no behavior changes
- [ ] Update `specs/tech-stack.md` project structure section

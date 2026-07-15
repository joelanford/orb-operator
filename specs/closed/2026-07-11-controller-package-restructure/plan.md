# Implementation Plan

Work bottom-up: create leaf packages first (no internal dependencies), then packages that depend on them, then the controllers last.

## 1. Create `internal/errors/`

- Define `ObjectResolutionError` and `InternalError` types
- Both implement `error` and `Unwrap() error`
- No dependencies on other internal packages

## 2. Create `internal/object/`

- Move `TransformClusterObjectSlice` from `controller/transform.go`
- Move `decompressGzip`, `unmarshalUnstructured` from `controller/cos_controller.go`
- Move `resolvedPhaseObjects`, `resolvedPhase`, `resolvedObject` types
- Introduce `Resolver` struct embedding `client.Reader` with `Resolve(ctx, []Phase) → (*Result, error)`
- Move `managedObjectsFromResolved` → `Result.ManagedObjects()`
- Move hash computation into `Result.Hash` field, `Result.VerifyHash(existing) → error`
- Migrate existing tests from `resolve_test.go` and `transform_test.go`
- Add tests for the new `Resolver` abstraction

## 3. Create `internal/status/cod/`

- Move `evaluateAvailability` → `EvaluateAvailability`
- Move `isCOSAvailable` → `IsAvailable`
- Move `evaluateProgressDeadline` → `EvaluateDeadline` (make it a function, not a method — pass `deadlineUnit` as parameter)
- Add `ActiveRevisionSummaries(ownedCOSs) → []ClusterObjectSetStatusSummary`
- Migrate existing tests from `cod_controller_test.go`

## 4. Create `internal/status/cos/`

- Define `Update` struct (Condition, ObservedPhases, CompletedAt)
- Implement `Apply(cos, Update)` — sets condition with ObservedGeneration, observed phases, CompletedAt
- Implement `FromReconcile(cos, RevisionResult, error, time.Time) → Update` — uses `errors.As` on typed errors to determine condition, computes observed phases from result, preserves completion times
- Implement `FromTeardown(cos, TeardownResult, error, time.Time) → Update` — same pattern for teardown
- Move all phase status mapping functions from `phase_status.go` (now unexported internals of this package)
- Move `truncateMessage` (unexported)
- Migrate existing tests from `phase_status_test.go`
- Add tests for `FromReconcile` and `FromTeardown` covering each error type branch

## 5. Create `internal/revision/`

- **chain.go**: Move `revisionChain` → `Chain`, `buildChain` → `Build`, `siblingsOf` → `SiblingsOf`, `controllerOwnerKey`/`controllerOwnerKeyOf` → unexported, `filterByControllerOwner` → `FilterByOwner`
- **engine.go**: Move `revisionEngine` → `Engine`, `revisionResult` → `Result`, `newRevisionEngine` → `New` with all methods
- **builder.go**: Move `buildRevisionFromResolved` → `Build` function (takes cos, resolved result, siblings, ownerStrategy), move `mapCollisionProtection` → `MapCollisionProtection`
- Add tests for chain building and filtering (currently untested)

## 6. Create `internal/cosutil/`

- Move `applyCOS` → `Apply`
- Move `removeFinalizer` → `RemoveFinalizer`
- Move `waitForFinalizerRemoval` → `WaitForFinalizerRemoval`
- Move `clearFinalizerFieldOwnership` → `ClearFinalizerFieldOwnership`

## 7. Create `internal/template/`

- Move `templateHash` → `Hash`
- Move `buildCOSFromTemplate` → `BuildCOS`
- Move `setCODControllerReference` → `SetControllerReference`
- Migrate existing tests from `template_hash_test.go`

## 8. Create `internal/controller/cos/`

- Move `COSReconciler` struct, constructor, `SetupWithManager`, `SetupIndexes`
- Restructure methods per the call tree:
  - `reconcile` as pure router (active vs teardown)
  - `reconcileActive`: resolveSiblings → ensureFinalizer → doReconcile → cosstatus.Apply → status update
  - `teardown`: orphan check → doTeardown → cosstatus.Apply → status update → release
  - `doReconcile(ctx, cos, siblings) → (RevisionResult, error)`: calls resolveAndPrepare then engine.Reconcile
  - `doTeardown(ctx, cos) → (TeardownResult, error)`: calls resolveAndPrepare then engine.Teardown
  - `resolveAndPrepare(ctx, cos, siblings) → (Engine, Revision, error)`: resolve → verify → newEngine → revision.Build
  - `resolveSiblings(ctx, cos) → ([]*COS, error)`: listGroupMembers → FilterByOwner → Build → SiblingsOf
  - `release(ctx, cos)`: free access manager → RemoveFinalizer → WaitForFinalizerRemoval
  - `newEngine(ctx, cos, resolved) → (Engine, error)`: managed objects → access manager → revision.New
- Move watch helpers: `cosesForSlice`, `cosReferencesSlice`, `mapCOSGroupToCOD` (stays with COD)

## 9. Create `internal/controller/cod/`

- Move `CODReconciler` struct, constructor, `SetupWithManager`
- Restructure methods per the call tree:
  - `reconcile`: listGroupRevisions → adoptOrphans → syncRevision → updateStatus → status update → archiveSuperseded → pruneArchived
  - `syncRevision(ctx, cod, allCOSs, &ownedCOSs)`: template.Hash → createRevision or ensureFieldOwnership
  - `createRevision(ctx, cod, allCOSs, hash)`: nextRevisionNumber → template.BuildCOS → create + apply
  - `ensureFieldOwnership(ctx, latest, hash)`: template.BuildCOS → compare → apply if drifted
  - `adoptOrphans(ctx, cod, allCOSs) → ownedCOSs`: iterate, adopt orphans via cosutil.Apply
  - `adopt(ctx, cod, cos)`: cosutil.Apply with owner ref
  - `updateStatus(cod, ownedCOSs, now) → requeueAfter`: codstatus.ActiveRevisionSummaries → EvaluateAvailability → EvaluateDeadline
  - `archiveSuperseded(ctx, &ownedCOSs)`: set lifecycle=Archived on non-latest when latest is available
  - `pruneArchived(ctx, cod, &ownedCOSs)`: delete excess beyond revisionHistoryLimit
  - `listGroupRevisions(ctx, group) → allCOSs`
  - `nextRevisionNumber(allCOSs) → uint32`
- Move watch helper `mapCOSGroupToCOD`

## 10. Remove `internal/controller/` and update imports

- Delete all files from old `internal/controller/` package
- Update `cmd/operator/` imports to reference new controller packages
- Update any test imports
- Verify no import cycles: `go vet ./...`

## 11. Update project documentation

- Update `specs/tech-stack.md` project structure section to reflect new layout

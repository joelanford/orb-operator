# Requirements

## API Types

- Flesh out `ClusterObjectSet` spec with `revisionHistoryLimit` and `template` fields
- `ClusterObjectSetSpec.RevisionHistoryLimit` is `*int32`, optional, defaults to 5 (applied by controller, not defaulting webhook)
- `ClusterObjectSetSpec.Template` is `ClusterObjectSetTemplate` with `Metadata` and `Spec` fields
- `ClusterObjectSetTemplate.Metadata` is `ClusterObjectSetTemplateMetadata` with `Labels` and `Annotations` (both `map[string]string`)
- `ClusterObjectSetTemplate.Spec` is `ClusterObjectSetTemplateSpec` containing only user-settable fields: `CollisionProtection *CollisionProtection` and `Phases []Phase`
- `ClusterObjectSetTemplateSpec` is a separate type containing the templatable subset of COSR fields
- **Refactor `ClusterObjectSetRevisionSpec`** to embed `ClusterObjectSetTemplateSpec` with `json:",inline"` — this replaces the existing `CollisionProtection` and `Phases` fields on the COSR spec with the shared embedded type, keeping the CRD serialization unchanged
- Existing COSR controller code continues to work unchanged since field access (`cosr.Spec.Phases`, `cosr.Spec.CollisionProtection`) is the same with inline embedding
- CEL immutability rules for `phases` and `collisionProtection` stay on `ClusterObjectSetRevisionSpec`, NOT on `ClusterObjectSetTemplateSpec` — the COS template fields must be freely mutable
- The COSR CRD shape must remain unchanged after the refactor (except for the group MaxLength update)
- Update COSR group field `MaxLength` from 253 to 52 (52 group + 1 dash + 10 revision digits = 63, fits in a label value)
- COS name is implicitly capped at 52 characters since it becomes the group
- `ClusterObjectSetStatus` has `Conditions []metav1.Condition`
- Add `Available` condition type and reason constants (reuse existing COSR constants)
- Add `Progressing` reason constant for COS-specific status
- Regenerate deepcopy methods via controller-gen
- Regenerate COS CRD manifest to `deploy/crds/`
- CRD has print columns: Availability (reason from Available condition), Age
- CRD has subresource status enabled

## E2E Test Infrastructure

- COS CRD is deployed to the kind cluster as part of `make run` (generated CRDs in `deploy/crds/` are applied via the jsonnet-rendered manifests)
- Add COS builder to `test/e2e/context.go` for constructing COS specs in step definitions
- COS builder reuses existing COSR builder for template.spec content
- Each COS scenario runs in the existing isolated-namespace pattern
- Tests run against the live kind cluster via `ctrl.GetConfig()`

## Feature Coverage

Feature files must cover these COS behaviors:

### COSR Stamping
- COS creates a COSR from its template with `group` matching the COS name
- Stamped COSR has `revision: 1` on initial creation
- Stamped COSR has `lifecycleState: Active`
- Stamped COSR spec matches `template.spec` (phases, collisionProtection)

### Template Metadata Propagation
- `template.metadata.labels` are set on the stamped COSR's ObjectMeta labels
- `template.metadata.annotations` are set on the stamped COSR's ObjectMeta annotations

### Revision Management
- Updating `template.spec` creates a new COSR with revision 2 in the same group
- Updating `template.metadata` creates a new COSR with incremented revision
- Multiple template changes produce revisions 1, 2, 3... (monotonically increasing)
- No new COSR is created when template hasn't changed (idempotent reconciliation)

### Status Derivation
- COS shows `Available=True, Reason=Available` when a single Active COSR exists and has `Available=True`
- COS shows `Available=False, Reason=Unavailable` when a single Active COSR exists and is not Available, or no COSR exists
- COS shows `Available=Unknown, Reason=Progressing` when multiple Active COSRs exist (rollout in progress)

### Ownership
- COS sets an owner reference on each stamped COSR
- Owner reference has `controller: true` and `blockOwnerDeletion: true`
- Deleting the COS cascades deletion to all owned COSRs

### Revision History Limit
- Archived COSRs beyond `revisionHistoryLimit` are deleted (lowest revision first)
- Active COSR is never counted toward the limit
- Default limit of 5 applies when `revisionHistoryLimit` is not set
- Setting `revisionHistoryLimit: 0` retains no archived COSRs

## Acceptance Criteria

- `go build ./...` succeeds
- `make generate` produces deepcopy and CRD manifests with no diff
- `make test-e2e` runs and all COS tests fail with timeout/assertion errors (not compile or setup errors)
- `make lint` passes
- COS CRD manifest in `deploy/crds/` has correct group, version, kind, scope
- Feature files are valid Gherkin (godog parses them without error)
- All COS step definitions are implemented (no undefined steps, no `godog.ErrPending`)
- Every COS scenario fails with a timeout or assertion error, not a compilation or infrastructure error
- Existing COSR tests continue to pass/fail unchanged

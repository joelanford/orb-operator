# Requirements

## API Types

- Define `ClusterObjectSetRevision` and `ClusterObjectSetRevisionList` types in `api/v1alpha1/`
- Define supporting types: `ClusterObjectSetRevisionSpec`, `ClusterObjectSetRevisionStatus` (conditions only), `Phase`, `PhaseObject`, `Assertion`, `ConditionEqualAssertion`, `FieldsEqualAssertion`, `FieldValueAssertion`, `CELExpressionAssertion`, `LifecycleState`
- Types follow ADR-0001 field names and semantics
- `lifecycleState` defaults to `Active` when unset
- Each assertion entry must set exactly one of `conditionEqual`, `fieldsEqual`, `fieldValue`, `celExpression`
- Generate deepcopy methods via controller-gen
- Register types with the runtime scheme in `api/v1alpha1/groupversion_info.go`
- Generate CRD manifest to `deploy/crds/` via controller-gen
- Group/version: `orb.io/v1alpha1`

## E2E Test Infrastructure

- Add `github.com/cucumber/godog` as a dependency in go.mod
- Create test suite in `test/e2e/` using godog with envtest backend
- COSR CRD installed in envtest from generated manifests in `deploy/crds/`
- Each scenario runs in an isolated namespace
- Polling-based assertions with configurable timeout (default: 10s — tests are expected to fail)
- Tests compile and run via `make test-e2e`

## Feature Coverage

Feature files must cover these COSR single-revision behaviors:

- **Object creation**: COSR with inline objects in one phase → objects created in cluster
- **Multiple objects**: COSR with multiple objects in one phase → all created
- **Phase ordering**: multi-phase COSR → phases execute sequentially, phase N completes before phase N+1 starts
- **Assertion gating**: failing assertion on phase N object → phase N+1 blocked
- **ConditionEqual**: asserts an object has a condition with specified type and status
- **FieldsEqual**: asserts two field paths on an object have matching values
- **FieldValue**: asserts a field path on an object has a specific value
- **CELExpression**: evaluates a CEL expression against the object, passes when it returns true; optional message for failure description
- **Built-in assertions**: known resource kinds have implicit readiness checks (CRD → Established=True; Deployment → updatedReplicas == replicas)
- **Status — Available**: COSR shows Available=True when all phases complete and all assertions pass
- **Active lifecycle — reconciliation**: controller recreates a managed object if it is deleted externally
- **Archive lifecycle — teardown**: setting lifecycleState to Archived on a single-revision COSR deletes all managed objects
- **Revision transition — handoff**: creating revision N+1 in the same group transfers object ownership from revision N as N+1's objects become ready
- **Revision transition — superseded status**: revision N shows Available=False/Superseded when N+1 exists in the same group
- **Revision transition — shared objects**: objects common to both revisions transfer ownership seamlessly (no delete/recreate)
- **Revision transition — removed objects**: objects in revision N but not in N+1 are deleted when N is archived
- **Revision transition — new objects**: objects in N+1 but not in N are created fresh
- **Revision transition — archival**: revision N is archived after N+1 fully rolls out, Available=False/Archived

## Acceptance Criteria

- `go build ./...` succeeds
- `make generate` produces deepcopy and CRD manifest with no diff
- `make test-e2e` runs and all tests fail with timeout/assertion errors (not compile or setup errors)
- `make lint` passes
- CRD manifest in `deploy/crds/` has correct group (`orb.io`), version (`v1alpha1`), kind (`ClusterObjectSetRevision`)
- Feature files are valid Gherkin (godog parses them without error)
- All step definitions are implemented (no undefined steps, no godog.ErrPending)
- Every scenario fails with a timeout or assertion error, not a compilation or infrastructure error

---
status: done
---
# COSR API Types & E2E Test Definitions

## Summary

Define the ClusterObjectSetRevision (COSR) v1alpha1 API types and write godog BDD e2e tests that define the single-revision COSR lifecycle. API types include Go struct definitions, deepcopy generation, scheme registration, and CRD manifest generation. E2E tests use envtest with full infrastructure — they compile and run but fail because no controller exists yet. This is the test-first definition of COSR behavior.

Scope: COSR only, inline objects only. No COS, no ClusterObjectSlice, no collision protection.

## Design

### COSR API Type

The ClusterObjectSetRevision is a cluster-scoped resource following the ADR-0001 design. For this iteration, objects are embedded inline in phases (ClusterObjectSlice references come later).

Group/version: `orb.operatorframework.io/v1alpha1`

Type hierarchy:

```
ClusterObjectSetRevision
├── spec
│   ├── group: string                    # revision chain identifier (immutable)
│   ├── revision: int32                  # monotonic revision number (immutable)
│   ├── lifecycleState: Active|Archived  # mutable, one-way transition
│   └── phases: []Phase                  # ordered phase list (immutable)
│       ├── name: string
│       └── objects: []PhaseObject
│           ├── object: RawExtension     # inline Kubernetes manifest
│           └── assertions: []Assertion  # per-object readiness checks
│               ├── conditionEqual: {type, status}
│               ├── fieldsEqual: {fieldA, fieldB}
│               ├── fieldValue: {fieldPath, value}
│               └── celExpression: {expression, message?}
└── status
    └── conditions: []Condition          # standard metav1 conditions
```

Available condition variants:
- `Available=True` — all phases completed, all assertions passing
- `Available=False, Reason=Unavailable` — phases not complete, assertions not met, or objects missing
- `Available=False, Reason=Superseded` — newer revision in the same group is taking over
- `Available=False, Reason=Archived` — lifecycleState is Archived, objects being removed

Status is intentionally minimal — just conditions. Structured phase/object status may be added later.

`lifecycleState` defaults to `Active` when unset.

Each assertion entry must set exactly one of its four fields (`conditionEqual`, `fieldsEqual`, `fieldValue`, `celExpression`).

### E2E Test Architecture

Tests use godog (Gherkin BDD) backed by controller-runtime envtest. The test suite:
1. Starts an envtest API server with the COSR CRD installed
2. Does NOT start any controllers — tests define expected behavior, not actual behavior
3. Creates COSR resources and asserts expected state via polling with timeouts

All tests are expected to fail (timeout waiting for controller actions). When the COSR controller is implemented later, these tests become the acceptance suite.

### Feature File Organization

Feature files are organized by behavior domain:

- `cosr_object_creation.feature` — COSR creates managed objects from inline phases
- `cosr_phase_progression.feature` — phases execute in order, gated by assertions
- `cosr_assertions.feature` — assertion type evaluation (ConditionEqual, FieldsEqual, FieldValue, built-in)
- `cosr_status.feature` — COSR status conditions reflect rollout state
- `cosr_lifecycle.feature` — Active state behavior (object reconciliation) and Archived state behavior (object deletion)
- `cosr_revision_transitions.feature` — multi-revision ownership handoffs within a group, Superseded status, archival cleanup

### Step Definition Patterns

Step definitions fall into three categories:

1. **Setup steps** — build COSR specs programmatically ("Given a COSR with group X and revision Y")
2. **Action steps** — create/update/delete resources via envtest client ("When the COSR is created")
3. **Assertion steps** — poll for expected state with timeout ("Then the ConfigMap should exist")

Each scenario gets an isolated namespace and a fresh test context carrying the envtest client and COSR builder state.

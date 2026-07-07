---
status: complete
---
# COSR Phase Status

## Summary

Add status fields to `ClusterObjectSetRevisionStatus` that support three use cases: diagnosing stuck rollouts, enabling progress deadline detection, and distinguishing initial rollouts from regressions.

## Prerequisites

- **Independent COSR reconciliation** (done) — each COSR reconciles independently with sibling awareness, producing meaningful boxcutter results for all active COSRs.

## Use Cases

### 1. Diagnose stuck rollouts

**Problem:** A user sees Available=False on a COSR and has no way to determine which phase is stuck, which objects are unavailable, or what the probe errors say — without access to controller logs.

**Need:** Per-phase breakdown showing which phase the rollout is blocked on, which objects in that phase have problems, and the specific failure messages.

### 2. Progress deadline (future)

**Problem:** The COS controller needs to detect whether a COSR is making forward progress toward Available=True, in order to implement Deployment-style progress deadline timeouts. Without a progress signal, the only option is a wall-clock timer from COSR creation, which can't distinguish "stuck" from "slow but progressing."

**Need:** A granular progress signal that changes when the rollout moves forward (an incomplete object is resolved, a phase completes). The COS controller compares this signal between reconciles to decide whether to reset the deadline timer.

### 3. Distinguish rollout vs regression

**Problem:** When a COSR shows Available=False, users can't tell whether it has never been available (still rolling out) or was previously available and regressed.

**Need:** A signal that records whether the COSR has ever reached Available=True.

## Design

### Derived API fields

Working backward from the use cases:

**Use case 1 (diagnostics)** requires:
- Per-phase status showing whether the phase is reconciling, complete, or unknown
- For active phases, a list of incomplete objects with their failure messages
- Object identity (group, kind, name) so the user knows where to look

**Use case 2 (progress deadline)** requires:
- A signal that changes on forward progress — phase status transitions (Unknown → Reconciling → Available) and the `incompleteObjects` list shrinking both serve this role.

**Use case 3 (rollout vs regression)** requires:
- A timestamp recording when the COSR first reached Available=True, set once and never cleared.

### Status shape

```go
type ClusterObjectSetRevisionStatus struct {
    // conditions represent the latest available observations of the revision's
    // state. The "Available" condition indicates whether all managed objects in
    // this revision satisfy their assertions.
    // +listType=map
    // +listMapKey=type
    // +optional
    Conditions []metav1.Condition `json:"conditions,omitempty"`

    // completedAt is the timestamp when all phases first completed
    // successfully. Set once and never cleared. Nil means the revision
    // has never been fully available. When set and Available is False,
    // the revision has regressed after a successful rollout.
    // +optional
    CompletedAt *metav1.Time `json:"completedAt,omitempty"`

    // observedPhases reports the observed state of each phase in the
    // revision. All phases from the spec are always listed, in spec
    // order. Each phase's status indicates whether the controller has
    // evaluated it and whether it has completed.
    // +listType=map
    // +listMapKey=name
    // +optional
    ObservedPhases []ObservedPhase `json:"observedPhases,omitempty"`
}

// PhaseStatus describes the current state of a phase in the rollout.
//
// +kubebuilder:validation:Enum=Reconciling;Available;Unknown;Superseded;TearingDown;TeardownComplete
type PhaseStatus string

const (
    // PhaseStatusReconciling indicates the controller is actively evaluating
    // this phase. Objects may or may not have failures.
    PhaseStatusReconciling PhaseStatus = "Reconciling"

    // PhaseStatusAvailable indicates all objects in this phase have been
    // successfully reconciled and pass their assertions.
    PhaseStatusAvailable PhaseStatus = "Available"

    // PhaseStatusUnknown indicates this phase was not evaluated during
    // the most recent reconcile. The controller has not yet reached it
    // or an earlier phase is incomplete.
    PhaseStatusUnknown PhaseStatus = "Unknown"

    // PhaseStatusTearingDown indicates the controller is actively deleting
    // objects in this phase. Objects still awaiting deletion are listed
    // in incompleteObjects.
    PhaseStatusTearingDown PhaseStatus = "TearingDown"

    // PhaseStatusTeardownComplete indicates all objects in this phase have
    // been deleted from the cluster.
    PhaseStatusTeardownComplete PhaseStatus = "TeardownComplete"
)

type ObservedPhase struct {
    // name is the name of the phase from the spec.
    // +required
    Name string `json:"name"`

    // status is the current state of this phase in the rollout.
    // +required
    Status PhaseStatus `json:"status"`

    // incompleteObjects lists objects in this phase that are not
    // successfully reconciled. For Reconciling phases, this includes
    // probe failures, collisions, creation/update errors, and any
    // other condition that prevents the object from being complete.
    // For TearingDown phases, this lists objects still awaiting
    // deletion. Each entry identifies the object and carries failure
    // messages. Empty when status is Available, TeardownComplete, or
    // Unknown.
    // +optional
    IncompleteObjects []ObjectStatus `json:"incompleteObjects,omitempty"`
}

type ObjectStatus struct {
    // group is the API group of the object (empty string for core resources).
    // +optional
    Group string `json:"group,omitempty"`

    // version is the API version of the object.
    // +required
    Version string `json:"version"`

    // kind is the kind of the object.
    // +required
    Kind string `json:"kind"`

    // namespace is the namespace of the object. Empty for cluster-scoped
    // resources.
    // +optional
    Namespace string `json:"namespace,omitempty"`

    // name is the name of the object.
    // +required
    Name string `json:"name"`

    // messages lists the failure reasons for this object.
    // +required
    Messages []string `json:"messages"`
}
```

### Field-to-use-case traceability

| Field | UC1 (diagnostics) | UC2 (progress deadline) | UC3 (rollout vs regression) |
|---|---|---|---|
| `completedAt` | | permanently satisfied gate | primary signal |
| `observedPhases[].name` | which phase | | |
| `observedPhases[].status` | phase state (includes Unknown for regressions) | phase-level progress | |
| `observedPhases[].incompleteObjects` | what's broken | intra-phase progress (list shrinks) | |

Every field traces to at least one use case. No field is redundant.

### Design decisions

- **All phases always listed** — every phase from the spec appears in `observedPhases`, even if the controller hasn't evaluated it yet. This gives users the full rollout plan at a glance and avoids confusion about whether a missing phase means "not yet reached" or "something is wrong."
- **Six-state `status` enum** — `Reconciling` (controller is working on it), `Available` (all objects reconciled), `Unknown` (not evaluated this reconcile), `Superseded` (objects adopted by a newer revision), `TearingDown` (actively deleting objects), `TeardownComplete` (all objects deleted). The teardown states provide visibility during archival and deletion. Status is derived entirely from the current reconcile — the controller never reads its own prior status to decide the new state.
- **`incompleteObjects` covers all failure modes** — probe failures, collisions, creation/update errors, and validation errors for `Reconciling` phases; objects awaiting deletion for `TearingDown` phases.
- **`completedAt` is write-once** — mirrors Deployment semantics where the progress deadline is permanently satisfied once the rollout succeeds. Prevents false timeouts on regressions.
- **`listType=map` with `listMapKey=name`** — enables SSA-friendly merging by phase name.

### Mapping from boxcutter results

#### Active reconciliation

The COSR controller calls `engine.Reconcile()` and gets a `RevisionResult` with `GetPhases() []PhaseResult`. The spec's full phase list is used to produce `observedPhases`:

- Phases returned by `GetPhases()` are either `Reconciling` or `Available` based on `IsComplete()`
- For `Reconciling` phases, `GetObjects()` is iterated: objects where `IsComplete()` is false become `ObjectStatus` entries with group/kind/name and messages from `ProbeResults()`, collision info, or reconcile errors. `GetValidationError()` is surfaced as an incomplete object message when present.
- Phases not returned by `GetPhases()` (beyond the first incomplete phase) are set to `Unknown`

#### Teardown (archival and deletion)

The COSR controller calls `engine.Teardown()` and gets a `RevisionTeardownResult` with `GetPhases() []PhaseTeardownResult`. The spec's full phase list is used to produce `observedPhases`:

- Phases returned by `GetPhases()` are either `TeardownComplete` or `TearingDown` based on `IsComplete()`
- For `TearingDown` phases, `Waiting()` is iterated: each `ObjectRef` becomes an `ObjectStatus` entry with "awaiting deletion" as the message
- Phases not returned by `GetPhases()` are set to `Unknown`
- Once teardown is fully complete, `observedPhases` is cleared

### COS status changes

None for this work item. The COS will consume the COSR's phase status in the progress deadline follow-up.

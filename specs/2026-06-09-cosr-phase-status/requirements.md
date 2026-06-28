# Requirements

- The COSR status must report all phases from the spec in `observedPhases`, in spec order, on every status update.
- Each observed phase has a `status` of `Reconciling`, `Available`, or `Unknown`.
- `Reconciling` phases are being evaluated and may have incomplete objects.
- `Available` phases have all objects successfully reconciled with passing assertions.
- `Unknown` phases were not evaluated during the most recent reconcile.
- `incompleteObjects` is populated for `Reconciling` phases (probe failures, collisions, creation/update errors, validation errors) and `TearingDown` phases (objects awaiting deletion).
- `incompleteObjects` is empty for `Available`, `TeardownComplete`, and `Unknown` phases.
- Status is derived entirely from the current reconcile — the controller never reads its own prior status to decide the new state.
- `completedAt` must be set to the current time when all phases first complete. Once set, it must never be cleared or updated, even if the revision later regresses or is archived.
- Phase status must be updated atomically with the existing `Available` condition (same status update call).
- During teardown (archival or deletion), `observedPhases` must report `TearingDown` or `TeardownComplete` status for each phase, with objects awaiting deletion listed as incomplete.
- Once teardown is fully complete, all phases show `TeardownComplete`. `completedAt` must be preserved.
- Superseded COSRs must show all phases with status `Superseded`.

## Acceptance Criteria

- A COSR with 3 phases where phase 1 is complete and phase 2 has an incomplete object shows `observedPhases: [{name: "phase-1", status: "Available"}, {name: "phase-2", status: "Reconciling", incompleteObjects: [{kind: "Deployment", name: "my-operator", messages: ["condition Available is not True"]}]}, {name: "phase-3", status: "Unknown"}]`.
- A COSR where all phases complete shows all phases with `status: "Available"`, the `Available` condition as `True`, and `completedAt` set.
- A COSR that was previously Available=True and then regresses shows `completedAt` still set (non-nil) with Available=False, the regressed phase as `Reconciling` with incomplete objects, and later phases as `Unknown`.
- An archived COSR shows all phases with status `TeardownComplete` and retains `completedAt`.
- A superseded COSR shows all phases with status `Superseded`.
- Phase status persists across controller restarts (stored in COSR status, not in memory).
- Integration tests verify phase status for multi-phase COSRs at each stage of rollout.
- E2e tests verify phase status is observable via the API for a multi-phase COS.

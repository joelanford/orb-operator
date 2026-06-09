# Requirements

- The COSR status must report all phases from the spec in `observedPhases`, in spec order, on every status update.
- Each observed phase has a `status` of `Reconciling`, `Complete`, or `Unknown`.
- `Reconciling` phases are being evaluated and may have unavailable objects.
- `Complete` phases have all objects successfully reconciled with passing assertions.
- `Unknown` phases were not evaluated during the most recent reconcile.
- `unavailableObjects` is populated only for `Reconciling` phases and covers all failure modes: probe failures, collisions, creation/update errors, and validation errors.
- `unavailableObjects` is empty for `Complete` and `Unknown` phases.
- Status is derived entirely from the current reconcile — the controller never reads its own prior status to decide the new state.
- `completedAt` must be set to the current time when all phases first complete. Once set, it must never be cleared or updated, even if the revision later regresses or is archived.
- Phase status must be updated atomically with the existing `Available` condition (same status update call).
- Archived and superseded COSRs must clear their `observedPhases` but preserve `completedAt`.
- Deleted COSRs undergoing teardown must not report `observedPhases`.

## Acceptance Criteria

- A COSR with 3 phases where phase 1 is complete and phase 2 has an unavailable object shows `observedPhases: [{name: "phase-1", status: "Complete"}, {name: "phase-2", status: "Reconciling", unavailableObjects: [{kind: "Deployment", name: "my-operator", messages: ["condition Available is not True"]}]}, {name: "phase-3", status: "Unknown"}]`.
- A COSR where all phases complete shows all phases with `status: "Complete"`, the `Available` condition as `True`, and `completedAt` set.
- A COSR that was previously Available=True and then regresses shows `completedAt` still set (non-nil) with Available=False, the regressed phase as `Reconciling` with unavailable objects, and later phases as `Unknown`.
- An archived COSR shows empty `observedPhases` but retains `completedAt`.
- A superseded COSR shows empty `observedPhases`.
- Phase status persists across controller restarts (stored in COSR status, not in memory).
- Integration tests verify phase status for multi-phase COSRs at each stage of rollout.
- E2e tests verify phase status is observable via the API for a multi-phase COS.

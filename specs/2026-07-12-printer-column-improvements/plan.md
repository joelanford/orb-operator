# Implementation Plan

1. **COS API and status changes**
   - Add `objectCounts ObjectCounts` to `ClusterObjectSetStatus` (reuses the existing struct).
   - Update printer column annotations: replace `Available` condition column with AVAILABLE, SYNCED, TOTAL integer columns. Keep GROUP, REV, LIFECYCLE, AGE.
   - In `internal/status/cos/status.go`, compute the sums from `observedPhases` when building the `Update`. Set all to zero when `ObservedPhases` is nil.
   - Run `make generate`.
   - Unit tests for count computation.

2. **COD API and status changes**
   - Add `objectCounts ObjectCounts` to `ClusterObjectDeploymentStatus` (reuses the existing struct).
   - Update printer column annotations: replace Availability and Progressing reason columns with AVAILABLE, SYNCED, TOTAL integer columns. Keep AGE.
   - In the COD controller's `updateStatus` path, derive counts from the latest active COS status fields.
   - Run `make generate`.
   - Unit tests.

3. **E2e scenario updates**
   - Update existing e2e scenarios to assert printer-column-backing fields where relevant.
   - Add scenario verifying COD columns reflect COS state during rollout.

4. **Verify**
   - `make verify`.
   - `make test-unit`.
   - `make test-e2e`.

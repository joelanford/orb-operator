# Requirements

- ObjectCounts includes a `present` field that tracks objects existing on the cluster
- During reconcile, `present` is populated alongside synced/available/total
- During teardown, synced and available are always 0; present tracks remaining objects
- Teardown-complete phases report present=0, synced=0, available=0, total=total
- Tearing-down phases report present=len(waitingForDeletion), synced=0, available=0, total=total
- Read-only teardown phases (not yet reached) get a read-only cache lookup for presence instead of being marked Unknown
- Aggregate COS-level objectCounts sums all per-phase counts, including present
- CEL validation enforces that objectCounts.present equals the sum of per-phase present values

## Acceptance Criteria

- An archived COS with all objects deleted shows objectCounts 0/0/0/total (present/synced/available/total)
- A COS mid-teardown shows accurate present counts for the actively tearing-down phase
- A COS mid-teardown shows accurate present counts for read-only phases (via cache lookup, not Unknown status)
- Reconcile path populates present alongside synced/available without behavior changes
- Printer columns show AVAILABLE, SYNCED, PRESENT, TOTAL in that order
- CEL validation rejects objectCounts.present that doesn't equal the sum of per-phase present
- Unit tests cover reconcile and teardown count computation for all phase states
- E2e scenarios assert on object counts during and after teardown

Feature: COS phase status reports per-phase rollout state

  Scenario: Phase status shows progression from Pending to WaitingForAssertions to Available
    Given a COS with group "ps" and revision 1
    And a phase "crds" with a gated ConfigMap "cm-crds"
    And a phase "operators" with a gated ConfigMap "cm-operators"
    And a phase "config" with a gated ConfigMap "cm-config"
    When the COS is created
    Then the COS should have 3 observed phases
    And the COS should have observed phase "crds" with status "WaitingForAssertions"
    And observed phase "crds" should have object counts total:1/present:1/synced:1/available:0
    And the COS should have observed phase "operators" with status "Pending" and message "Waiting for earlier phases to complete"
    And observed phase "operators" should have object counts total:1/present:0/synced:0/available:0
    And the COS should have observed phase "config" with status "Pending" and message "Waiting for earlier phases to complete"
    And observed phase "config" should have object counts total:1/present:0/synced:0/available:0
    And the COS should have object counts total:3/present:1/synced:1/available:0
    And the COS should not have completedAt set
    When the gate on ConfigMap "cm-crds" is opened
    Then the COS should have observed phase "crds" with status "Available"
    And observed phase "crds" should have object counts total:1/present:1/synced:1/available:1
    And the COS should have observed phase "operators" with status "WaitingForAssertions"
    And observed phase "operators" should have object counts total:1/present:1/synced:1/available:0
    And the COS should have observed phase "config" with status "Pending" and message "Waiting for earlier phases to complete"
    And observed phase "config" should have object counts total:1/present:0/synced:0/available:0
    And the COS should have object counts total:3/present:2/synced:2/available:1
    When the gate on ConfigMap "cm-operators" is opened
    And the gate on ConfigMap "cm-config" is opened
    Then the COS should have condition "Available" with status "True"
    And the COS should have observed phase "crds" with status "Available"
    And observed phase "crds" should have object counts total:1/present:1/synced:1/available:1
    And the COS should have observed phase "operators" with status "Available"
    And observed phase "operators" should have object counts total:1/present:1/synced:1/available:1
    And the COS should have observed phase "config" with status "Available"
    And observed phase "config" should have object counts total:1/present:1/synced:1/available:1
    And the COS should have object counts total:3/present:3/synced:3/available:3
    And the COS should have completedAt set

  Scenario: Object details listed in WaitingForAssertions phases
    Given a COS with group "ps-unavail" and revision 1
    And a phase "install" with a gated ConfigMap "cm-unavail"
    When the COS is created
    Then the COS should have observed phase "install" with status "WaitingForAssertions"
    And observed phase "install" should have object counts total:1/present:1/synced:1/available:0
    And observed phase "install" should have object details for "cm-unavail"

  Scenario: Object details listed in Pending phases
    Given a COS with group "ps-pending" and revision 1
    And a phase "phase-1" with a gated ConfigMap "cm-pending-1"
    And a phase "phase-2" with a ConfigMap "cm-pending-2"
    When the COS is created
    Then the COS should have observed phase "phase-2" with status "Pending"
    And observed phase "phase-2" should have object counts total:1/present:0/synced:0/available:0
    And observed phase "phase-2" should have object details for "cm-pending-2"

  Scenario: completedAt is preserved through regression
    Given a COS with group "ps-regress" and revision 1
    And a phase "install" with a gated ConfigMap "cm-regress"
    When the COS is created
    And the gate on ConfigMap "cm-regress" is opened
    Then the COS should have condition "Available" with status "True"
    And the COS should have completedAt set
    And the COS completedAt is tracked
    When the gate on ConfigMap "cm-regress" is closed
    Then the COS should have condition "Available" with status "False"
    And the COS should have observed phase "install" with status "WaitingForAssertions"
    And observed phase "install" should have object counts total:1/present:1/synced:1/available:0
    And the COS completedAt should be preserved

  Scenario: Archived COS shows all phases TeardownComplete after teardown and keeps completedAt
    Given an available COS with group "ps-archive" and revision 1
    And the COS should have completedAt set
    And the COS completedAt is tracked
    When the COS lifecycleState is set to "Archived"
    Then the COS should have condition "Available" with status "False" and reason "Archived"
    And the COS should have observed phase "install" with status "TeardownComplete"
    And observed phase "install" should have object counts total:1/present:0/synced:0/available:0
    And the COS should have object counts total:1/present:0/synced:0/available:0
    And the COS completedAt should be preserved

  Scenario: Archived COS shows TeardownComplete, TearingDown, and Pending during teardown
    Given a COS with group "ps-td-archive" and revision 1
    And a phase "phase-1" with a ConfigMap "cm-td-archive-1"
    And a phase "phase-2" with a ConfigMap "cm-td-archive-2"
    And a phase "phase-3" with a ConfigMap "cm-td-archive-3"
    When the COS is created and becomes Available
    And a finalizer "e2e.orb.dev/block" is added to ConfigMap "cm-td-archive-2"
    And the COS lifecycleState is set to "Archived"
    Then the COS should have observed phase "phase-3" with status "TeardownComplete"
    And observed phase "phase-3" should have object counts total:1/present:0/synced:0/available:0
    And the COS should have observed phase "phase-2" with status "TearingDown"
    And observed phase "phase-2" should have object details for "cm-td-archive-2"
    And observed phase "phase-2" should have object counts total:1/present:1/synced:0/available:0
    And the COS should have observed phase "phase-1" with status "Pending" and message "Waiting for later phases to complete teardown"
    And observed phase "phase-1" should have object counts total:1/present:1/synced:0/available:0
    And the COS should have object counts total:3/present:2/synced:0/available:0
    When the finalizer "e2e.orb.dev/block" is removed from ConfigMap "cm-td-archive-2"
    Then the COS should have condition "Available" with status "False" and reason "Archived"
    And the COS should have observed phase "phase-1" with status "TeardownComplete"
    And observed phase "phase-1" should have object counts total:1/present:0/synced:0/available:0
    And the COS should have observed phase "phase-2" with status "TeardownComplete"
    And observed phase "phase-2" should have object counts total:1/present:0/synced:0/available:0
    And the COS should have observed phase "phase-3" with status "TeardownComplete"
    And observed phase "phase-3" should have object counts total:1/present:0/synced:0/available:0
    And the COS should have object counts total:3/present:0/synced:0/available:0

  Scenario: Deleted COS shows TeardownComplete, TearingDown, and Pending during teardown
    Given a COS with group "ps-td-delete" and revision 1
    And a phase "phase-1" with a ConfigMap "cm-td-delete-1"
    And a phase "phase-2" with a ConfigMap "cm-td-delete-2"
    And a phase "phase-3" with a ConfigMap "cm-td-delete-3"
    When the COS is created and becomes Available
    And a finalizer "e2e.orb.dev/block" is added to ConfigMap "cm-td-delete-2"
    And the COS is deleted
    Then the COS should have observed phase "phase-3" with status "TeardownComplete"
    And observed phase "phase-3" should have object counts total:1/present:0/synced:0/available:0
    And the COS should have observed phase "phase-2" with status "TearingDown"
    And observed phase "phase-2" should have object details for "cm-td-delete-2"
    And observed phase "phase-2" should have object counts total:1/present:1/synced:0/available:0
    And the COS should have observed phase "phase-1" with status "Pending" and message "Waiting for later phases to complete teardown"
    And observed phase "phase-1" should have object counts total:1/present:1/synced:0/available:0
    And the COS should have object counts total:3/present:2/synced:0/available:0
    When the finalizer "e2e.orb.dev/block" is removed from ConfigMap "cm-td-delete-2"
    Then the COS should not exist

  Scenario: Archived COS reports teardown error in condition
    Given an available COS with group "ps-td-err" and revision 1
    And ConfigMap operations are blocked
    When the COS lifecycleState is set to "Archived"
    Then the COS should have condition "Available" with status "Unknown" and reason "TeardownError"
    And the COS should have no observed phases

  Scenario: Deleted COS reports teardown error in condition
    Given an available COS with group "ps-td-del-err" and revision 1
    And ConfigMap operations are blocked
    When the COS is deleted
    Then the COS should have condition "Available" with status "Unknown" and reason "TeardownError"
    And the COS should have no observed phases

  Scenario: InvalidRevision when same object appears in multiple phases
    Given a COS with group "ps-dup" and revision 1
    And a phase "phase-1" with a ConfigMap "cm-dup"
    And a phase "phase-2" with a ConfigMap "cm-dup"
    When the COS is created
    Then the COS should have condition "Available" with status "False" and reason "InvalidRevision" and message containing "duplicate object found in phases"
    And the COS should have observed phase "phase-1" with status "Invalid"
    And observed phase "phase-1" should have object details for "cm-dup"
    And the COS should have observed phase "phase-2" with status "Invalid"
    And observed phase "phase-2" should have object details for "cm-dup"
    And the ConfigMap "cm-dup" should not exist

  Scenario: Superseded COS shows all phases as Superseded
    Given an available COS with group "ps-super" and revision 1
    And a COS with group "ps-super" and revision 2
    And a phase "install" with a ConfigMap "cm-ps-super"
    When the COS is created and becomes Available
    Then revision 1 should have condition "Available" with status "False" and reason "Superseded"
    And revision 1 should have observed phase "install" with status "Superseded"

  Scenario: Completed phase gets drift correction, in-progress phase keeps waiting, later phases stay pending
    Given a COS with group "ps-drift" and revision 1
    And a phase "phase-1" with a gated ConfigMap "cm-drift-1"
    And a phase "phase-2" with a ConfigMap "cm-drift-2"
    And a phase "phase-3" with a gated ConfigMap "cm-drift-3"
    And a phase "phase-4" with a ConfigMap "cm-drift-4"
    When the COS is created
    And the gate on ConfigMap "cm-drift-1" is opened
    Then the COS should have observed phase "phase-1" with status "Available"
    And observed phase "phase-1" should have object counts total:1/present:1/synced:1/available:1
    And the COS should have observed phase "phase-2" with status "Available"
    And observed phase "phase-2" should have object counts total:1/present:1/synced:1/available:1
    And the COS should have observed phase "phase-3" with status "WaitingForAssertions"
    And observed phase "phase-3" should have object counts total:1/present:1/synced:1/available:0
    And the COS should have observed phase "phase-4" with status "Pending" and message "Waiting for earlier phases to complete"
    And observed phase "phase-4" should have object counts total:1/present:0/synced:0/available:0
    When the gate on ConfigMap "cm-drift-1" is closed
    Then the COS should have condition "Available" with status "False"
    And the COS should have observed phase "phase-1" with status "WaitingForAssertions"
    And observed phase "phase-1" should have object counts total:1/present:1/synced:1/available:0
    And the COS should have observed phase "phase-2" with status "Available"
    And observed phase "phase-2" should have object counts total:1/present:1/synced:1/available:1
    And the COS should have observed phase "phase-3" with status "WaitingForAssertions"
    And observed phase "phase-3" should have object counts total:1/present:1/synced:1/available:0
    And the COS should have observed phase "phase-4" with status "Pending" and message "Waiting for earlier phases to complete"
    And observed phase "phase-4" should have object counts total:1/present:0/synced:0/available:0
    And the ConfigMap "cm-drift-2" should exist
    When the ConfigMap "cm-drift-2" is deleted
    Then the ConfigMap "cm-drift-2" should be recreated

  Scenario: InvalidRevision with mixed phase errors shows Invalid and Unknown
    Given a COS with group "ps-mixed" and revision 1
    And a phase "good-phase" with a ConfigMap "cm-mixed-good"
    And a phase "bad-phase-1" with a ConfigMap "cm-mixed-dup"
    And a phase "bad-phase-2" with a ConfigMap "cm-mixed-dup"
    When the COS is created
    Then the COS should have condition "Available" with status "False" and reason "InvalidRevision"
    And the COS should have observed phase "good-phase" with status "Unknown" and message "Blocked by preflight errors in other phases"
    And the COS should have observed phase "bad-phase-1" with status "Invalid"
    And the COS should have observed phase "bad-phase-2" with status "Invalid"

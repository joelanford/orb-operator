Feature: COSR phase status reports per-phase rollout state

  Scenario: Phase status shows progression from Unknown to Reconciling to Complete
    Given a COSR with group "ps" and revision 1
    And a phase "crds" with a gated ConfigMap "cm-crds"
    And a phase "operators" with a gated ConfigMap "cm-operators"
    And a phase "config" with a gated ConfigMap "cm-config"
    When the COSR is created
    Then the COSR should have 3 observed phases
    And the COSR should have observed phase "crds" with status "Reconciling"
    And the COSR should have observed phase "operators" with status "Unknown"
    And the COSR should have observed phase "config" with status "Unknown"
    And the COSR should not have completedAt set
    When the gate on ConfigMap "cm-crds" is opened
    Then the COSR should have observed phase "crds" with status "Available"
    And the COSR should have observed phase "operators" with status "Reconciling"
    And the COSR should have observed phase "config" with status "Unknown"
    When the gate on ConfigMap "cm-operators" is opened
    And the gate on ConfigMap "cm-config" is opened
    Then the COSR should have condition "Available" with status "True"
    And the COSR should have observed phase "crds" with status "Available"
    And the COSR should have observed phase "operators" with status "Available"
    And the COSR should have observed phase "config" with status "Available"
    And the COSR should have completedAt set

  Scenario: Incomplete objects listed in Reconciling phases
    Given a COSR with group "ps-unavail" and revision 1
    And a phase "install" with a gated ConfigMap "cm-unavail"
    When the COSR is created
    Then the COSR should have observed phase "install" with status "Reconciling"
    And observed phase "install" should have 1 incomplete objects

  Scenario: completedAt is preserved through regression
    Given a COSR with group "ps-regress" and revision 1
    And a phase "install" with a gated ConfigMap "cm-regress"
    When the COSR is created
    And the gate on ConfigMap "cm-regress" is opened
    Then the COSR should have condition "Available" with status "True"
    And the COSR should have completedAt set
    And the COSR completedAt is tracked
    When the gate on ConfigMap "cm-regress" is closed
    Then the COSR should have condition "Available" with status "False"
    And the COSR should have observed phase "install" with status "Reconciling"
    And the COSR completedAt should be preserved

  Scenario: Archived COSR shows all phases TeardownComplete after teardown and keeps completedAt
    Given an available COSR with group "ps-archive" and revision 1
    And the COSR should have completedAt set
    And the COSR completedAt is tracked
    When the COSR lifecycleState is set to "Archived"
    Then the COSR should have condition "Available" with status "False" and reason "Archived"
    And the COSR should have observed phase "install" with status "TeardownComplete"
    And the COSR completedAt should be preserved

  Scenario: Archived COSR shows TeardownComplete, TearingDown, and Unknown during teardown
    Given a COSR with group "ps-td-archive" and revision 1
    And a phase "phase-1" with a ConfigMap "cm-td-archive-1"
    And a phase "phase-2" with a ConfigMap "cm-td-archive-2"
    And a phase "phase-3" with a ConfigMap "cm-td-archive-3"
    When the COSR is created and becomes Available
    And a finalizer "e2e.orb.dev/block" is added to ConfigMap "cm-td-archive-2"
    And the COSR lifecycleState is set to "Archived"
    Then the COSR should have observed phase "phase-3" with status "TeardownComplete"
    And the COSR should have observed phase "phase-2" with status "TearingDown"
    And observed phase "phase-2" should have an incomplete object "cm-td-archive-2"
    And the COSR should have observed phase "phase-1" with status "Unknown"
    When the finalizer "e2e.orb.dev/block" is removed from ConfigMap "cm-td-archive-2"
    Then the COSR should have condition "Available" with status "False" and reason "Archived"
    And the COSR should have observed phase "phase-1" with status "TeardownComplete"
    And the COSR should have observed phase "phase-2" with status "TeardownComplete"
    And the COSR should have observed phase "phase-3" with status "TeardownComplete"

  Scenario: Deleted COSR shows TeardownComplete, TearingDown, and Unknown during teardown
    Given a COSR with group "ps-td-delete" and revision 1
    And a phase "phase-1" with a ConfigMap "cm-td-delete-1"
    And a phase "phase-2" with a ConfigMap "cm-td-delete-2"
    And a phase "phase-3" with a ConfigMap "cm-td-delete-3"
    When the COSR is created and becomes Available
    And a finalizer "e2e.orb.dev/block" is added to ConfigMap "cm-td-delete-2"
    And the COSR is deleted
    Then the COSR should have observed phase "phase-3" with status "TeardownComplete"
    And the COSR should have observed phase "phase-2" with status "TearingDown"
    And observed phase "phase-2" should have an incomplete object "cm-td-delete-2"
    And the COSR should have observed phase "phase-1" with status "Unknown"
    When the finalizer "e2e.orb.dev/block" is removed from ConfigMap "cm-td-delete-2"
    Then the COSR should not exist

  Scenario: Archived COSR reports teardown error in condition
    Given ConfigMap deletes are blocked in the test namespace
    And an available COSR with group "ps-td-err" and revision 1
    When the COSR lifecycleState is set to "Archived"
    Then the COSR should have condition "Available" with status "Unknown" and reason "TeardownError"
    And the COSR should have no observed phases

  Scenario: Deleted COSR reports teardown error in condition
    Given ConfigMap deletes are blocked in the test namespace
    And an available COSR with group "ps-td-del-err" and revision 1
    When the COSR is deleted
    Then the COSR should have condition "Available" with status "Unknown" and reason "TeardownError"
    And the COSR should have no observed phases

  Scenario: Superseded COSR shows all phases as Superseded
    Given an available COSR with group "ps-super" and revision 1
    And a COSR with group "ps-super" and revision 2
    And a phase "install" with a ConfigMap "cm-ps-super"
    When the COSR is created and becomes Available
    Then revision 1 should have condition "Available" with status "False" and reason "Superseded"
    And revision 1 should have observed phase "install" with status "Superseded"

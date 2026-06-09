Feature: COSR status conditions reflect rollout state

  Scenario: COSR becomes Available after all phases complete
    Given a COSR with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-status"
    When the COSR is created
    Then the COSR should have condition "Available" with status "True"
    And the COSR should have observed phase "install" with status "Available"

  Scenario: COSR is not Available while phases are incomplete
    Given a COSR with group "test" and revision 1
    And a phase "deploy" with a gated ConfigMap "cm-blocked"
    When the COSR is created
    Then the COSR should have condition "Available" with status "False" and reason "Unavailable"
    And the COSR should have observed phase "deploy" with status "Reconciling"

  Scenario: COSR reports ReconcileError when reconcile fails with an error
    Given a COSR with group "test" and revision 1
    And a phase "install" with an unregistered resource type
    When the COSR is created
    Then the COSR should have condition "Available" with status "Unknown" and reason "ReconcileError"
    And the COSR should have observed phase "install" with status "Unknown"

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

  Scenario: Reconcile error on one object does not prevent reconciling other objects in the same phase
    Given ConfigMap "cm-blocked" operations are blocked
    And a COSR with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-before-err"
    And the phase "install" also has a ConfigMap "cm-blocked"
    And the phase "install" also has a ConfigMap "cm-after-err"
    When the COSR is created
    Then the COSR should have condition "Available" with status "Unknown" and reason "ReconcileError"
    And the ConfigMap "cm-before-err" should exist
    And the ConfigMap "cm-blocked" should not exist
    And the ConfigMap "cm-after-err" should exist

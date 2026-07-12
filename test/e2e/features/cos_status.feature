Feature: COS status conditions reflect rollout state

  Scenario: COS becomes Available after all phases complete
    Given a COS with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-status"
    When the COS is created
    Then the COS should have condition "Available" with status "True"
    And the COS should have observed phase "install" with status "Available"

  Scenario: COS is not Available while phases are incomplete
    Given a COS with group "test" and revision 1
    And a phase "deploy" with a gated ConfigMap "cm-blocked"
    When the COS is created
    Then the COS should have condition "Available" with status "False" and reason "Unavailable"
    And the COS should have observed phase "deploy" with status "WaitingForAssertions"

  Scenario: COS with unregistered resource type shows validation error
    Given a COS with group "test" and revision 1
    And a phase "install" with an unregistered resource type
    When the COS is created
    Then the COS should have condition "Available" with status "False" and reason "Unavailable"
    And the COS should have observed phase "install" with status "Invalid"
    And observed phase "install" should have object details for "fake-resource"

  Scenario: Reconcile error on one object does not prevent reconciling other objects in the same phase
    Given ConfigMap "cm-blocked" operations are blocked
    And a COS with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-before-err"
    And the phase "install" also has a ConfigMap "cm-blocked"
    And the phase "install" also has a ConfigMap "cm-after-err"
    When the COS is created
    Then the COS should have condition "Available" with status "Unknown" and reason "ReconcileError"
    And the ConfigMap "cm-before-err" should exist
    And the ConfigMap "cm-blocked" should not exist
    And the ConfigMap "cm-after-err" should exist

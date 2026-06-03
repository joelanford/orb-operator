Feature: COSR status conditions reflect rollout state

  Scenario: COSR becomes Available after all phases complete
    Given a COSR with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-status"
    When the COSR is created
    And all phases complete successfully
    Then the COSR should have condition "Available" with status "True"

  Scenario: COSR is not Available while phases are incomplete
    Given a COSR with group "test" and revision 1
    And a phase "deploy" with a ConfigMap "cm-blocked" with assertion fieldValue path ".data.ready" value "true"
    When the COSR is created
    Then the COSR should have condition "Available" with status "False" and reason "Unavailable"

  Scenario: COSR reports Unavailable when reconcile fails with an error
    Given a COSR with group "test" and revision 1
    And a phase "install" with an unregistered resource type
    When the COSR is created
    Then the COSR should have condition "Available" with status "False" and reason "Unavailable"

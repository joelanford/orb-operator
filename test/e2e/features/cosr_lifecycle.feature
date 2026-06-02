Feature: COSR Active and Archived lifecycle behavior

  Scenario: Active COSR recreates a deleted managed object
    Given a COSR with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-recreate"
    And the COSR is created and becomes Available
    When the ConfigMap "cm-recreate" is deleted
    Then the ConfigMap "cm-recreate" should be recreated

  Scenario: Available condition flaps as per-phase assertions fail and recover
    Given a COSR with group "test" and revision 1
    And a phase "p1" with a ConfigMap "cm-p1" with assertion celExpression "!has(object.data) || object.data['fail'] != 'true'"
    And a phase "p2" with a ConfigMap "cm-p2" with assertion celExpression "!has(object.data) || object.data['fail'] != 'true'"
    And a phase "p3" with a ConfigMap "cm-p3" with assertion celExpression "!has(object.data) || object.data['fail'] != 'true'"
    # All probes pass by default — COSR becomes Available
    When the COSR is created
    Then the COSR should have condition "Available" with status "True"
    # Phase 1 probe fails
    When the ConfigMap "cm-p1" field ".data.fail" is set to "true"
    Then the COSR should have condition "Available" with status "False"
    # Phase 1 probe recovers
    When the ConfigMap "cm-p1" field ".data.fail" is set to "false"
    Then the COSR should have condition "Available" with status "True"
    # Phase 2 probe fails
    When the ConfigMap "cm-p2" field ".data.fail" is set to "true"
    Then the COSR should have condition "Available" with status "False"
    # Phase 2 probe recovers
    When the ConfigMap "cm-p2" field ".data.fail" is set to "false"
    Then the COSR should have condition "Available" with status "True"
    # Phase 3 probe fails
    When the ConfigMap "cm-p3" field ".data.fail" is set to "true"
    Then the COSR should have condition "Available" with status "False"
    # Phase 3 probe recovers
    When the ConfigMap "cm-p3" field ".data.fail" is set to "false"
    Then the COSR should have condition "Available" with status "True"

  Scenario: COSR recovers after managed object deletion
    Given a COSR with group "test" and revision 1
    And a phase "p1" with a ConfigMap "cm-p1"
    And a phase "p2" with a ConfigMap "cm-p2"
    And a phase "p3" with a ConfigMap "cm-p3"
    And the COSR is created and becomes Available
    # Phase 1 object deleted
    When the ConfigMap "cm-p1" is deleted
    Then the COSR should have condition "Available" with status "False" and reason "Unavailable"
    When the ConfigMap "cm-p1" is recreated by the controller
    Then the COSR should have condition "Available" with status "True"
    # Phase 2 object deleted
    When the ConfigMap "cm-p2" is deleted
    Then the COSR should have condition "Available" with status "False" and reason "Unavailable"
    When the ConfigMap "cm-p2" is recreated by the controller
    Then the COSR should have condition "Available" with status "True"
    # Phase 3 object deleted
    When the ConfigMap "cm-p3" is deleted
    Then the COSR should have condition "Available" with status "False" and reason "Unavailable"
    When the ConfigMap "cm-p3" is recreated by the controller
    Then the COSR should have condition "Available" with status "True"

  Scenario: Archived COSR cannot be unarchived
    Given a COSR with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-no-reactivate"
    And the COSR is created and becomes Available
    When the COSR lifecycleState is set to "Archived"
    Then setting the COSR lifecycleState to "Active" should fail

  Scenario: Archiving a single-revision COSR deletes managed objects
    Given a COSR with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-archive"
    And the COSR is created and becomes Available
    When the COSR lifecycleState is set to "Archived"
    Then the ConfigMap "cm-archive" should not exist
    And the COSR should have condition "Available" with status "False" and reason "Archived"

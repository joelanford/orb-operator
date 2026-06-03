Feature: COSR multi-revision ownership handoffs within a group

  Scenario: New revision supersedes the old revision
    Given a COSR with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-shared"
    And the COSR is created and becomes Available
    When a COSR with group "test" and revision 2 is created
    And the phase "install" has a ConfigMap "cm-shared"
    And the new COSR is created
    Then revision 1 should have condition "Available" with status "False" and reason "Superseded"

  Scenario: Shared objects transfer ownership without recreation
    Given a COSR with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-transfer"
    And the COSR is created and becomes Available
    And the ConfigMap "cm-transfer" UID is tracked
    When a COSR with group "test" and revision 2 is created
    And the phase "install" has a ConfigMap "cm-transfer"
    And the new COSR is created and becomes Available
    Then the ConfigMap "cm-transfer" should exist
    And the ConfigMap "cm-transfer" should not have been deleted and recreated

  Scenario: Revision transition deletes old objects, updates shared objects, and creates new objects
    Given a COSR with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-old"
    And the phase "install" also has a ConfigMap "cm-shared" with data:
      | key    | value |
      | keep   | old   |
      | remove | gone  |
    And the COSR is created and becomes Available
    When a COSR with group "test" and revision 2 is created
    And the phase "install" has a ConfigMap "cm-shared" with data:
      | key  | value |
      | keep | new   |
      | add  | fresh |
    And the phase "install" also has a ConfigMap "cm-new"
    And the new COSR is created and becomes Available
    Then revision 1 should have condition "Available" with status "False" and reason "Archived"
    And the ConfigMap "cm-old" should not exist
    And the ConfigMap "cm-shared" should have data key "keep" with value "new"
    And the ConfigMap "cm-shared" should not have data key "remove"
    And the ConfigMap "cm-shared" should have data key "add" with value "fresh"
    And the ConfigMap "cm-new" should exist

  Scenario: Superseded revision stays superseded while newer revision is unavailable
    Given a COSR with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-stuck"
    And the COSR is created and becomes Available
    When a COSR with group "test" and revision 2 is created
    And a phase "install" with a ConfigMap "cm-stuck" with assertion fieldValue path ".data.ready" value "yes"
    And the new COSR is created
    Then revision 1 should have condition "Available" with status "False" and reason "Superseded"
    And revision 2 should have condition "Available" with status "False" and reason "Unavailable"

  Scenario: Non-contiguous revision numbers work correctly
    Given a COSR with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-gap"
    And the COSR is created and becomes Available
    When a COSR with group "test" and revision 5 is created
    And the phase "install" has a ConfigMap "cm-gap"
    And the new COSR is created and becomes Available
    Then revision 1 should have condition "Available" with status "False" and reason "Archived"

  Scenario: Old revision is archived after new revision succeeds
    Given a COSR with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-archive-test"
    And the COSR is created and becomes Available
    When a COSR with group "test" and revision 2 is created
    And the phase "install" has a ConfigMap "cm-archive-test"
    And the new COSR is created and becomes Available
    Then revision 1 should have condition "Available" with status "False" and reason "Archived"

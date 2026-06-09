Feature: COSR multi-revision ownership handoffs within a group

  Scenario: New revision supersedes the old revision
    Given a COSR with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-shared"
    And the COSR is created and becomes Available
    When a COSR with group "test" and revision 2
    And the phase "install" has a ConfigMap "cm-shared"
    And the COSR is created
    Then revision 1 should have condition "Available" with status "False" and reason "Superseded"

  Scenario: Shared objects transfer ownership without recreation
    Given a COSR with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-transfer"
    And the COSR is created and becomes Available
    And the ConfigMap "cm-transfer" UID is tracked
    When a COSR with group "test" and revision 2
    And the phase "install" has a ConfigMap "cm-transfer"
    And the COSR is created and becomes Available
    Then the ConfigMap "cm-transfer" should exist
    And the ConfigMap "cm-transfer" should not have been deleted and recreated

  Scenario: Superseded revision stays superseded while newer revision is unavailable
    Given a COSR with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-stuck"
    And the COSR is created and becomes Available
    When a COSR with group "test" and revision 2
    And a phase "install" with a gated ConfigMap "cm-stuck"
    And the COSR is created
    Then revision 1 should have condition "Available" with status "False" and reason "Superseded"
    And revision 2 should have condition "Available" with status "False" and reason "Unavailable"

  Scenario: Non-contiguous revision numbers work correctly
    Given a COSR with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-gap"
    And the COSR is created and becomes Available
    When a COSR with group "test" and revision 5
    And the phase "install" has a ConfigMap "cm-gap"
    And the COSR is created and becomes Available
    Then revision 1 should have condition "Available" with status "False" and reason "Superseded"

  Scenario: Old revision is superseded after new revision succeeds
    Given a COSR with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-archive-test"
    And the COSR is created and becomes Available
    When a COSR with group "test" and revision 2
    And the phase "install" has a ConfigMap "cm-archive-test"
    And the COSR is created and becomes Available
    Then revision 1 should have condition "Available" with status "False" and reason "Superseded"

  Scenario: Superseded COSR retains objects not present in the new revision
    Given a COSR with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-old-only"
    And the phase "install" also has a ConfigMap "cm-shared"
    And the COSR is created and becomes Available
    When a COSR with group "test" and revision 2
    And the phase "install" has a ConfigMap "cm-shared"
    And the phase "install" also has a ConfigMap "cm-new-only"
    And the COSR is created and becomes Available
    Then revision 1 should have condition "Available" with status "False" and reason "Superseded"
    And the ConfigMap "cm-old-only" should exist
    And the ConfigMap "cm-old-only" should have an owner reference
    And the ConfigMap "cm-shared" should exist
    And the ConfigMap "cm-new-only" should exist

  Scenario: Superseded COSR should recreate its deleted unique object
    Given a COSR with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-old-only"
    And the phase "install" also has a ConfigMap "cm-shared"
    And the COSR is created and becomes Available
    When a COSR with group "test" and revision 2
    And the phase "install" has a ConfigMap "cm-shared"
    And the phase "install" also has a ConfigMap "cm-new-only"
    And the COSR is created and becomes Available
    Then revision 1 should have condition "Available" with status "False" and reason "Superseded"
    When the ConfigMap "cm-old-only" is deleted
    Then the ConfigMap "cm-old-only" should be recreated

  Scenario: Superseded COSR with disjoint GVKs recreates deleted object
    Given a COSR with group "test" and revision 1
    And a phase "crds" with a CRD "widgets" with assertion conditionEqual type "Established" status "True"
    And a phase "instances" with a "widgets" named "w1"
    And the COSR is created and becomes Available
    When a COSR with group "test" and revision 2
    And a phase "install" with a ConfigMap "cm-rev2"
    And the COSR is created and becomes Available
    Then revision 1 should have condition "Available" with status "False" and reason "Superseded"
    And the "widgets" named "w1" should exist
    When the CRD "widgets" is deleted
    Then the CRD "widgets" should exist

Feature: COSR multi-revision ownership handoffs within a group

  Scenario: Partially overlapping revisions transfer shared objects and retain unique objects
    Given a COSR with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-old-only"
    And the phase "install" also has a ConfigMap "cm-shared"
    And the COSR is created and becomes Available
    When a COSR with group "test" and revision 2
    And the phase "install" has a ConfigMap "cm-shared"
    And the phase "install" also has a ConfigMap "cm-new-only"
    And the COSR is created and becomes Available
    Then the ConfigMap "cm-old-only" should have a controller owner reference to COSR with group "test" and revision 1
    And the ConfigMap "cm-shared" should have a controller owner reference to COSR with group "test" and revision 2
    And the ConfigMap "cm-new-only" should have a controller owner reference to COSR with group "test" and revision 2
    And revision 1 should have condition "Available" with status "True" and reason "Available"
    And revision 2 should have condition "Available" with status "True" and reason "Available"

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

  Scenario: Non-contiguous revision numbers work correctly
    Given a COSR with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-gap"
    And the COSR is created and becomes Available
    When a COSR with group "test" and revision 5
    And the phase "install" has a ConfigMap "cm-gap"
    And the COSR is created and becomes Available
    Then the ConfigMap "cm-gap" should have a controller owner reference to COSR with group "test" and revision 5

  Scenario: Fully overlapping revisions supersede the predecessor
    Given a COSR with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-shared"
    And the COSR is created and becomes Available
    When a COSR with group "test" and revision 2
    And the phase "install" has a ConfigMap "cm-shared"
    And the COSR is created and becomes Available
    Then the ConfigMap "cm-shared" should have a controller owner reference to COSR with group "test" and revision 2
    And revision 1 should have condition "Available" with status "False" and reason "Superseded"

  Scenario: Predecessor COSR recreates its deleted unique object
    Given a COSR with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-old-only"
    And the phase "install" also has a ConfigMap "cm-shared"
    And the COSR is created and becomes Available
    When a COSR with group "test" and revision 2
    And the phase "install" has a ConfigMap "cm-shared"
    And the phase "install" also has a ConfigMap "cm-new-only"
    And the COSR is created and becomes Available
    Then the ConfigMap "cm-old-only" should have a controller owner reference to COSR with group "test" and revision 1
    When the ConfigMap "cm-old-only" is deleted
    Then the ConfigMap "cm-old-only" should be recreated

  Scenario: Revisions with disjoint objects in the same group independently manage their own objects
    Given a COSR with group "test" and revision 1
    And a phase "crds" with a CRD "widgets" with assertion conditionEqual type "Established" status "True"
    And the COSR is created and becomes Available
    When a COSR with group "test" and revision 2
    And a phase "install" with a ConfigMap "cm-rev2"
    And the COSR is created and becomes Available
    Then revision 1 should have condition "Available" with status "True" and reason "Available"
    And revision 2 should have condition "Available" with status "True" and reason "Available"
    When the CRD "widgets" is deleted
    Then the CRD "widgets" should exist
    When the ConfigMap "cm-rev2" is deleted
    Then the ConfigMap "cm-rev2" should be recreated

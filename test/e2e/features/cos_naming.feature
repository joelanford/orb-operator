Feature: COS naming policy enforcement

  Scenario: COS name is group-revision
    Given an available COS with group "naming-pos" and revision 1
    Then the COS with group "naming-pos" and revision 1 should be named "naming-pos-1"

  Scenario: COS name must match group-revision
    Given a COS named "wrong-name" with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-naming"
    Then creating the COS should fail

  Scenario: Duplicate group and revision pair is rejected
    Given an available COS with group "test" and revision 1
    Given a COS with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-duplicate"
    Then creating the COS should fail

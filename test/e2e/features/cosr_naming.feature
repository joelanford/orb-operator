Feature: COSR naming policy enforcement

  Scenario: COSR name is group-revision
    Given an available COSR with group "naming-pos" and revision 1
    Then the COSR with group "naming-pos" and revision 1 should be named "naming-pos-1"

  Scenario: COSR name must match group-revision
    Given a COSR named "wrong-name" with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-naming"
    Then creating the COSR should fail

  Scenario: Duplicate group and revision pair is rejected
    Given an available COSR with group "test" and revision 1
    Given a COSR with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-duplicate"
    Then creating the COSR should fail

Feature: COSR naming policy enforcement

  Scenario: COSR name must match group and revision
    Given a COSR named "wrong-name" with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-naming"
    Then creating the COSR should fail

  Scenario: Duplicate group and revision pair is rejected
    Given a COSR with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-first"
    And the COSR is created and becomes Available
    Given a COSR with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-duplicate"
    Then creating the COSR should fail

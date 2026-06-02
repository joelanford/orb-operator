Feature: COSR phases execute in order, gated by assertions

  Scenario: Multi-phase COSR rolls out phases sequentially
    Given a COSR with group "test" and revision 1
    And a phase "gate" with a ConfigMap "cm-gate" with assertion fieldValue path ".data.ready" value "true"
    And a phase "app" with a ConfigMap "cm-app"
    When the COSR is created
    Then the COSR should have condition "Available" with status "False"
    And the ConfigMap "cm-app" should not exist
    When the ConfigMap "cm-gate" field ".data.ready" is set to "true"
    Then the COSR should have condition "Available" with status "True"
    And the ConfigMap "cm-app" should exist

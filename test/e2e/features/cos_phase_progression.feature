Feature: COS phases execute in order, gated by assertions

  Scenario: Multi-phase COS rolls out phases sequentially
    Given a COS with group "test" and revision 1
    And a phase "gate" with a gated ConfigMap "cm-gate"
    And a phase "app" with a ConfigMap "cm-app"
    When the COS is created
    Then the COS should have condition "Available" with status "False"
    And the ConfigMap "cm-app" should not exist
    When the gate on ConfigMap "cm-gate" is opened
    Then the COS should have condition "Available" with status "True"
    And the ConfigMap "cm-app" should exist

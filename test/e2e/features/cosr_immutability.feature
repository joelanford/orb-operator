Feature: COSR spec immutability

  All spec fields except lifecycleState are immutable after creation.

  Scenario: Updating group is rejected
    Given a COSR with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-immutable"
    And the COSR is created and becomes Available
    Then updating the COSR group should fail

  Scenario: Updating revision is rejected
    Given a COSR with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-immutable"
    And the COSR is created and becomes Available
    Then updating the COSR revision should fail

  Scenario: Updating phases is rejected
    Given a COSR with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-immutable"
    And the COSR is created and becomes Available
    Then updating the COSR phases should fail

  Scenario: Updating collisionProtection is rejected
    Given a COSR with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-immutable"
    And the COSR is created and becomes Available
    Then updating the COSR collisionProtection should fail

Feature: COSR spec immutability

  All spec fields except lifecycleState are immutable after creation.

  Scenario: Updating group is rejected
    Given an available COSR with group "test" and revision 1
    Then updating the COSR group should fail

  Scenario: Updating revision is rejected
    Given an available COSR with group "test" and revision 1
    Then updating the COSR revision should fail

  Scenario: Updating phases is rejected
    Given an available COSR with group "test" and revision 1
    Then updating the COSR phases should fail

  Scenario: Updating collisionProtection is rejected
    Given an available COSR with group "test" and revision 1
    Then updating the COSR collisionProtection should fail

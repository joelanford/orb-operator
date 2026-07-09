Feature: COS spec immutability

  All spec fields except lifecycleState are immutable after creation.

  Scenario: Updating group is rejected
    Given an available COS with group "test" and revision 1
    Then updating the COS group should fail

  Scenario: Updating revision is rejected
    Given an available COS with group "test" and revision 1
    Then updating the COS revision should fail

  Scenario: Updating phases is rejected
    Given an available COS with group "test" and revision 1
    Then updating the COS phases should fail

  Scenario: Updating collisionProtection is rejected
    Given an available COS with group "test" and revision 1
    Then updating the COS collisionProtection should fail

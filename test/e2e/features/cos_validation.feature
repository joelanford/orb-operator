Feature: COS structural validation

  Scenario: COS with zero phases is rejected
    Then creating a COS with zero phases should fail

  Scenario: COS with a phase containing zero objects is rejected
    Then creating a COS with a phase with zero objects should fail

  Scenario: COS with revision 0 is rejected
    Then creating a COS with revision 0 should fail

  Scenario: COS with a group name of exactly 52 characters is accepted
    Then creating a COS with a group name of exactly 52 characters should succeed

  Scenario: COS with a group name longer than 52 characters is rejected
    Then creating a COS with a group name longer than 52 characters should fail

  Scenario: COS with unset lifecycleState is rejected
    Then creating a COS with unset lifecycleState should fail

  Scenario: COS with unknown lifecycleState is rejected
    Then creating a COS with unknown lifecycleState should fail

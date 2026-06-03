Feature: COSR structural validation

  Scenario: COSR with zero phases is rejected
    Then creating a COSR with zero phases should fail

  Scenario: COSR with a phase containing zero objects is rejected
    Then creating a COSR with a phase with zero objects should fail

  Scenario: COSR with revision 0 is rejected
    Then creating a COSR with revision 0 should fail

  Scenario: COSR with a group name of exactly 52 characters is accepted
    Then creating a COSR with a group name of exactly 52 characters should succeed

  Scenario: COSR with a group name longer than 52 characters is rejected
    Then creating a COSR with a group name longer than 52 characters should fail

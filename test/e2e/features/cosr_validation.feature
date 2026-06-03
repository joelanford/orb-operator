Feature: COSR structural validation

  Scenario: COSR with zero phases is rejected
    Then creating a COSR with zero phases should fail

  Scenario: COSR with a phase containing zero objects is rejected
    Then creating a COSR with a phase with zero objects should fail

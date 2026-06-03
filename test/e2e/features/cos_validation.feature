Feature: COS structural validation

  Scenario: COS with a name of exactly 52 characters is accepted
    Then creating a COS with a name of exactly 52 characters should succeed

  Scenario: COS with a name longer than 52 characters is rejected
    Then creating a COS with a name longer than 52 characters should fail

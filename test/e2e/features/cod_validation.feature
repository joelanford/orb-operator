Feature: COD structural validation

  Scenario: COD with a name of exactly 52 characters is accepted
    Then creating a COD with a name of exactly 52 characters should succeed

  Scenario: COD with a name longer than 52 characters is rejected
    Then creating a COD with a name longer than 52 characters should fail

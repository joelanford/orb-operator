Feature: COS assertion type evaluation

  Scenario: ConditionEqual assertion passes
    Given a COS with group "test" and revision 1
    And a phase "install" with a CRD "testwidgets" with assertion conditionEqual type "Established" status "True"
    When the COS is created
    Then the COS should have condition "Available" with status "True"

  Scenario: FieldsEqual assertion passes
    Given a COS with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-fields" with data:
      | key | value |
      | a   | same  |
      | b   | same  |
    And the last object has assertion fieldsEqual fieldA ".data.a" fieldB ".data.b"
    When the COS is created
    Then the COS should have condition "Available" with status "True"

  Scenario: FieldValue assertion passes
    Given a COS with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-fv" with data key "ready" value "true"
    And the last object has assertion fieldValue path ".data.ready" value "true"
    When the COS is created
    Then the COS should have condition "Available" with status "True"

  Scenario: CEL expression assertion passes
    Given a COS with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-cel" with assertion celExpression "self.metadata.name == 'cm-cel'"
    When the COS is created
    Then the COS should have condition "Available" with status "True"

  Scenario: ConditionEqual assertion fails
    Given a COS with group "test" and revision 1
    And a phase "install" with a CRD "testfails" with assertion conditionEqual type "NonExistent" status "True"
    When the COS is created
    Then the COS should have condition "Available" with status "False"

  Scenario: FieldsEqual assertion fails
    Given a COS with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-fields-fail" with data:
      | key | value |
      | a   | one   |
      | b   | two   |
    And the last object has assertion fieldsEqual fieldA ".data.a" fieldB ".data.b"
    When the COS is created
    Then the COS should have condition "Available" with status "False"

  Scenario: FieldValue assertion fails
    Given a COS with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-fv-fail" with data key "ready" value "false"
    And the last object has assertion fieldValue path ".data.ready" value "true"
    When the COS is created
    Then the COS should have condition "Available" with status "False"

  Scenario: CEL expression assertion fails with custom message
    Given a COS with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-cel-fail" with assertion celExpression "self.metadata.name == 'wrong'" message "name must be wrong"
    When the COS is created
    Then the COS should have condition "Available" with status "False"

  Scenario: Invalid CEL expression keeps COS unavailable
    Given a COS with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-bad-cel" with assertion celExpression "this is not valid CEL %%%" message "bad"
    When the COS is created
    Then the COS should have condition "Available" with status "False"

Feature: COS owns stamped COSRs via owner references

  Scenario: Stamped COSR has an owner reference to the COS
    Given a COS named "own-ref"
    And a phase "install" with a ConfigMap "cm-own-ref"
    When the COS is created
    Then a COSR should exist with group "own-ref" and revision 1
    And the COSR with group "own-ref" and revision 1 should have a controller owner reference to COS "own-ref"

  Scenario: Deleting COS cascades deletion to owned COSRs
    Given a COS named "own-cascade"
    And a phase "install" with a ConfigMap "cm-cascade"
    When the COS is created
    Then a COSR should exist with group "own-cascade" and revision 1
    When the COS "own-cascade" is deleted
    Then the COSR with group "own-cascade" and revision 1 should not exist

  Scenario: Orphan-deleting COS preserves owned COSRs
    Given a COS named "own-orphan"
    And a phase "install" with a ConfigMap "cm-orphan"
    When the COS is created
    Then a COSR should exist with group "own-orphan" and revision 1
    When the COS "own-orphan" is deleted with cascade orphan
    Then a COSR should exist with group "own-orphan" and revision 1
    And the COSR with group "own-orphan" and revision 1 should not have an owner reference

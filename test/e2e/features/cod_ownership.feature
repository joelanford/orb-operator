Feature: COD owns stamped COSRs via owner references

  Scenario: Stamped COSR has an owner reference to the COD
    Given a COD named "own-ref"
    And a phase "install" with a ConfigMap "cm-own-ref"
    When the COD is created
    Then the COSR with group "own-ref" and revision 1 should have a controller owner reference to COD "own-ref"

  Scenario: Deleting COD cascades deletion to owned COSRs
    Given a COD named "own-cascade"
    And a phase "install" with a ConfigMap "cm-cascade"
    When the COD is created
    Then a COSR should exist with group "own-cascade" and revision 1
    When the COD "own-cascade" is deleted
    Then the COSR with group "own-cascade" and revision 1 should not exist

  Scenario: Orphan-deleting COD preserves owned COSRs
    Given a COD named "own-orphan"
    And a phase "install" with a ConfigMap "cm-orphan"
    When the COD is created
    Then a COSR should exist with group "own-orphan" and revision 1
    When the COD "own-orphan" is deleted with cascade orphan
    Then a COSR should exist with group "own-orphan" and revision 1
    And the COSR with group "own-orphan" and revision 1 should not have an owner reference

  Scenario: COD adopts a pre-existing unowned COSR in its group
    Given a COSR with group "own-adopt" and revision 1
    And a phase "install" with a ConfigMap "cm-adopt"
    And the COSR is created and becomes Available
    Given a COD named "own-adopt"
    And a phase "install" with a ConfigMap "cm-adopt"
    When the COD is created
    Then the COSR with group "own-adopt" and revision 1 should have a controller owner reference to COD "own-adopt"
    And a COSR should exist with group "own-adopt" and revision 2

  Scenario: COD adopts an unowned COSR created after the COD
    Given an available COD named "own-late"
    And a COSR with group "own-late" and revision 10
    And a phase "install" with a ConfigMap "cm-late-external"
    And the COSR is created
    Then the COSR with group "own-late" and revision 10 should have a controller owner reference to COD "own-late"

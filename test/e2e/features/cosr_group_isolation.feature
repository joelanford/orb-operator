Feature: COSRs in separate groups are independent

  Scenario: COSRs in different groups do not interfere
    # Create alpha group
    Given a COSR with group "alpha" and revision 1
    And a phase "install" with a ConfigMap "cm-alpha"
    And the COSR is created and becomes Available
    # Create beta group
    Given a COSR with group "beta" and revision 1
    And a phase "install" with a ConfigMap "cm-beta"
    And the COSR is created and becomes Available
    # Both groups are independently Available
    Then the COSR in group "alpha" revision 1 should have condition "Available" with status "True"
    And the COSR in group "beta" revision 1 should have condition "Available" with status "True"
    # Delete alpha's object — beta is unaffected, alpha recovers
    When the ConfigMap "cm-alpha" is deleted
    Then the ConfigMap "cm-alpha" should be recreated
    And the COSR in group "beta" revision 1 should have condition "Available" with status "True"
    And the ConfigMap "cm-beta" should exist

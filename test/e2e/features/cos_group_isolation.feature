Feature: COSs in separate groups are independent

  Scenario: COSs in different groups do not interfere
    # Create alpha group
    Given a COS with group "alpha" and revision 1
    And a phase "install" with a ConfigMap "cm-alpha"
    And the COS is created and becomes Available
    # Create beta group
    Given a COS with group "beta" and revision 1
    And a phase "install" with a ConfigMap "cm-beta"
    And the COS is created and becomes Available
    # Both groups are independently Available
    Then the COS in group "alpha" revision 1 should have condition "Available" with status "True"
    And the COS in group "beta" revision 1 should have condition "Available" with status "True"
    # Delete alpha's object — beta is unaffected, alpha recovers
    When the ConfigMap "cm-alpha" is deleted
    Then the ConfigMap "cm-alpha" should be recreated
    And the COS in group "beta" revision 1 should have condition "Available" with status "True"
    And the ConfigMap "cm-beta" should exist

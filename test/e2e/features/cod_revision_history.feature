Feature: COD enforces revisionHistoryLimit on archived COSRs

  Scenario: Archived COSRs beyond revisionHistoryLimit are pruned
    Given a COD named "hist-serial" with revisionHistoryLimit 2
    And a phase "install" with a ConfigMap "cm-hs-1"
    When the COD is created
    Then the COD "hist-serial" should be Available
    When the COD template spec is updated with a ConfigMap "cm-hs-2" in phase "install"
    Then the COD "hist-serial" should have active revision 2
    And the COD "hist-serial" should be Available
    When the COD template spec is updated with a ConfigMap "cm-hs-3" in phase "install"
    Then the COD "hist-serial" should have active revision 3
    And the COD "hist-serial" should be Available
    When the COD template spec is updated with a ConfigMap "cm-hs-4" in phase "install"
    Then the COD "hist-serial" should have active revision 4
    And the COD "hist-serial" should be Available
    # With limit 2, only the 2 highest-revision archived COSRs are retained
    And the COSR with group "hist-serial" and revision 1 should not exist

  Scenario: Active COSR is never counted toward the limit
    Given a COD named "hist-active" with revisionHistoryLimit 0
    And a phase "install" with a ConfigMap "cm-hist-active"
    When the COD is created
    Then a COSR should exist with group "hist-active" and revision 1
    # Even with limit 0, the active COSR is preserved
    And the COSR count for COD "hist-active" should be 1

  Scenario: Default revisionHistoryLimit of 5 applies
    Given a COD named "hist-default"
    And a phase "install" with a ConfigMap "cm-hd-1"
    When the COD is created
    Then the COD "hist-default" should be Available
    When the COD template spec is updated with a ConfigMap "cm-hd-2" in phase "install"
    Then the COD "hist-default" should have active revision 2
    And the COD "hist-default" should be Available
    When the COD template spec is updated with a ConfigMap "cm-hd-3" in phase "install"
    Then the COD "hist-default" should have active revision 3
    And the COD "hist-default" should be Available
    When the COD template spec is updated with a ConfigMap "cm-hd-4" in phase "install"
    Then the COD "hist-default" should have active revision 4
    And the COD "hist-default" should be Available
    When the COD template spec is updated with a ConfigMap "cm-hd-5" in phase "install"
    Then the COD "hist-default" should have active revision 5
    And the COD "hist-default" should be Available
    When the COD template spec is updated with a ConfigMap "cm-hd-6" in phase "install"
    Then the COD "hist-default" should have active revision 6
    And the COD "hist-default" should be Available
    When the COD template spec is updated with a ConfigMap "cm-hd-7" in phase "install"
    Then the COD "hist-default" should have active revision 7
    And the COD "hist-default" should be Available
    # Revision 1 should be pruned (6 archived > limit of 5)
    And the COSR with group "hist-default" and revision 1 should not exist

  Scenario: Lowering revisionHistoryLimit retroactively prunes archived COSRs
    Given a COD named "hist-lower"
    And a phase "install" with a ConfigMap "cm-hl-1"
    When the COD is created
    Then the COD "hist-lower" should be Available
    When the COD template spec is updated with a ConfigMap "cm-hl-2" in phase "install"
    Then the COD "hist-lower" should have active revision 2
    And the COD "hist-lower" should be Available
    When the COD template spec is updated with a ConfigMap "cm-hl-3" in phase "install"
    Then the COD "hist-lower" should have active revision 3
    And the COD "hist-lower" should be Available
    When the COD template spec is updated with a ConfigMap "cm-hl-4" in phase "install"
    Then the COD "hist-lower" should have active revision 4
    And the COD "hist-lower" should be Available
    When the COD template spec is updated with a ConfigMap "cm-hl-5" in phase "install"
    Then the COD "hist-lower" should have active revision 5
    And the COD "hist-lower" should be Available
    When the COD template spec is updated with a ConfigMap "cm-hl-6" in phase "install"
    Then the COD "hist-lower" should have active revision 6
    And the COD "hist-lower" should be Available
    When the COD template spec is updated with a ConfigMap "cm-hl-7" in phase "install"
    Then the COD "hist-lower" should have active revision 7
    And the COD "hist-lower" should be Available
    # Default limit 5: revisions 1 is pruned, 2-6 archived, 7 active
    And the COSR with group "hist-lower" and revision 1 should not exist
    # Lower limit to 2: revisions 2-4 should be pruned retroactively
    When the COD "hist-lower" revisionHistoryLimit is set to 2
    Then the COSR with group "hist-lower" and revision 2 should not exist
    And the COSR with group "hist-lower" and revision 3 should not exist
    And the COSR with group "hist-lower" and revision 4 should not exist
    # Revisions 5-6 (archived) and 7 (active) remain
    And a COSR should exist with group "hist-lower" and revision 5
    And a COSR should exist with group "hist-lower" and revision 6
    And a COSR should exist with group "hist-lower" and revision 7

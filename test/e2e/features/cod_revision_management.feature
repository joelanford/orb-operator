Feature: COD creates new revisions on template changes

  Scenario: Updating template spec creates a new revision
    Given an available COD named "rev-spec"
    When the COD template spec is updated with a ConfigMap "cm-rev2" in phase "install"
    Then a COSR should exist with group "rev-spec" and revision 2

  Scenario: Updating template metadata creates a new revision
    Given an available COD named "rev-meta"
    When the COD template label "version" is updated to "v2"
    Then a COSR should exist with group "rev-meta" and revision 2

  Scenario: Multiple template changes produce monotonically increasing revisions
    Given an available COD named "rev-multi"
    When the COD template spec is updated with a ConfigMap "cm-multi-2" in phase "install"
    Then the COD "rev-multi" should have active revision 2
    And the COD "rev-multi" should be Available
    When the COD template spec is updated with a ConfigMap "cm-multi-3" in phase "install"
    Then a COSR should exist with group "rev-multi" and revision 3

  Scenario: Template change during in-progress transition
    Given an available COD named "rev-midtx"
    # Rev 2 has a probe that won't pass — holds it unavailable
    When the COD template spec is updated with a gated ConfigMap "cm-midtx-2" in phase "install"
    Then a COSR should exist with group "rev-midtx" and revision 2
    And the COD "rev-midtx" should have condition "Available" with status "Unknown" and reason "Progressing"
    # Update again while rev 2 is still not available — rev 3 has no gate
    When the COD template spec is updated with a ConfigMap "cm-midtx-3" in phase "install"
    Then a COSR should exist with group "rev-midtx" and revision 3
    # Rev 3 eventually becomes available; rev 1 and 2 are archived
    And the COD "rev-midtx" should be Available
    And the COSR with group "rev-midtx" and revision 1 should have lifecycleState "Archived"
    And the COSR with group "rev-midtx" and revision 2 should have lifecycleState "Archived"

  Scenario: No new COSR when template has not changed
    Given a COD named "rev-idempotent"
    And a phase "install" with a ConfigMap "cm-idempotent"
    When the COD is created
    Then a COSR should exist with group "rev-idempotent" and revision 1
    # Trigger a reconcile by changing a COD label (not the template)
    When the COD "rev-idempotent" label "trigger" is set to "reconcile"
    Then the COSR count for COD "rev-idempotent" should be 1

  Scenario: Revision transition archives old revision and cleans up old objects
    Given a COD named "rev-cleanup"
    And a phase "install" with a ConfigMap "cm-cleanup-old"
    When the COD is created
    Then the COD "rev-cleanup" should be Available
    And the ConfigMap "cm-cleanup-old" should exist
    When the COD template spec is updated with a ConfigMap "cm-cleanup-new" in phase "install"
    Then the COD "rev-cleanup" should be Available
    And the COSR with group "rev-cleanup" and revision 1 should have lifecycleState "Archived"
    And the ConfigMap "cm-cleanup-old" should not exist
    And the ConfigMap "cm-cleanup-new" should exist

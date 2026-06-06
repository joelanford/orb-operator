Feature: COS creates new revisions on template changes

  Scenario: Updating template spec creates a new revision
    Given an available COS named "rev-spec"
    When the COS template spec is updated with a ConfigMap "cm-rev2" in phase "install"
    Then a COSR should exist with group "rev-spec" and revision 2

  Scenario: Updating template metadata creates a new revision
    Given an available COS named "rev-meta"
    When the COS template label "version" is updated to "v2"
    Then a COSR should exist with group "rev-meta" and revision 2

  Scenario: Multiple template changes produce monotonically increasing revisions
    Given an available COS named "rev-multi"
    When the COS template spec is updated with a ConfigMap "cm-multi-2" in phase "install"
    Then the COS "rev-multi" should have active revision 2
    And the COS "rev-multi" should be Available
    When the COS template spec is updated with a ConfigMap "cm-multi-3" in phase "install"
    Then a COSR should exist with group "rev-multi" and revision 3

  Scenario: Template change during in-progress transition
    Given an available COS named "rev-midtx"
    # Rev 2 has a probe that won't pass — holds it unavailable
    When the COS template spec is updated with a gated ConfigMap "cm-midtx-2" in phase "install"
    Then a COSR should exist with group "rev-midtx" and revision 2
    And the COS "rev-midtx" should have condition "Available" with status "Unknown" and reason "Progressing"
    # Update again while rev 2 is still not available — rev 3 has no gate
    When the COS template spec is updated with a ConfigMap "cm-midtx-3" in phase "install"
    Then a COSR should exist with group "rev-midtx" and revision 3
    # Rev 3 eventually becomes available; rev 1 and 2 are archived
    And the COS "rev-midtx" should be Available
    And the COSR with group "rev-midtx" and revision 1 should have lifecycleState "Archived"
    And the COSR with group "rev-midtx" and revision 2 should have lifecycleState "Archived"

  Scenario: No new COSR when template has not changed
    Given a COS named "rev-idempotent"
    And a phase "install" with a ConfigMap "cm-idempotent"
    When the COS is created
    Then a COSR should exist with group "rev-idempotent" and revision 1
    # Trigger a reconcile by changing a COS label (not the template)
    When the COS "rev-idempotent" label "trigger" is set to "reconcile"
    Then the COSR count for COS "rev-idempotent" should be 1

  Scenario: Revision transition archives old revision and cleans up old objects
    Given a COS named "rev-cleanup"
    And a phase "install" with a ConfigMap "cm-cleanup-old"
    When the COS is created
    Then the COS "rev-cleanup" should be Available
    And the ConfigMap "cm-cleanup-old" should exist
    When the COS template spec is updated with a ConfigMap "cm-cleanup-new" in phase "install"
    Then the COS "rev-cleanup" should be Available
    And the COSR with group "rev-cleanup" and revision 1 should have lifecycleState "Archived"
    And the ConfigMap "cm-cleanup-old" should not exist
    And the ConfigMap "cm-cleanup-new" should exist

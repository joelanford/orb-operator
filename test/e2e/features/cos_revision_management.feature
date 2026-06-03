Feature: COS creates new revisions on template changes

  Scenario: Updating template spec creates a new revision
    Given a COS named "rev-spec"
    And a phase "install" with a ConfigMap "cm-rev1"
    When the COS is created
    Then a COSR should exist with group "rev-spec" and revision 1
    When the COS template spec is updated with a ConfigMap "cm-rev2" in phase "install"
    Then a COSR should exist with group "rev-spec" and revision 2

  Scenario: Updating template metadata creates a new revision
    Given a COS named "rev-meta"
    And a phase "install" with a ConfigMap "cm-rev-meta"
    When the COS is created
    Then a COSR should exist with group "rev-meta" and revision 1
    When the COS template label "version" is updated to "v2"
    Then a COSR should exist with group "rev-meta" and revision 2

  Scenario: Multiple template changes produce monotonically increasing revisions
    Given a COS named "rev-multi"
    And a phase "install" with a ConfigMap "cm-multi-1"
    When the COS is created
    Then a COSR should exist with group "rev-multi" and revision 1
    When the COS template spec is updated with a ConfigMap "cm-multi-2" in phase "install"
    Then a COSR should exist with group "rev-multi" and revision 2
    When the COS template spec is updated with a ConfigMap "cm-multi-3" in phase "install"
    Then a COSR should exist with group "rev-multi" and revision 3

  Scenario: No new COSR when template has not changed
    Given a COS named "rev-idempotent"
    And a phase "install" with a ConfigMap "cm-idempotent"
    When the COS is created
    Then a COSR should exist with group "rev-idempotent" and revision 1
    # Trigger a reconcile by changing a COS label (not the template)
    When the COS "rev-idempotent" label "trigger" is set to "reconcile"
    Then the COSR count for COS "rev-idempotent" should be 1

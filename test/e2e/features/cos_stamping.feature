Feature: COS stamps out a COSR from its template

  Scenario: COS creates a COSR with correct group and revision
    Given a COS named "stamp-basic"
    And a phase "install" with a ConfigMap "cm-stamp"
    When the COS is created
    Then a COSR should exist with group "stamp-basic" and revision 1
    And the COSR with group "stamp-basic" and revision 1 should have lifecycleState "Active"

  Scenario: Stamped COSR spec matches complex template without root collisionProtection
    Given a COS named "stamp-no-cp"
    And a phase "crds" with a CRD "testwidgets"
    And the phase "crds" collisionProtection is "None"
    And a phase "deploy" with a ConfigMap "cm-deploy-1"
    And the phase "deploy" also has a ConfigMap "cm-deploy-2"
    And the last object has assertion fieldValue path ".data.ready" value "true"
    And the last object collisionProtection is "IfNoController"
    And a phase "post" with a ConfigMap "cm-post"
    When the COS is created
    Then a COSR should exist with group "stamp-no-cp" and revision 1
    And the stamped COSR spec for "stamp-no-cp" revision 1 should match the COS template spec

  Scenario: Stamped COSR spec matches complex template with root collisionProtection
    Given a COS named "stamp-with-cp"
    And the COSR collisionProtection is "Prevent"
    And a phase "crds" with a CRD "testgadgets"
    And a phase "deploy" with a ConfigMap "cm-cp-deploy"
    And the last object has assertion celExpression "has(self.data)"
    And the phase "deploy" collisionProtection is "IfNoController"
    And the phase "deploy" also has a ConfigMap "cm-cp-extra"
    When the COS is created
    Then a COSR should exist with group "stamp-with-cp" and revision 1
    And the stamped COSR spec for "stamp-with-cp" revision 1 should match the COS template spec

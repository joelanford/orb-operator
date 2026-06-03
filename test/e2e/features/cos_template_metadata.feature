Feature: COS propagates template metadata to stamped COSRs

  Scenario: Template labels and annotations are propagated to the stamped COSR
    Given a COS named "tmpl-meta"
    And the COS template has label "app" with value "ext-a"
    And the COS template has label "version" with value "v2"
    And the COS template has annotation "meta.example.com/version" with value "1.0.0"
    And the COS template has annotation "note" with value "test"
    And a phase "install" with a ConfigMap "cm-meta"
    When the COS is created
    Then a COSR should exist with group "tmpl-meta" and revision 1
    And the COSR with group "tmpl-meta" and revision 1 should have label "app" with value "ext-a"
    And the COSR with group "tmpl-meta" and revision 1 should have label "version" with value "v2"
    And the COSR with group "tmpl-meta" and revision 1 should have annotation "meta.example.com/version" with value "1.0.0"
    And the COSR with group "tmpl-meta" and revision 1 should have annotation "note" with value "test"

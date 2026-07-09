Feature: COD propagates template metadata to stamped COSs

  Scenario: Template labels and annotations are propagated to the stamped COS
    Given a COD named "tmpl-meta"
    And the COD template has label "app" with value "ext-a"
    And the COD template has label "version" with value "v2"
    And the COD template has annotation "meta.example.com/version" with value "1.0.0"
    And the COD template has annotation "note" with value "test"
    And a phase "install" with a ConfigMap "cm-meta"
    When the COD is created
    Then the COS with group "tmpl-meta" and revision 1 should have label "app" with value "ext-a"
    And the COS with group "tmpl-meta" and revision 1 should have label "version" with value "v2"
    And the COS with group "tmpl-meta" and revision 1 should have annotation "meta.example.com/version" with value "1.0.0"
    And the COS with group "tmpl-meta" and revision 1 should have annotation "note" with value "test"

  Scenario: Controller-managed template hash label overrides user-provided value
    Given a COD named "tmpl-hash"
    And the COD template has label "orb.operatorframework.io/template-hash" with value "user-provided-value"
    And a phase "install" with a ConfigMap "cm-hash"
    When the COD is created
    Then a COS should exist with group "tmpl-hash" and revision 1
    And the COS with group "tmpl-hash" and revision 1 should not have label "orb.operatorframework.io/template-hash" with value "user-provided-value"

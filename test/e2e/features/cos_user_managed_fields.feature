Feature: User-managed fields on managed objects survive reconciliation

  Scenario: User-added fields on a managed object survive reconciliation
    Given a COS with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-preserve" with data key "app" value "test"
    And the COS is created and becomes Available
    When a resource is patched with:
      """
      apiVersion: v1
      kind: ConfigMap
      metadata:
        name: cm-preserve
        namespace: ${NAMESPACE}
        annotations:
          foo: bar
        labels:
          fizz: buzz
      data:
        app: drifted
      """
    Then a resource should match:
      """
      apiVersion: v1
      kind: ConfigMap
      metadata:
        name: cm-preserve
        namespace: ${NAMESPACE}
        annotations:
          foo: bar
        labels:
          fizz: buzz
      data:
        app: test
      """

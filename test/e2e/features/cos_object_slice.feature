Feature: COS resolves objects from ClusterObjectSlice via objectRef

  Scenario: COS with objectRef resolves from a ClusterObjectSlice and becomes Available
    Given a ClusterObjectSlice "slice1" with a ConfigMap "cm-from-slice"
    And a COS with group "cosl-ref" and revision 1
    And a phase "install" with an objectRef to slice "slice1" for ConfigMap "cm-from-slice"
    When the COS is created
    Then the COS should have condition "Available" with status "True"
    And the ConfigMap "cm-from-slice" should exist

  Scenario: COS with mixed inline and objectRef in the same phase
    Given a ClusterObjectSlice "slice-mixed" with a ConfigMap "cm-slice"
    And a COS with group "cosl-mixed" and revision 1
    And a phase "install" with a ConfigMap "cm-inline"
    And the phase "install" also has an objectRef to slice "slice-mixed" for ConfigMap "cm-slice"
    When the COS is created
    Then the COS should have condition "Available" with status "True"
    And the ConfigMap "cm-inline" should exist
    And the ConfigMap "cm-slice" should exist

  Scenario: COS with objectRef to nonexistent slice reports InvalidRevision
    Given a COS with group "cosl-nosuch" and revision 1
    And a phase "install" with an objectRef to slice "no-such-slice" for ConfigMap "phantom"
    When the COS is created
    Then the COS should have condition "Available" with status "False" and reason "InvalidRevision"

  Scenario: COS with objectRef to object not present in slice reports InvalidRevision
    Given a ClusterObjectSlice "slice-wrong" with a ConfigMap "actual-cm"
    And a COS with group "cosl-noobj" and revision 1
    And a phase "install" with an objectRef to slice "slice-wrong" for ConfigMap "wrong-name"
    When the COS is created
    Then the COS should have condition "Available" with status "False" and reason "InvalidRevision"

  Scenario: COS with objectRef to gzip-compressed slice content becomes Available
    Given a ClusterObjectSlice "slice-gz" with a gzip-compressed ConfigMap "cm-compressed"
    And a COS with group "cosl-gz" and revision 1
    And a phase "install" with an objectRef to slice "slice-gz" for ConfigMap "cm-compressed"
    When the COS is created
    Then the COS should have condition "Available" with status "True"
    And the ConfigMap "cm-compressed" should exist

  Scenario: COS detects content hash mismatch after slice delete and recreate
    Given a ClusterObjectSlice "slice-hash" with a ConfigMap "cm-hash"
    And a COS with group "cosl-hash" and revision 1
    And a phase "install" with an objectRef to slice "slice-hash" for ConfigMap "cm-hash"
    When the COS is created and becomes Available
    And the ClusterObjectSlice "slice-hash" is deleted
    And the ClusterObjectSlice "slice-hash" is recreated with a ConfigMap "cm-hash" with data key "mutated" value "true"
    Then the COS should have condition "Available" with status "Unknown" and reason "InvalidRevision"

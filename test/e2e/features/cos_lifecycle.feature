Feature: COS Active and Archived lifecycle behavior

  Scenario: Active COS recreates a deleted managed object
    Given a COS with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-recreate"
    And the COS is created and becomes Available
    When the ConfigMap "cm-recreate" is deleted
    Then the ConfigMap "cm-recreate" should be recreated

  Scenario: COS stays Available when deleted object is recreated instantly
    Given a COS with group "test" and revision 1
    And a phase "p1" with a ConfigMap "cm-p1"
    And a phase "p2" with a ConfigMap "cm-p2"
    And a phase "p3" with a ConfigMap "cm-p3"
    And the COS is created and becomes Available
    When the ConfigMap "cm-p1" is deleted
    Then the ConfigMap "cm-p1" should be recreated
    And the COS should have condition "Available" with status "True"
    When the ConfigMap "cm-p2" is deleted
    Then the ConfigMap "cm-p2" should be recreated
    And the COS should have condition "Available" with status "True"
    When the ConfigMap "cm-p3" is deleted
    Then the ConfigMap "cm-p3" should be recreated
    And the COS should have condition "Available" with status "True"

  Scenario: Available condition flaps as per-phase assertions fail and recover
    Given a COS with group "test" and revision 1
    And a phase "p1" with a ConfigMap "cm-p1"
    And a phase "p2" with a ConfigMap "cm-p2"
    And a phase "p3" with a ConfigMap "cm-p3"
    When the COS is created
    Then the COS should have condition "Available" with status "True"
    # Phase 1 gate closes
    When the gate on ConfigMap "cm-p1" is closed
    Then the COS should have condition "Available" with status "False"
    # Phase 1 gate reopens
    When the gate on ConfigMap "cm-p1" is opened
    Then the COS should have condition "Available" with status "True"
    # Phase 2 gate closes
    When the gate on ConfigMap "cm-p2" is closed
    Then the COS should have condition "Available" with status "False"
    # Phase 2 gate reopens
    When the gate on ConfigMap "cm-p2" is opened
    Then the COS should have condition "Available" with status "True"
    # Phase 3 gate closes
    When the gate on ConfigMap "cm-p3" is closed
    Then the COS should have condition "Available" with status "False"
    # Phase 3 gate reopens
    When the gate on ConfigMap "cm-p3" is opened
    Then the COS should have condition "Available" with status "True"

  Scenario: Archived COS cannot be unarchived
    Given an available COS with group "test" and revision 1
    When the COS lifecycleState is set to "Archived"
    Then setting the COS lifecycleState to "Active" should fail

  Scenario: Deleting a COS with cascade orphan preserves managed objects
    Given a COS with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-orphan"
    And the COS is created and becomes Available
    Then the ConfigMap "cm-orphan" should have an owner reference
    When the COS is deleted with cascade orphan
    Then the COS should not exist
    And the ConfigMap "cm-orphan" should exist
    And the ConfigMap "cm-orphan" should not have an owner reference

  Scenario: Foreground deletion tears down managed objects
    Given a COS with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-fg-delete"
    And the COS is created and becomes Available
    When the COS is deleted with cascade foreground
    Then the COS should not exist
    And the ConfigMap "cm-fg-delete" should not exist

  Scenario: Background deletion tears down managed objects
    Given a COS with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-bg-delete"
    And the COS is created and becomes Available
    When the COS is deleted with cascade background
    Then the COS should not exist
    And the ConfigMap "cm-bg-delete" should not exist

  Scenario: Archiving a single-revision COS deletes managed objects
    Given a COS with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-archive"
    And the COS is created and becomes Available
    When the COS lifecycleState is set to "Archived"
    Then the ConfigMap "cm-archive" should not exist
    And the COS should have condition "Available" with status "False" and reason "Archived"

  Scenario: Archived COS finalizer is removed after teardown completes
    Given a COS with group "fin-remove" and revision 1
    And a phase "install" with a ConfigMap "cm-fin-remove"
    And the COS is created and becomes Available
    When the COS with group "fin-remove" and revision 1 lifecycleState is set to "Archived"
    Then the ConfigMap "cm-fin-remove" should not exist
    And the COS with group "fin-remove" and revision 1 should not have finalizer "orb.operatorframework.io/cos-finalizer"

  Scenario: Teardown error on one object does not prevent tearing down other objects in the same phase
    Given a COS with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-td-before"
    And the phase "install" also has a ConfigMap "cm-td-blocked"
    And the phase "install" also has a ConfigMap "cm-td-after"
    And the COS is created and becomes Available
    And ConfigMap "cm-td-blocked" operations are blocked
    When the COS lifecycleState is set to "Archived"
    Then the COS should have condition "Available" with status "Unknown" and reason "TeardownError"
    And the ConfigMap "cm-td-before" should not exist
    And the ConfigMap "cm-td-blocked" should exist
    And the ConfigMap "cm-td-after" should not exist

  Scenario: Deleting a lower-revision COS in a chain tears down its managed objects
    Given a COS with group "chain-del" and revision 1
    And a phase "install" with a ConfigMap "cm-chain-del-1"
    And the COS is created and becomes Available
    When a COS with group "chain-del" and revision 2
    And the phase "install" has a ConfigMap "cm-chain-del-2"
    And the COS is created and becomes Available
    Then the ConfigMap "cm-chain-del-1" should exist
    When the COS with group "chain-del" and revision 1 is deleted
    Then the COS with group "chain-del" and revision 1 should not exist
    And the ConfigMap "cm-chain-del-1" should not exist
    And the ConfigMap "cm-chain-del-2" should exist

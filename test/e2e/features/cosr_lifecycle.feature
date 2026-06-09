Feature: COSR Active and Archived lifecycle behavior

  Scenario: Active COSR recreates a deleted managed object
    Given a COSR with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-recreate"
    And the COSR is created and becomes Available
    When the ConfigMap "cm-recreate" is deleted
    Then the ConfigMap "cm-recreate" should be recreated

  Scenario: COSR stays Available when deleted object is recreated instantly
    Given a COSR with group "test" and revision 1
    And a phase "p1" with a ConfigMap "cm-p1"
    And a phase "p2" with a ConfigMap "cm-p2"
    And a phase "p3" with a ConfigMap "cm-p3"
    And the COSR is created and becomes Available
    When the ConfigMap "cm-p1" is deleted
    Then the ConfigMap "cm-p1" should be recreated
    And the COSR should have condition "Available" with status "True"
    When the ConfigMap "cm-p2" is deleted
    Then the ConfigMap "cm-p2" should be recreated
    And the COSR should have condition "Available" with status "True"
    When the ConfigMap "cm-p3" is deleted
    Then the ConfigMap "cm-p3" should be recreated
    And the COSR should have condition "Available" with status "True"

  Scenario: Available condition flaps as per-phase assertions fail and recover
    Given a COSR with group "test" and revision 1
    And a phase "p1" with a ConfigMap "cm-p1"
    And a phase "p2" with a ConfigMap "cm-p2"
    And a phase "p3" with a ConfigMap "cm-p3"
    When the COSR is created
    Then the COSR should have condition "Available" with status "True"
    # Phase 1 gate closes
    When the gate on ConfigMap "cm-p1" is closed
    Then the COSR should have condition "Available" with status "False"
    # Phase 1 gate reopens
    When the gate on ConfigMap "cm-p1" is opened
    Then the COSR should have condition "Available" with status "True"
    # Phase 2 gate closes
    When the gate on ConfigMap "cm-p2" is closed
    Then the COSR should have condition "Available" with status "False"
    # Phase 2 gate reopens
    When the gate on ConfigMap "cm-p2" is opened
    Then the COSR should have condition "Available" with status "True"
    # Phase 3 gate closes
    When the gate on ConfigMap "cm-p3" is closed
    Then the COSR should have condition "Available" with status "False"
    # Phase 3 gate reopens
    When the gate on ConfigMap "cm-p3" is opened
    Then the COSR should have condition "Available" with status "True"

  Scenario: Archived COSR cannot be unarchived
    Given an available COSR with group "test" and revision 1
    When the COSR lifecycleState is set to "Archived"
    Then setting the COSR lifecycleState to "Active" should fail

  Scenario: Deleting a COSR with cascade orphan preserves managed objects
    Given a COSR with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-orphan"
    And the COSR is created and becomes Available
    Then the ConfigMap "cm-orphan" should have an owner reference
    When the COSR is deleted with cascade orphan
    Then the COSR should not exist
    And the ConfigMap "cm-orphan" should exist
    And the ConfigMap "cm-orphan" should not have an owner reference

  Scenario: Foreground deletion tears down managed objects
    Given a COSR with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-fg-delete"
    And the COSR is created and becomes Available
    When the COSR is deleted with cascade foreground
    Then the COSR should not exist
    And the ConfigMap "cm-fg-delete" should not exist

  Scenario: Background deletion tears down managed objects
    Given a COSR with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-bg-delete"
    And the COSR is created and becomes Available
    When the COSR is deleted with cascade background
    Then the COSR should not exist
    And the ConfigMap "cm-bg-delete" should not exist

  Scenario: Archiving a single-revision COSR deletes managed objects
    Given a COSR with group "test" and revision 1
    And a phase "install" with a ConfigMap "cm-archive"
    And the COSR is created and becomes Available
    When the COSR lifecycleState is set to "Archived"
    Then the ConfigMap "cm-archive" should not exist
    And the COSR should have condition "Available" with status "False" and reason "Archived"

  Scenario: Archived COSR finalizer is removed after teardown completes
    Given a COSR with group "fin-remove" and revision 1
    And a phase "install" with a ConfigMap "cm-fin-remove"
    And the COSR is created and becomes Available
    When the COSR with group "fin-remove" and revision 1 lifecycleState is set to "Archived"
    Then the ConfigMap "cm-fin-remove" should not exist
    And the COSR with group "fin-remove" and revision 1 should not have finalizer "orb.operatorframework.io/cosr-finalizer"

  Scenario: Deleting a lower-revision COSR in a chain tears down its managed objects
    Given a COSR with group "chain-del" and revision 1
    And a phase "install" with a ConfigMap "cm-chain-del-1"
    And the COSR is created and becomes Available
    When a COSR with group "chain-del" and revision 2
    And the phase "install" has a ConfigMap "cm-chain-del-2"
    And the COSR is created and becomes Available
    Then the ConfigMap "cm-chain-del-1" should exist
    When the COSR with group "chain-del" and revision 1 is deleted
    Then the COSR with group "chain-del" and revision 1 should not exist
    And the ConfigMap "cm-chain-del-1" should not exist
    And the ConfigMap "cm-chain-del-2" should exist

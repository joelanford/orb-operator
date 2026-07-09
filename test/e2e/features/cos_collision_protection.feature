Feature: Collision protection at root, phase, and object levels

  # --- Behavior: Prevent ---

  Scenario: Default collision protection prevents a second group from managing the same object
    Given a COS with group "alpha" and revision 1
    And a phase "install" with a ConfigMap "cm-contested"
    And the COS is created and becomes Available
    Given a COS with group "beta" and revision 1
    And a phase "install" with a ConfigMap "cm-contested"
    When the COS is created
    Then the COS in group "beta" revision 1 should have condition "Available" with status "False"
    And the COS in group "alpha" revision 1 should have condition "Available" with status "True"

  Scenario: Explicit Prevent blocks a second group from managing the same object
    Given a COS with group "alpha" and revision 1
    And a phase "install" with a ConfigMap "cm-explicit-prevent"
    And the COS is created and becomes Available
    Given a COS with group "beta" and revision 1
    And the COS collisionProtection is "Prevent"
    And a phase "install" with a ConfigMap "cm-explicit-prevent"
    When the COS is created
    Then the COS in group "beta" revision 1 should have condition "Available" with status "False"

  Scenario: Prevent blocks adoption of a standalone object
    Given a standalone ConfigMap "cm-standalone-prevent" exists
    And a COS with group "adopt" and revision 1
    And the COS collisionProtection is "Prevent"
    And a phase "install" with a ConfigMap "cm-standalone-prevent"
    When the COS is created
    Then the COS should have condition "Available" with status "False"

  # --- Behavior: None ---

  Scenario: CollisionProtection None allows a second group to manage the same object
    Given a COS with group "alpha" and revision 1
    And the COS collisionProtection is "None"
    And a phase "install" with a ConfigMap "cm-none"
    And the COS is created and becomes Available
    Given a COS with group "beta" and revision 1
    And the COS collisionProtection is "None"
    And a phase "install" with a ConfigMap "cm-none"
    When the COS is created
    Then the COS in group "beta" revision 1 should have condition "Available" with status "True"

  # --- Behavior: IfNoController ---

  Scenario: IfNoController adopts a standalone object
    Given a standalone ConfigMap "cm-standalone-adopt" exists
    And a COS with group "adopt" and revision 1
    And the COS collisionProtection is "IfNoController"
    And a phase "install" with a ConfigMap "cm-standalone-adopt"
    When the COS is created
    Then the COS should have condition "Available" with status "True"
    And the ConfigMap "cm-standalone-adopt" should have an owner reference

  Scenario: IfNoController cannot adopt an object owned by another COS
    Given a COS with group "alpha" and revision 1
    And a phase "install" with a ConfigMap "cm-owned"
    And the COS is created and becomes Available
    Given a COS with group "beta" and revision 1
    And the COS collisionProtection is "IfNoController"
    And a phase "install" with a ConfigMap "cm-owned"
    When the COS is created
    Then the COS in group "beta" revision 1 should have condition "Available" with status "False"

  # --- Precedence ---

  Scenario: Phase-level CP overrides root-level CP
    Given a COS with group "alpha" and revision 1
    And a phase "install" with a ConfigMap "cm-phase-override"
    And the COS is created and becomes Available
    Given a COS with group "beta" and revision 1
    And the COS collisionProtection is "Prevent"
    And a phase "install" with a ConfigMap "cm-phase-override"
    And the phase "install" collisionProtection is "None"
    When the COS is created
    Then the COS in group "beta" revision 1 should have condition "Available" with status "True"

  Scenario: Object-level CP overrides phase-level CP
    Given a COS with group "alpha" and revision 1
    And a phase "install" with a ConfigMap "cm-obj-override"
    And the COS is created and becomes Available
    Given a COS with group "beta" and revision 1
    And a phase "install" with a ConfigMap "cm-obj-override"
    And the phase "install" collisionProtection is "None"
    And the last object collisionProtection is "Prevent"
    When the COS is created
    Then the COS in group "beta" revision 1 should have condition "Available" with status "False"

  Scenario: Object-level CP overrides root-level CP
    Given a COS with group "alpha" and revision 1
    And a phase "install" with a ConfigMap "cm-root-obj"
    And the COS is created and becomes Available
    Given a COS with group "beta" and revision 1
    And the COS collisionProtection is "None"
    And a phase "install" with a ConfigMap "cm-root-obj"
    And the last object collisionProtection is "Prevent"
    When the COS is created
    Then the COS in group "beta" revision 1 should have condition "Available" with status "False"

  Scenario: Three-level CP precedence — object wins over phase and root
    Given a COS with group "alpha" and revision 1
    And a phase "install" with a ConfigMap "cm-three-level"
    And the COS is created and becomes Available
    Given a COS with group "beta" and revision 1
    And the COS collisionProtection is "None"
    And a phase "install" with a ConfigMap "cm-three-level"
    And the phase "install" collisionProtection is "None"
    And the last object collisionProtection is "Prevent"
    When the COS is created
    Then the COS in group "beta" revision 1 should have condition "Available" with status "False"

Feature: Collision protection at root, phase, and object levels

  # --- Behavior: Prevent ---

  Scenario: Default collision protection prevents a second group from managing the same object
    Given a COSR with group "alpha" and revision 1
    And a phase "install" with a ConfigMap "cm-contested"
    And the COSR is created and becomes Available
    Given a COSR with group "beta" and revision 1
    And a phase "install" with a ConfigMap "cm-contested"
    When the COSR is created
    Then the COSR in group "beta" revision 1 should have condition "Available" with status "False"
    And the COSR in group "alpha" revision 1 should have condition "Available" with status "True"

  Scenario: Explicit Prevent blocks a second group from managing the same object
    Given a COSR with group "alpha" and revision 1
    And a phase "install" with a ConfigMap "cm-explicit-prevent"
    And the COSR is created and becomes Available
    Given a COSR with group "beta" and revision 1
    And the COSR collisionProtection is "Prevent"
    And a phase "install" with a ConfigMap "cm-explicit-prevent"
    When the COSR is created
    Then the COSR in group "beta" revision 1 should have condition "Available" with status "False"

  Scenario: Prevent blocks adoption of a standalone object
    Given a standalone ConfigMap "cm-standalone-prevent" exists
    And a COSR with group "adopt" and revision 1
    And the COSR collisionProtection is "Prevent"
    And a phase "install" with a ConfigMap "cm-standalone-prevent"
    When the COSR is created
    Then the COSR should have condition "Available" with status "False"

  # --- Behavior: None ---

  Scenario: CollisionProtection None allows a second group to manage the same object
    Given a COSR with group "alpha" and revision 1
    And the COSR collisionProtection is "None"
    And a phase "install" with a ConfigMap "cm-none"
    And the COSR is created and becomes Available
    Given a COSR with group "beta" and revision 1
    And the COSR collisionProtection is "None"
    And a phase "install" with a ConfigMap "cm-none"
    When the COSR is created
    Then the COSR in group "beta" revision 1 should have condition "Available" with status "True"

  # --- Behavior: IfNoController ---

  Scenario: IfNoController adopts a standalone object
    Given a standalone ConfigMap "cm-standalone-adopt" exists
    And a COSR with group "adopt" and revision 1
    And the COSR collisionProtection is "IfNoController"
    And a phase "install" with a ConfigMap "cm-standalone-adopt"
    When the COSR is created
    Then the COSR should have condition "Available" with status "True"
    And the ConfigMap "cm-standalone-adopt" should have an owner reference

  Scenario: IfNoController cannot adopt an object owned by another COSR
    Given a COSR with group "alpha" and revision 1
    And a phase "install" with a ConfigMap "cm-owned"
    And the COSR is created and becomes Available
    Given a COSR with group "beta" and revision 1
    And the COSR collisionProtection is "IfNoController"
    And a phase "install" with a ConfigMap "cm-owned"
    When the COSR is created
    Then the COSR in group "beta" revision 1 should have condition "Available" with status "False"

  # --- Precedence ---

  Scenario: Phase-level CP overrides root-level CP
    Given a COSR with group "alpha" and revision 1
    And a phase "install" with a ConfigMap "cm-phase-override"
    And the COSR is created and becomes Available
    Given a COSR with group "beta" and revision 1
    And the COSR collisionProtection is "Prevent"
    And a phase "install" with a ConfigMap "cm-phase-override"
    And the phase "install" collisionProtection is "None"
    When the COSR is created
    Then the COSR in group "beta" revision 1 should have condition "Available" with status "True"

  Scenario: Object-level CP overrides phase-level CP
    Given a COSR with group "alpha" and revision 1
    And a phase "install" with a ConfigMap "cm-obj-override"
    And the COSR is created and becomes Available
    Given a COSR with group "beta" and revision 1
    And a phase "install" with a ConfigMap "cm-obj-override"
    And the phase "install" collisionProtection is "None"
    And the last object collisionProtection is "Prevent"
    When the COSR is created
    Then the COSR in group "beta" revision 1 should have condition "Available" with status "False"

  Scenario: Object-level CP overrides root-level CP
    Given a COSR with group "alpha" and revision 1
    And a phase "install" with a ConfigMap "cm-root-obj"
    And the COSR is created and becomes Available
    Given a COSR with group "beta" and revision 1
    And the COSR collisionProtection is "None"
    And a phase "install" with a ConfigMap "cm-root-obj"
    And the last object collisionProtection is "Prevent"
    When the COSR is created
    Then the COSR in group "beta" revision 1 should have condition "Available" with status "False"

  Scenario: Three-level CP precedence — object wins over phase and root
    Given a COSR with group "alpha" and revision 1
    And a phase "install" with a ConfigMap "cm-three-level"
    And the COSR is created and becomes Available
    Given a COSR with group "beta" and revision 1
    And the COSR collisionProtection is "None"
    And a phase "install" with a ConfigMap "cm-three-level"
    And the phase "install" collisionProtection is "None"
    And the last object collisionProtection is "Prevent"
    When the COSR is created
    Then the COSR in group "beta" revision 1 should have condition "Available" with status "False"

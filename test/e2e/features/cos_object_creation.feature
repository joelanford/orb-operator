Feature: COS creates managed objects across multiple phases

  Scenario: CRDs in phase 1, instances of each CRD in phases 2, 3, and 4
    Given a COS with group "test" and revision 1
    # Phase 1: create 3 CRDs with explicit Established assertions
    And a phase "crds" with a CRD "widgets" with assertion conditionEqual type "Established" status "True"
    And the phase "crds" also has a CRD "gadgets"
    And the last object has assertion conditionEqual type "Established" status "True"
    And the phase "crds" also has a CRD "doodads"
    And the last object has assertion conditionEqual type "Established" status "True"
    # Phase 2: one instance of each CRD
    And a phase "batch-1" with a "widgets" named "w1"
    And the phase "batch-1" also has a "gadgets" named "g1"
    And the phase "batch-1" also has a "doodads" named "d1"
    # Phase 3: one instance of each CRD
    And a phase "batch-2" with a "widgets" named "w2"
    And the phase "batch-2" also has a "gadgets" named "g2"
    And the phase "batch-2" also has a "doodads" named "d2"
    # Phase 4: one instance of each CRD
    And a phase "batch-3" with a "widgets" named "w3"
    And the phase "batch-3" also has a "gadgets" named "g3"
    And the phase "batch-3" also has a "doodads" named "d3"
    When the COS is created
    # CRDs exist
    Then the CRD "widgets" should exist
    And the CRD "gadgets" should exist
    And the CRD "doodads" should exist
    # All CR instances exist
    And the "widgets" named "w1" should exist
    And the "gadgets" named "g1" should exist
    And the "doodads" named "d1" should exist
    And the "widgets" named "w2" should exist
    And the "gadgets" named "g2" should exist
    And the "doodads" named "d2" should exist
    And the "widgets" named "w3" should exist
    And the "gadgets" named "g3" should exist
    And the "doodads" named "d3" should exist

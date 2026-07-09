Feature: COD status is derived from its COSRs

  Scenario: COD is Available when single Active COSR is Available
    Given a COD named "status-avail"
    And a phase "install" with a ConfigMap "cm-status-avail"
    When the COD is created
    Then the COD "status-avail" should be Available

  Scenario: COD is Unavailable when single Active COSR is not Available
    Given a COD named "status-unavail"
    And a phase "install" with a gated ConfigMap "cm-status-unavail"
    When the COD is created
    Then the COD "status-unavail" should have condition "Available" with status "False" and reason "Unavailable"

  Scenario: COD status updates when COSR status changes
    Given a COD named "status-propagate"
    And a phase "install" with a gated ConfigMap "cm-propagate"
    When the COD is created
    And the gate on ConfigMap "cm-propagate" is opened
    Then the COD "status-propagate" should be Available
    # Close the gate — COD should reflect Unavailable
    When the gate on ConfigMap "cm-propagate" is closed
    Then the COD "status-propagate" should have condition "Available" with status "False" and reason "Unavailable"
    # Reopen the gate — COD should reflect Available again
    When the gate on ConfigMap "cm-propagate" is opened
    Then the COD "status-propagate" should be Available

  Scenario: COD is Progressing when multiple Active COSRs exist
    Given an available COD named "status-progress"
    When the COD template spec is updated with a ConfigMap "cm-progress-2" in phase "install"
    Then the COD "status-progress" should have condition "Available" with status "Unknown" and reason "Progressing"

  Scenario: COD never becomes Unavailable during a rollout
    Given an available COD named "status-rollout"
    When the COD template spec is updated with a ConfigMap "cm-rollout-2" in phase "install"
    Then the COD "status-rollout" should have active revision 2
    And the COD "status-rollout" should become available without becoming unavailable

  Scenario: COD does not archive predecessor while newer revision is unavailable
    Given an available COD named "status-stuck"
    When the COD template spec is updated with a gated ConfigMap "cm-stuck-2" in phase "install"
    Then the COD "status-stuck" should have condition "Available" with status "Unknown" and reason "Progressing"
    And the COD "status-stuck" should have active revision 1
    And the COD "status-stuck" should have active revision 2

  Scenario: COD remains Available when latest revision is manually archived
    Given a COD named "status-archive-latest"
    And a phase "install" with a ConfigMap "cm-sal-1"
    When the COD is created
    Then the COD "status-archive-latest" should be Available
    # Create rev 2 with a gate so it stays Unavailable
    When the COD template spec is updated with a gated ConfigMap "cm-sal-2" in phase "install"
    Then a COSR should exist with group "status-archive-latest" and revision 2
    And the COD "status-archive-latest" should have condition "Available" with status "Unknown" and reason "Progressing"
    # Manually archive rev 2 — rev 1 is still Active and Available
    When the COSR with group "status-archive-latest" and revision 2 lifecycleState is set to "Archived"
    Then the COD "status-archive-latest" should be Available

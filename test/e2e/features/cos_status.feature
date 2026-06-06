Feature: COS status is derived from its COSRs

  Scenario: COS is Available when single Active COSR is Available
    Given a COS named "status-avail"
    And a phase "install" with a ConfigMap "cm-status-avail"
    When the COS is created
    Then the COS "status-avail" should be Available

  Scenario: COS is Unavailable when single Active COSR is not Available
    Given a COS named "status-unavail"
    And a phase "install" with a ConfigMap "cm-status-unavail" with assertion fieldValue path ".data.ready" value "true"
    When the COS is created
    Then the COS "status-unavail" should have condition "Available" with status "False" and reason "Unavailable"

  Scenario: COS status updates when COSR status changes
    Given a COS named "status-propagate"
    And a phase "install" with a ConfigMap "cm-propagate" with assertion celExpression "!has(self.data) || self.data['fail'] != 'true'"
    When the COS is created
    Then the COS "status-propagate" should be Available
    # COSR assertion starts failing — COS should reflect Unavailable
    When the ConfigMap "cm-propagate" field ".data.fail" is set to "true"
    Then the COS "status-propagate" should have condition "Available" with status "False" and reason "Unavailable"
    # COSR assertion recovers — COS should reflect Available again
    When the ConfigMap "cm-propagate" field ".data.fail" is set to "false"
    Then the COS "status-propagate" should be Available

  Scenario: COS is Progressing when multiple Active COSRs exist
    Given an available COS named "status-progress"
    When the COS template spec is updated with a ConfigMap "cm-progress-2" in phase "install"
    Then the COS "status-progress" should have condition "Available" with status "Unknown" and reason "Progressing"

  Scenario: COS never becomes Unavailable during a rollout
    Given an available COS named "status-rollout"
    When the COS template spec is updated with a ConfigMap "cm-rollout-2" in phase "install"
    Then the COS "status-rollout" should have active revision 2
    And the COS "status-rollout" should become available without becoming unavailable

  Scenario: COS remains Available when latest revision is manually archived
    Given a COS named "status-archive-latest"
    And a phase "install" with a ConfigMap "cm-sal-1"
    When the COS is created
    Then the COS "status-archive-latest" should be Available
    # Create rev 2 with a gate so it stays Unavailable
    When the COS template spec is updated with a gated ConfigMap "cm-sal-2" in phase "install"
    Then a COSR should exist with group "status-archive-latest" and revision 2
    And the COS "status-archive-latest" should have condition "Available" with status "Unknown" and reason "Progressing"
    # Manually archive rev 2 — rev 1 is still Active and Available
    When the COSR with group "status-archive-latest" and revision 2 lifecycleState is set to "Archived"
    Then the COS "status-archive-latest" should be Available

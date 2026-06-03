Feature: COS status is derived from its COSRs

  Scenario: COS is Available when single Active COSR is Available
    Given a COS named "status-avail"
    And a phase "install" with a ConfigMap "cm-status-avail"
    When the COS is created
    Then the COS "status-avail" should have condition "Available" with status "True" and reason "Available"

  Scenario: COS is Unavailable when single Active COSR is not Available
    Given a COS named "status-unavail"
    And a phase "install" with a ConfigMap "cm-status-unavail" with assertion fieldValue path ".data.ready" value "true"
    When the COS is created
    Then the COS "status-unavail" should have condition "Available" with status "False" and reason "Unavailable"

  Scenario: COS status updates when COSR status changes
    Given a COS named "status-propagate"
    And a phase "install" with a ConfigMap "cm-propagate" with assertion celExpression "!has(self.data) || self.data['fail'] != 'true'"
    When the COS is created
    Then the COS "status-propagate" should have condition "Available" with status "True" and reason "Available"
    # COSR assertion starts failing — COS should reflect Unavailable
    When the ConfigMap "cm-propagate" field ".data.fail" is set to "true"
    Then the COS "status-propagate" should have condition "Available" with status "False" and reason "Unavailable"
    # COSR assertion recovers — COS should reflect Available again
    When the ConfigMap "cm-propagate" field ".data.fail" is set to "false"
    Then the COS "status-propagate" should have condition "Available" with status "True" and reason "Available"

  Scenario: COS is Progressing when multiple Active COSRs exist
    Given a COS named "status-progress"
    And a phase "install" with a ConfigMap "cm-progress-1"
    When the COS is created
    Then the COS "status-progress" should have condition "Available" with status "True" and reason "Available"
    When the COS template spec is updated with a ConfigMap "cm-progress-2" in phase "install"
    Then the COS "status-progress" should have condition "Available" with status "Unknown" and reason "Progressing"

  Scenario: COS never becomes Unavailable during a rollout
    Given a COS named "status-rollout"
    And a phase "install" with a ConfigMap "cm-rollout-1"
    When the COS is created
    Then the COS "status-rollout" should have condition "Available" with status "True" and reason "Available"
    When the COS template spec is updated with a ConfigMap "cm-rollout-2" in phase "install"
    Then the COS "status-rollout" should have active revision 2
    And the COS "status-rollout" should become available without becoming unavailable

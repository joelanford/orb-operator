Feature: COD progress deadline detects stalled rollouts

  Scenario: COD reports ProgressDeadlineExceeded when COS never completes
    Given a COD named "deadline-exceeded"
    And the COD has progressDeadlineMinutes 500
    And a phase "install" with a gated ConfigMap "cm-blocked"
    When the COD is created
    Then the COD "deadline-exceeded" should have condition "Progressing" with status "False" and reason "ProgressDeadlineExceeded"

  Scenario: COD reports NewClusterObjectSetProgressed when COS completes
    Given a COD named "deadline-success"
    And the COD has progressDeadlineMinutes 500
    And a phase "install" with a ConfigMap "cm-ok"
    When the COD is created
    Then the COD "deadline-success" should be Available
    And the COD "deadline-success" should have condition "Progressing" with status "True" and reason "NewClusterObjectSetProgressed"

  Scenario: COD recovers from ProgressDeadlineExceeded when COS eventually completes
    Given a COD named "deadline-recover"
    And the COD has progressDeadlineMinutes 500
    And a phase "install" with a gated ConfigMap "cm-recover"
    When the COD is created
    Then the COD "deadline-recover" should have condition "Progressing" with status "False" and reason "ProgressDeadlineExceeded"
    When the gate on ConfigMap "cm-recover" is opened
    Then the COD "deadline-recover" should be Available
    And the COD "deadline-recover" should have condition "Progressing" with status "True" and reason "NewClusterObjectSetProgressed"

  Scenario: COD without progressDeadlineMinutes reports NewClusterObjectSetProgressing
    Given a COD named "no-deadline"
    And a phase "install" with a gated ConfigMap "cm-no-deadline"
    When the COD is created
    Then the COD "no-deadline" should have condition "Progressing" with status "True" and reason "NewClusterObjectSetProgressing"

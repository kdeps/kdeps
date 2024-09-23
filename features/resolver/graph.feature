Feature: Graph
  Scenario: Dependency Resolution
    Given an ai agent with "100" resources
    When I load the workflow resources
    Then I was able to see the "100" top-down dependencies
    And each resource are reloaded when opened

  Scenario: Dependency Resolution 2
    Given an ai agent with "100" resources that was configured differently
    When I load the workflow resources
    Then I was able to see the "100" top-down dependencies
    And each resource are reloaded when opened

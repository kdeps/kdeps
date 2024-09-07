Feature: Workflow enforcer
  Background:
    Given the current directory is "/current/directory"
    And a system configuration is defined
    And an agent folder "my-agent" exists in the current directory

  Scenario: Find a workflow.pkl configuration file in a folder
    When a file "workflow.pkl" exists in the "my-agent" folder
    And it have a workflow amends line on top of the file
    And it have a "kdeps.com" amends url line on top of the file
    Then it is a valid agent

  Scenario: Find a workflow.pkl configuration file in a folder without an amends line
    When a file "workflow.pkl" exists in the "my-agent" folder
    And it does not have an workflow amends line on top of the file
    Then it is an invalid agent

  Scenario: Find a workflow.pkl configuration file in a folder with a different url in the amends line
    When a file "workflow.pkl" exists in the "my-agent" folder
    And it have a workflow amends line on top of the file
    And it have a "domain.com" amends url line on top of the file
    Then it is an invalid agent

  Scenario: Find a workflow.pkl configuration file in a folder with a different url in the amends line
    When a file "workflow.pkl" exists in the "my-agent" folder
    And it have a other amends line on top of the file
    And it have a "kdeps.com" amends url line on top of the file
    Then it is an invalid agent

  Scenario: Find a single configuration in a folder with a different name
    When a file "agent.pkl" exists in the "my-agent" folder
    Then it is an invalid agent

  Scenario: Multiple PKL found in a folder with workflow.pkl
    When a file "workflow.pkl" exists in the "my-agent" folder
    And a file "extras.pkl" also exists in the "my-agent" folder
    Then it is a valid agent

  Scenario: Multiple PKL found in a folder with a different name
    When a file "agent.pkl" exists in the "my-agent" folder
    And a file "extras.pkl" also exists in the "my-agent" folder
    Then it is a invalid agent

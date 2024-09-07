Feature: Workflow enforcer
  Background:
    Given the current directory is "/current/directory"
    And a system configuration is defined
    And an agent folder "my-agent" exists in the current directory
    And we have a blank workflow file

  Scenario: Workflow file exists in the "my-agent" with an amends line
    Given it have a workflow amends line on top of the file
    And it have a "kdeps.com" amends url line on top of the file
    When a file "workflow.pkl" exists in the "my-agent"
    Then it is a valid agent

  Scenario: Workflow file exists in the "my-agent" without an amends line
    Given it does not have a workflow amends line on top of the file
    When a file "workflow.pkl" exists in the "my-agent"
    Then it is an invalid agent

  Scenario: Workflow file exists in the "my-agent" with an amends line
    Given it have a workflow amends line on top of the file
    And it have a "domain.com" amends url line on top of the file
    When a file "workflow.pkl" exists in the "my-agent"
    Then it is an invalid agent

  Scenario: Workflow file exists in the "my-agent" with an amends line and different pkl file
    Given it have a workflow amends line on top of the file
    And it have a "kdeps.com" amends url line on top of the file
    When a file "workflow1.pkl" exists in the "my-agent"
    Then it is an invalid agent

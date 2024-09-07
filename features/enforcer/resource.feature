Feature: Resource enforcer
  Background:
    Given the current directory is "/current/directory"
    And a system configuration is defined
    And an agent folder "my-agent" exists in the current directory
    And we have a blank workflow file
    And it have a workflow amends line on top of the file
    And it have a "kdeps.com" amends url line on top of the file
    And a folder named "resources" exists in the "my-agent"
    And a folder named "data" exists in the "my-agent"
    And a file "workflow.pkl" exists in the "my-agent"
    And it is a valid pkl file
    And it is a valid agent

  Scenario: Resource file exists in the "my-agent/resources" with an amends line
    Given it have a resource amends line on top of the file
    And it have a "kdeps.com" amends url line on top of the file
    When a file "resource.pkl" exists in the "my-agent/resources"
    Then it is a valid agent

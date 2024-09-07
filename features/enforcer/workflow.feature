Feature: Workflow enforcer
  Background:
    Given the current directory is "/current/directory"
    And a system configuration is defined
    And an agent folder "my-agent" exists in the current directory
    And we have a blank workflow file

  Scenario: Workflow file exists in the "my-agent" with an amends line
    Given it have a workflow amends line on top of the file
    And it have a "kdeps.com" amends url line on top of the file
    And a folder named "resources" exists in the "my-agent"
    And a folder named "data" exists in the "my-agent"
    When a file "workflow.pkl" exists in the "my-agent"
    Then it is a valid pkl file
    Then it is a valid agent

  Scenario: Workflow file exists in the "my-agent" without an amends line
    Given it does not have a workflow amends line on top of the file
    And a folder named "resources" exists in the "my-agent"
    And a folder named "data" exists in the "my-agent"
    When a file "workflow.pkl" exists in the "my-agent"
    Then it is an invalid pkl file
    Then it is a valid agent

  Scenario: Workflow file exists in the "my-agent" with an amends line
    Given it have a workflow amends line on top of the file
    And it have a "domain.com" amends url line on top of the file
    And a folder named "resources" exists in the "my-agent"
    And a folder named "data" exists in the "my-agent"
    When a file "workflow.pkl" exists in the "my-agent"
    Then it is an invalid pkl file
    Then it is a valid agent

  Scenario: Workflow file exists in the "my-agent" with an amends line and different pkl file
    Given it have a workflow amends line on top of the file
    And it have a "kdeps.com" amends url line on top of the file
    And a folder named "resources" exists in the "my-agent"
    And a folder named "data" exists in the "my-agent"
    When a file "workflow1.pkl" exists in the "my-agent"
    Then it is an invalid pkl file
    Then it is an invalid agent

  Scenario: Workflow file exists in the "my-agent" with an amends line with multiple files
    Given it have a workflow amends line on top of the file
    And it have a "kdeps.com" amends url line on top of the file
    And a folder named "resources" exists in the "my-agent"
    And a folder named "data" exists in the "my-agent"
    And a file "others.txt" exists in the "my-agent"
    When a file "workflow.pkl" exists in the "my-agent"
    Then it is a valid pkl file
    Then it is an invalid agent

  Scenario: Workflow file exists in the "my-agent" with an amends line with an ignored file
    Given it have a workflow amends line on top of the file
    And it have a "kdeps.com" amends url line on top of the file
    And a folder named "resources" exists in the "my-agent"
    And a folder named "data" exists in the "my-agent"
    And a file ".kdeps.pkl" exists in the "my-agent"
    When a file "workflow.pkl" exists in the "my-agent"
    Then it is a valid pkl file
    Then it is a valid agent

  Scenario: Workflow file exists in the "my-agent" with an amends line without the resources folder
    Given it have a workflow amends line on top of the file
    And it have a "kdeps.com" amends url line on top of the file
    And a folder named "data" exists in the "my-agent"
    When a file "workflow.pkl" exists in the "my-agent"
    Then it is a valid pkl file
    Then it is a valid agent

  Scenario: Workflow file exists in the "my-agent" with an amends line without the data folder
    Given it have a workflow amends line on top of the file
    And it have a "kdeps.com" amends url line on top of the file
    And a folder named "resources" exists in the "my-agent"
    When a file "workflow.pkl" exists in the "my-agent"
    Then it is a valid pkl file
    Then it is a valid agent

  Scenario: Workflow file exists in the "my-agent" with an amends line without any folder
    Given it have a workflow amends line on top of the file
    And it have a "kdeps.com" amends url line on top of the file
    When a file "workflow.pkl" exists in the "my-agent"
    Then it is a valid pkl file
    Then it is a valid agent

  Scenario: Workflow file exists in the "my-agent" with an amends line without any allowed folder and a "other" folder
    Given it have a workflow amends line on top of the file
    And it have a "kdeps.com" amends url line on top of the file
    And a folder named "other" exists in the "my-agent"
    When a file "workflow.pkl" exists in the "my-agent"
    Then it is a valid pkl file
    Then it is an invalid agent

  Scenario: Workflow file exists in the "my-agent" with an amends line with all allowed folders and a "other" folder
    Given it have a workflow amends line on top of the file
    And it have a "kdeps.com" amends url line on top of the file
    And a folder named "resources" exists in the "my-agent"
    And a folder named "data" exists in the "my-agent"
    And a folder named "other" exists in the "my-agent"
    When a file "workflow.pkl" exists in the "my-agent"
    Then it is a valid pkl file
    Then it is an invalid agent

  Scenario: Workflow file exists in the "my-agent" with an amends line with allowed folder and subfiles
    Given it have a workflow amends line on top of the file
    And it have a "kdeps.com" amends url line on top of the file
    And a folder named "resources" exists in the "my-agent"
    And a folder named "data" exists in the "my-agent"
    When a file "workflow.pkl" exists in the "resources"
    When a file "workflow.pkl" exists in the "data"
    When a file "workflow.pkl" exists in the "my-agent"
    Then it is a valid pkl file
    Then it is a valid agent

Feature: Archiver of AI Agents
  Background:
    Given the ".kdeps" system folder exists
    And an ai agent on "ai-agent" folder exists
    And it has a workflow file that has name property "myAgent" and version property "1.0.0"
    And it has a resource file with id property "helloWorld"
    And theres a data file

  Scenario: Valid ai agent will be packaged
    When the project is valid
    When the pkl files is valid
    Then the project will be archived to "myAgent-1.0.0.kdeps"

  Scenario: Invalid ai agent will not be packaged
    When the pkl files is invalid
    When the project is invalid
    Then the project will not be archived to "myAgent-1.0.0.kdeps"

  Scenario: Kdeps packages will be extracted
    When a kdeps archive "myAgent-1.0.0.kdeps" is passed
    Then the content of that archive file will be extracted to ".kdeps/agents/myAgent-1.0.0"

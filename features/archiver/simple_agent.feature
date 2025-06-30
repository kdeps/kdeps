Feature: Simple AI Agent Compilation

  Scenario: Compile simple agent without dependencies
    Given the ".kdeps" system folder exists
    And an ai agent on "simple-agent" folder exists
    And the resources and data folder exists
    And theres a data file
    And it has a workflow file that has name property "simpleAgent" and version property "1.0.0" and default action "sayHello"
    And it has a "resource1.pkl" file with no dependency with id property "sayHello"
    Given the project is valid
    And the pkl files is valid
    When the project is compiled
    And it will be stored to "agents/simpleAgent/1.0.0/workflow.pkl"
    And the workflow action configuration will be rewritten to "@simpleAgent/sayHello:1.0.0"
    And the data files will be copied to "agents/simpleAgent/1.0.0/data/simpleAgent/1.0.0"
    And the package file "simpleAgent-1.0.0.kdeps" will be created 
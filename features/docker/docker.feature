Feature: Docker integration
  Background:
    Given a system configuration is defined
    And existing docker container "kdeps-cpu-test" system is deleted
    And the docker gpu "cpu" is defined
    And custom <packages> has been defined
      | ftp      |
      | git      |
    And llm <models> has been installed
      | llama3.1 |
      | llama2   |
    And copy the necessary files to make it ready to be used
    And the docker subsystem "test" is invoked


  Scenario: Docker will start with the given image
    Then it should initialize the "kdeps-cpu-test" docker subsystem

Feature: Docker integration
  Background:
    Given ".kdeps" directory exists in the "HOME" directory

  Scenario: Basic default build
    Given a ".kdeps.pkl" system configuration file with dockerGPU "cpu" and runMode "docker" is defined in the "HOME" directory
    And a valid ai-agent "agentX" is present in the "HOME" directory with packages "apt-utils, git, ruby" and models "tinyllama"
    And the valid ai-agent "agentX" has been compiled as "agentX-1.0.0.kdeps" in the packages directory
    When kdeps open the package "agentX-1.0.0.kdeps" and extract it's content to the agents directory
    Then it should check if the docker container "kdeps-agentX-1.0.0-cpu" is not running
    Then it should create the Dockerfile for the agent in the "agentX/1.0.0" directory with package "git" and copy the kdeps package to the "/agent" directory
    And it should run the container build step for "kdeps-agentX-1.0.0-cpu"
    And it should start the container "kdeps-agentX-1.0.0-cpu"

  # Scenario: Custom build
  #   Given a ".kdeps.pkl" system configuration file with dockerGPU "cpu" and runMode "docker" is defined in the "HOME" directory
  #   And a valid ai-agent "agentX" is present in the "HOME" directory
  #   And the valid ai-agent "agentX" has been compiled as "agentX-1.0.0.kdeps" in the packages directory
  #   When kdeps open the package "agentX-1.0.0.kdeps" and extract it's content to the agents directory
  #   Then kdeps should parse the workflow of the "agentX" agent version "1.0.0" in the agents directory with model "tinyllama, tinydolphin" and packages "wget, curl"
  #   And it should check if the docker container "kdeps-agentX-1.0.0-cpu" is not running
  #   Then it should create the Dockerfile for the agent in the "agentX/1.0.0" directory with model "tinyllama, tinydolphin, llama3.1" and package "git, wget, curl" and copy the kdeps package to the "/agent" directory
  #   And it should run the container build step for "kdeps-agentX-1.0.0-cpu"
  #   And it should start the container "kdeps-agentX-1.0.0-cpu"

Feature: Docker integration
  Background:
    Given ".kdeps" directory exists in the "HOME" directory

  Scenario: Basic default build
    Given a ".kdeps.pkl" system configuration file with dockerGPU "cpu" and runMode "docker" is defined in the "HOME" directory
    And a valid ai-agent "agentX" is present in the "HOME" directory with packages "apt-utils, git, ruby" and models "tinyllama"
    And the valid ai-agent "agentX" has been compiled as "agentX-1.0.0.kdeps" in the packages directory
    When kdeps open the package "agentX-1.0.0.kdeps" and extract it's content to the agents directory
    Then it should create the Dockerfile for the agent in the "agentX/1.0.0" directory with package "git" and copy the kdeps package to the "/agents" directory
    And it should run the container build step for "kdeps-agentX-1.0.0-cpu"
    And it should start the container "kdeps-agentX-1.0.0-cpu"
    # And the Docker entrypoint should be "/bin/kdeps"
    # And the command should be run "agentX" action by default

  Scenario: Ability to bootstrap the Docker environment
    # Given a kdeps docker image with kdeps entrypoint
    # When the docker image container is started
    Then kdeps will check the presence of the "/.dockerenv" file
    And it will install the models defined in the ".kdeps.pkl" configuration if found

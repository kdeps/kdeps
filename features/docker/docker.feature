Feature: Docker integration
  Background:
    Given a system configuration is defined

  Scenario: Docker will start with the given image
    Given the docker image "alpine:3.14" is defined
    When the docker subsystem is invoked
    Then it initialize the "alpine:3.14" docker subsystem
    And copy the necessary files to make it ready to be used

Feature: Find and generate config
  Background:
    Given the home directory is "/home/user"
    And the current directory is "/current/directory"

  Scenario: Configuration file exists in the current directory
    Given a file ".kdeps.pkl" exists in the current directory
    When the configuration is loaded
    Then the configuration file is "/current/directory/.kdeps.pkl"

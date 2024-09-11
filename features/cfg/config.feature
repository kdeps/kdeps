Feature: Find and generate config
  Background:
    Given the home directory is "/home/user"
    And the current directory is "/current/directory"

  Scenario: Configuration file exists in the current directory
    Given a file ".kdeps.pkl" exists in the current directory
    When the configuration is loaded in the current directory
    Then the configuration file is "/current/directory/.kdeps.pkl"

  Scenario: Configuration file exists in the home directory
    Given a file ".kdeps.pkl" exists in the home directory
    When the configuration is loaded in the home directory
    Then the configuration file is "/home/user/.kdeps.pkl"

  Scenario: No Configuration file exists in both home and current directory
    And a file ".kdeps.pkl" does not exists in the home or current directory
    When the configuration fails to load any configuration
    Then the configuration file will be generated to "/home/user/.kdeps.pkl"
    And the configuration will be edited
    And the configuration will be validated

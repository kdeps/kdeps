Feature: Config enforcer
  Background:
    Given the home directory is "/home/user"
    And the current directory is "/current/directory"
    And we have a blank config file

  Scenario: Configuration file exists in the current directory with an amends line
    Given it have a config amends line on top of the file
    And it have a "kdeps.com" amends url line on top of the file
    When a file ".kdeps.pkl" exists in the current directory
    Then it is a valid configuration file

  Scenario: Configuration file exists in the current directory without an amends line
    Given it does not have a config amends line on top of the file
    When a file ".kdeps.pkl" exists in the current directory
    Then it is an invalid configuration file

  Scenario: Configuration file exists in the current directory with an amends line
    Given it have a config amends line on top of the file
    And it have a "domain.com" amends url line on top of the file
    When a file ".kdeps.pkl" exists in the current directory
    Then it is an invalid configuration file

  Scenario: Configuration file exists in the current directory with an amends line and different pkl file
    Given it have a config amends line on top of the file
    And it have a "kdeps.com" amends url line on top of the file
    When a file ".kdeps1.pkl" exists in the current directory
    Then it is an invalid configuration file

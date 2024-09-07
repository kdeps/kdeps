# Feature: Resources enforcer
#   Background:
#     Given the current directory is "/current/directory"
#     And a system configuration is defined
#     And an agent folder "my-agent" exists in the current directory
#     And a file "workflow.pkl" exists in the "my-agent" folder
#     And it have a workflow amends line on top of the file
#     And it have a "kdeps.com" amends url line on top of the file
#     And it is a valid agent

#   Scenario: Find a workflow.pkl configuration file without a resources
#     When the valid workflow does not have a resources
#     Then it is a valid agent with a warning

#   Scenario: Find a workflow.pkl configuration file with resources and valid amends line
#     When the valid workflow have the following <resources>
#       | resource1.pkl   |
#       | resource2.pkl   |
#       | resource3.pkl   |
#     And all resources have a valid amends line
#     And all resources have a valid domain
#     Then it is a valid agent without a warning

#   Scenario: Find a workflow.pkl configuration file with resources and some invalid amends file line
#     When the valid workflow have the following <resources>
#       | resource1.pkl   |
#       | resource2.pkl   |
#       | resource3.pkl   |
#     And all resources have an amends line
#     And "resource2.pkl" have a "Other.pkl" file in the amends
#     And all resources have a valid domain
#     Then it is an invalid agent

#   Scenario: Find a workflow.pkl configuration file with resources and some invalid amends domain line
#     When the valid workflow have the following <resources>
#       | resource1.pkl   |
#       | resource2.pkl   |
#       | resource3.pkl   |
#     And all resources have an amends line
#     And "resource3.pkl" have a "domain.com" url in the amends
#     Then it is an invalid agent

#   Scenario: Find a workflow.pkl configuration file with resources and some missing amends line
#     When the valid workflow have the following <resources>
#       | resource1.pkl   |
#       | resource2.pkl   |
#       | resource3.pkl   |
#     And "resource1.pkl" does not have an amends line
#     Then it is an invalid agent

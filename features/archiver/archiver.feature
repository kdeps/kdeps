Feature: Archiver of AI Agents
  Background:
    Given the ".kdeps" system folder exists
    And an ai agent on "ai-agent" folder exists
    And the resources and data folder exists
    And theres a data file
    And it has a workflow file that has name property "myAgent" and version property "1.0.0" and default action "helloWorld"
    And it has a "resource1.pkl" file with no dependency with id property "helloWorld"
    And it has a "resource2.pkl" file with id property "helloWorld2" and dependent on "helloWorld"
    And it has a "resource3.pkl" file with id property "helloWorld3" and dependent on "helloWorld2, helloWorld"
    And it has a "resource4.pkl" file with id property "helloWorld4" and dependent on "nonexistent, helloWorld3, helloWorld2, helloWorld"

  Scenario: Project without external dependencies will be compiled to system agent folder
    Given the project is valid
    And the pkl files is valid
    When the project is compiled
    And it will be stored to "agents/myAgent/1.0.0/workflow.pkl"
    And the workflow action configuration will be rewritten to "@myAgent/helloWorld:1.0.0"
    And the resource id for "myAgent_helloWorld-1.0.0.pkl" will be rewritten to "@myAgent/helloWorld:1.0.0"
    And the resource id for "myAgent_helloWorld2-1.0.0.pkl" will be "@myAgent/helloWorld2:1.0.0" and dependency "@myAgent/helloWorld:1.0.0"
    And the resource id for "myAgent_helloWorld3-1.0.0.pkl" will be "@myAgent/helloWorld3:1.0.0" and dependency "@myAgent/helloWorld2:1.0.0"
    And the resource id for "myAgent_helloWorld4-1.0.0.pkl" will be "@myAgent/helloWorld4:1.0.0" and dependency "@myAgent/helloWorld3:1.0.0"
    And the data files will be copied to "agents/myAgent/1.0.0/data/myAgent/1.0.0"
    And the package file "myAgent-1.0.0.kdeps" will be created

  Scenario: Project with external dependencies with version will copy the resources and data and be compiled to system agent folder (complete agent name, version and action details)
    Given the project is valid
    And the pkl files is valid
    When the project is compiled
    And it will be stored to "agents/myAgent/1.0.0/workflow.pkl"
    And the workflow action configuration will be rewritten to "@myAgent/helloWorld:1.0.0"
    And the resource id for "myAgent_helloWorld-1.0.0.pkl" will be rewritten to "@myAgent/helloWorld:1.0.0"
    And the resource id for "myAgent_helloWorld2-1.0.0.pkl" will be "@myAgent/helloWorld2:1.0.0" and dependency "@myAgent/helloWorld:1.0.0"
    And the resource id for "myAgent_helloWorld3-1.0.0.pkl" will be "@myAgent/helloWorld3:1.0.0" and dependency "@myAgent/helloWorld2:1.0.0"
    And the resource id for "myAgent_helloWorld4-1.0.0.pkl" will be "@myAgent/helloWorld4:1.0.0" and dependency "@myAgent/helloWorld3:1.0.0"
    And the data files will be copied to "agents/myAgent/1.0.0/data/myAgent/1.0.0"
    And the package file "myAgent-1.0.0.kdeps" will be created
    And an ai agent on "awesome-agent" folder exists
    And the resources and data folder exists
    And theres a data file
    And it has a workflow file that has name property "externalAgent" and version property "1.0.0" and default action "fooBar" and workspaces "@myAgent/helloWorld4:1.0.0"
    And it has a "resource1.pkl" file with no dependency with id property "fooBar"
    And it has a "resource2.pkl" file with id property "fooBar2" and dependent on "fooBar, @myAgent/helloWorld:1.0.0"
    And it has a "resource3.pkl" file with id property "fooBar3" and dependent on "fooBar2, fooBar, @myAgent/helloWorld3:1.0.0"
    And it has a "resource4.pkl" file with id property "fooBar4" and dependent on "nonexistent, fooBar3, @myAgent/helloWorld2:1.0.0, fooBar2, fooBar, @myAgent/helloWorld4:1.0.0"
    And the project is valid
    And the pkl files is valid
    And the project is compiled
    And it will be stored to "agents/externalAgent/1.0.0/workflow.pkl"
    And the workflow action configuration will be rewritten to "@externalAgent/fooBar:1.0.0"
    And the resource id for "externalAgent_fooBar-1.0.0.pkl" will be rewritten to "@externalAgent/fooBar:1.0.0"
    And the resource id for "externalAgent_fooBar2-1.0.0.pkl" will be "@externalAgent/fooBar2:1.0.0" and dependency "@myAgent/helloWorld:1.0.0"
    And the resource id for "externalAgent_fooBar2-1.0.0.pkl" will be "@externalAgent/fooBar2:1.0.0" and dependency "@externalAgent/fooBar:1.0.0"
    And the resource id for "externalAgent_fooBar3-1.0.0.pkl" will be "@externalAgent/fooBar3:1.0.0" and dependency "@myAgent/helloWorld3:1.0.0"
    And the resource id for "externalAgent_fooBar3-1.0.0.pkl" will be "@externalAgent/fooBar3:1.0.0" and dependency "@myAgent/helloWorld3:1.0.0"
    And the resource id for "externalAgent_fooBar4-1.0.0.pkl" will be rewritten to "@externalAgent/fooBar4:1.0.0"
    And the resource id for "externalAgent_fooBar4-1.0.0.pkl" will be "@externalAgent/fooBar4:1.0.0" and dependency "@myAgent/helloWorld2:1.0.0"
    And the data files will be copied to "agents/externalAgent/1.0.0/data/externalAgent/1.0.0"
    And the package file "externalAgent-1.0.0.kdeps" will be created
    And the resource file "myAgent_helloWorld4-1.0.0.pkl" exists in the "externalAgent" agent "1.0.0"
    And the data files will be copied to "agents/externalAgent/1.0.0/data/myAgent/1.0.0"

  Scenario: Project with external dependencies with version will copy the resources and data and be compiled to system agent folder (agent name and version provided but no action provided)
    Given the ".kdeps" system folder exists
    And an ai agent on "ai-agent" folder exists
    And the resources and data folder exists
    And theres a data file
    And it has a workflow file that has name property "newAgent" and version property "1.0.0" and default action "helloWorld"
    And it has a "resource1.pkl" file with no dependency with id property "helloWorld"
    And it has a "resource2.pkl" file with id property "helloWorld2" and dependent on "helloWorld"
    And it has a "resource3.pkl" file with id property "helloWorld3" and dependent on "helloWorld2, helloWorld"
    And it has a "resource4.pkl" file with id property "helloWorld4" and dependent on "nonexistent, helloWorld3, helloWorld2, helloWorld"
    Given the project is valid
    And the pkl files is valid
    When the project is compiled
    And it will be stored to "agents/newAgent/1.0.0/workflow.pkl"
    And the workflow action configuration will be rewritten to "@newAgent/helloWorld:1.0.0"
    And the resource id for "newAgent_helloWorld-1.0.0.pkl" will be rewritten to "@newAgent/helloWorld:1.0.0"
    And the resource id for "newAgent_helloWorld2-1.0.0.pkl" will be "@newAgent/helloWorld2:1.0.0" and dependency "@newAgent/helloWorld:1.0.0"
    And the resource id for "newAgent_helloWorld3-1.0.0.pkl" will be "@newAgent/helloWorld3:1.0.0" and dependency "@newAgent/helloWorld2:1.0.0"
    And the resource id for "newAgent_helloWorld4-1.0.0.pkl" will be "@newAgent/helloWorld4:1.0.0" and dependency "@newAgent/helloWorld3:1.0.0"
    And the data files will be copied to "agents/newAgent/1.0.0/data/newAgent/1.0.0"
    And the package file "newAgent-1.0.0.kdeps" will be created
    And an ai agent on "awesome-agent" folder exists
    And the resources and data folder exists
    And theres a data file
    And it has a workflow file that has name property "acmeCoAgent" and version property "1.0.0" and default action "fooBar" and workspaces "@newAgent:1.0.0"
    And it has a "resource1.pkl" file with no dependency with id property "fooBar"
    And it has a "resource2.pkl" file with id property "fooBar2" and dependent on "fooBar, @newAgent/helloWorld:1.0.0"
    And it has a "resource3.pkl" file with id property "fooBar3" and dependent on "fooBar2, fooBar, @newAgent/helloWorld3:1.0.0"
    And it has a "resource4.pkl" file with id property "fooBar4" and dependent on "nonexistent, fooBar3, @newAgent/helloWorld2:1.0.0, fooBar2, fooBar, @newAgent/helloWorld4:1.0.0"
    And the project is valid
    And the pkl files is valid
    And the project is compiled
    And it will be stored to "agents/acmeCoAgent/1.0.0/workflow.pkl"
    And the workflow action configuration will be rewritten to "@acmeCoAgent/fooBar:1.0.0"
    And the resource id for "acmeCoAgent_fooBar-1.0.0.pkl" will be rewritten to "@acmeCoAgent/fooBar:1.0.0"
    And the resource id for "acmeCoAgent_fooBar2-1.0.0.pkl" will be "@acmeCoAgent/fooBar2:1.0.0" and dependency "@newAgent/helloWorld:1.0.0"
    And the resource id for "acmeCoAgent_fooBar2-1.0.0.pkl" will be "@acmeCoAgent/fooBar2:1.0.0" and dependency "@acmeCoAgent/fooBar:1.0.0"
    And the resource id for "acmeCoAgent_fooBar3-1.0.0.pkl" will be "@acmeCoAgent/fooBar3:1.0.0" and dependency "@newAgent/helloWorld3:1.0.0"
    And the resource id for "acmeCoAgent_fooBar3-1.0.0.pkl" will be "@acmeCoAgent/fooBar3:1.0.0" and dependency "@newAgent/helloWorld3:1.0.0"
    And the resource id for "acmeCoAgent_fooBar4-1.0.0.pkl" will be rewritten to "@acmeCoAgent/fooBar4:1.0.0"
    And the resource id for "acmeCoAgent_fooBar4-1.0.0.pkl" will be "@acmeCoAgent/fooBar4:1.0.0" and dependency "@newAgent/helloWorld2:1.0.0"
    And the data files will be copied to "agents/acmeCoAgent/1.0.0/data/acmeCoAgent/1.0.0"
    And the package file "acmeCoAgent-1.0.0.kdeps" will be created
    And the resource file "newAgent_helloWorld4-1.0.0.pkl" exists in the "acmeCoAgent" agent "1.0.0"
    And the data files will be copied to "agents/acmeCoAgent/1.0.0/data/newAgent/1.0.0"

  Scenario: Project with external dependencies with version will copy the resources and data and be compiled to system agent folder (agent name and action provided but no version provided)
    Given the ".kdeps" system folder exists
    And an ai agent on "agentx" folder exists
    And the resources and data folder exists
    And theres a data file
    And it has a workflow file that has name property "awesomeAgent" and version property "1.0.0" and default action "helloWorld"
    And it has a "resource1.pkl" file with no dependency with id property "helloWorld"
    And it has a "resource2.pkl" file with id property "helloWorld2" and dependent on "helloWorld"
    And it has a "resource3.pkl" file with id property "helloWorld3" and dependent on "helloWorld2, helloWorld"
    And it has a "resource4.pkl" file with id property "helloWorld4" and dependent on "nonexistent, helloWorld3, helloWorld2, helloWorld"
    Given the project is valid
    And the pkl files is valid
    When the project is compiled
    And it will be stored to "agents/awesomeAgent/1.0.0/workflow.pkl"
    And the workflow action configuration will be rewritten to "@awesomeAgent/helloWorld:1.0.0"
    And the resource id for "awesomeAgent_helloWorld-1.0.0.pkl" will be rewritten to "@awesomeAgent/helloWorld:1.0.0"
    And the resource id for "awesomeAgent_helloWorld2-1.0.0.pkl" will be "@awesomeAgent/helloWorld2:1.0.0" and dependency "@awesomeAgent/helloWorld:1.0.0"
    And the resource id for "awesomeAgent_helloWorld3-1.0.0.pkl" will be "@awesomeAgent/helloWorld3:1.0.0" and dependency "@awesomeAgent/helloWorld2:1.0.0"
    And the resource id for "awesomeAgent_helloWorld4-1.0.0.pkl" will be "@awesomeAgent/helloWorld4:1.0.0" and dependency "@awesomeAgent/helloWorld3:1.0.0"
    And the data files will be copied to "agents/awesomeAgent/1.0.0/data/awesomeAgent/1.0.0"
    And the package file "awesomeAgent-1.0.0.kdeps" will be created
    And an ai agent on "awesome-agent" folder exists
    And the resources and data folder exists
    And theres a data file
    And it has a workflow file that has name property "xAgent" and version property "1.0.0" and default action "fooBar" and workspaces "@awesomeAgent/helloWorld"
    And it has a "resource1.pkl" file with no dependency with id property "fooBar"
    And it has a "resource2.pkl" file with id property "fooBar2" and dependent on "fooBar, @awesomeAgent/helloWorld:1.0.0"
    And it has a "resource3.pkl" file with id property "fooBar3" and dependent on "fooBar2, fooBar, @awesomeAgent/helloWorld3:1.0.0"
    And it has a "resource4.pkl" file with id property "fooBar4" and dependent on "nonexistent, fooBar3, @awesomeAgent/helloWorld2:1.0.0, fooBar2, fooBar, @awesomeAgent/helloWorld4:1.0.0"
    And the project is valid
    And the pkl files is valid
    And the project is compiled
    And it will be stored to "agents/xAgent/1.0.0/workflow.pkl"
    And the workflow action configuration will be rewritten to "@xAgent/fooBar:1.0.0"
    And the resource id for "xAgent_fooBar-1.0.0.pkl" will be rewritten to "@xAgent/fooBar:1.0.0"
    And the resource id for "xAgent_fooBar2-1.0.0.pkl" will be "@xAgent/fooBar2:1.0.0" and dependency "@awesomeAgent/helloWorld:1.0.0"
    And the resource id for "xAgent_fooBar2-1.0.0.pkl" will be "@xAgent/fooBar2:1.0.0" and dependency "@xAgent/fooBar:1.0.0"
    And the resource id for "xAgent_fooBar3-1.0.0.pkl" will be "@xAgent/fooBar3:1.0.0" and dependency "@awesomeAgent/helloWorld3:1.0.0"
    And the resource id for "xAgent_fooBar3-1.0.0.pkl" will be "@xAgent/fooBar3:1.0.0" and dependency "@awesomeAgent/helloWorld3:1.0.0"
    And the resource id for "xAgent_fooBar4-1.0.0.pkl" will be rewritten to "@xAgent/fooBar4:1.0.0"
    And the resource id for "xAgent_fooBar4-1.0.0.pkl" will be "@xAgent/fooBar4:1.0.0" and dependency "@awesomeAgent/helloWorld2:1.0.0"
    And the data files will be copied to "agents/xAgent/1.0.0/data/xAgent/1.0.0"
    And the package file "xAgent-1.0.0.kdeps" will be created
    And the resource file "awesomeAgent_helloWorld4-1.0.0.pkl" exists in the "xAgent" agent "1.0.0"
    And the data files will be copied to "agents/xAgent/1.0.0/data/awesomeAgent/1.0.0"

  Scenario: Project with external dependencies with version will copy the resources and data and be compiled to system agent folder (only the agent name was provided)
    Given the ".kdeps" system folder exists
    And an ai agent on "agentxyz" folder exists
    And the resources and data folder exists
    And theres a data file
    And it has a workflow file that has name property "abcAgent" and version property "1.0.0" and default action "helloWorld"
    And it has a "resource1.pkl" file with no dependency with id property "helloWorld"
    And it has a "resource2.pkl" file with id property "helloWorld2" and dependent on "helloWorld"
    And it has a "resource3.pkl" file with id property "helloWorld3" and dependent on "helloWorld2, helloWorld"
    And it has a "resource4.pkl" file with id property "helloWorld4" and dependent on "nonexistent, helloWorld3, helloWorld2, helloWorld"
    Given the project is valid
    And the pkl files is valid
    When the project is compiled
    And it will be stored to "agents/abcAgent/1.0.0/workflow.pkl"
    And the workflow action configuration will be rewritten to "@abcAgent/helloWorld:1.0.0"
    And the resource id for "abcAgent_helloWorld-1.0.0.pkl" will be rewritten to "@abcAgent/helloWorld:1.0.0"
    And the resource id for "abcAgent_helloWorld2-1.0.0.pkl" will be "@abcAgent/helloWorld2:1.0.0" and dependency "@abcAgent/helloWorld:1.0.0"
    And the resource id for "abcAgent_helloWorld3-1.0.0.pkl" will be "@abcAgent/helloWorld3:1.0.0" and dependency "@abcAgent/helloWorld2:1.0.0"
    And the resource id for "abcAgent_helloWorld4-1.0.0.pkl" will be "@abcAgent/helloWorld4:1.0.0" and dependency "@abcAgent/helloWorld3:1.0.0"
    And the data files will be copied to "agents/abcAgent/1.0.0/data/abcAgent/1.0.0"
    And the package file "abcAgent-1.0.0.kdeps" will be created
    And an ai agent on "abc-agent" folder exists
    And the resources and data folder exists
    And theres a data file
    And it has a workflow file that has name property "xyzAgent" and version property "1.0.0" and default action "fooBar" and workspaces "@abcAgent"
    And it has a "resource1.pkl" file with no dependency with id property "fooBar"
    And it has a "resource2.pkl" file with id property "fooBar2" and dependent on "fooBar, @abcAgent/helloWorld:1.0.0"
    And it has a "resource3.pkl" file with id property "fooBar3" and dependent on "fooBar2, fooBar, @abcAgent/helloWorld3:1.0.0"
    And it has a "resource4.pkl" file with id property "fooBar4" and dependent on "nonexistent, fooBar3, @abcAgent/helloWorld2:1.0.0, fooBar2, fooBar, @abcAgent/helloWorld4:1.0.0"
    And the project is valid
    And the pkl files is valid
    And the project is compiled
    And it will be stored to "agents/xyzAgent/1.0.0/workflow.pkl"
    And the workflow action configuration will be rewritten to "@xyzAgent/fooBar:1.0.0"
    And the resource id for "xyzAgent_fooBar-1.0.0.pkl" will be rewritten to "@xyzAgent/fooBar:1.0.0"
    And the resource id for "xyzAgent_fooBar2-1.0.0.pkl" will be "@xyzAgent/fooBar2:1.0.0" and dependency "@abcAgent/helloWorld:1.0.0"
    And the resource id for "xyzAgent_fooBar2-1.0.0.pkl" will be "@xyzAgent/fooBar2:1.0.0" and dependency "@xyzAgent/fooBar:1.0.0"
    And the resource id for "xyzAgent_fooBar3-1.0.0.pkl" will be "@xyzAgent/fooBar3:1.0.0" and dependency "@abcAgent/helloWorld3:1.0.0"
    And the resource id for "xyzAgent_fooBar3-1.0.0.pkl" will be "@xyzAgent/fooBar3:1.0.0" and dependency "@abcAgent/helloWorld3:1.0.0"
    And the resource id for "xyzAgent_fooBar4-1.0.0.pkl" will be rewritten to "@xyzAgent/fooBar4:1.0.0"
    And the resource id for "xyzAgent_fooBar4-1.0.0.pkl" will be "@xyzAgent/fooBar4:1.0.0" and dependency "@abcAgent/helloWorld2:1.0.0"
    And the data files will be copied to "agents/xyzAgent/1.0.0/data/xyzAgent/1.0.0"
    And the package file "xyzAgent-1.0.0.kdeps" will be created
    And the resource file "abcAgent_helloWorld4-1.0.0.pkl" exists in the "xyzAgent" agent "1.0.0"
    And the data files will be copied to "agents/xyzAgent/1.0.0/data/abcAgent/1.0.0"

  Scenario: Project with external dependencies with version will copy the resources and data and be compiled to system agent folder (multiple dependencies provided)
    Given the ".kdeps" system folder exists
    And an ai agent on "agentx" folder exists
    And the resources and data folder exists
    And theres a data file
    And it has a workflow file that has name property "awesomeAgent" and version property "1.0.0" and default action "helloWorld"
    And it has a "resource1.pkl" file with no dependency with id property "helloWorld"
    And it has a "resource2.pkl" file with id property "helloWorld2" and dependent on "helloWorld"
    And it has a "resource3.pkl" file with id property "helloWorld3" and dependent on "helloWorld2, helloWorld"
    And it has a "resource4.pkl" file with id property "helloWorld4" and dependent on "nonexistent, helloWorld3, helloWorld2, helloWorld"
    Given the project is valid
    And the pkl files is valid
    When the project is compiled
    And it will be stored to "agents/awesomeAgent/1.0.0/workflow.pkl"
    And the workflow action configuration will be rewritten to "@awesomeAgent/helloWorld:1.0.0"
    And the resource id for "awesomeAgent_helloWorld-1.0.0.pkl" will be rewritten to "@awesomeAgent/helloWorld:1.0.0"
    And the resource id for "awesomeAgent_helloWorld2-1.0.0.pkl" will be "@awesomeAgent/helloWorld2:1.0.0" and dependency "@awesomeAgent/helloWorld:1.0.0"
    And the resource id for "awesomeAgent_helloWorld3-1.0.0.pkl" will be "@awesomeAgent/helloWorld3:1.0.0" and dependency "@awesomeAgent/helloWorld2:1.0.0"
    And the resource id for "awesomeAgent_helloWorld4-1.0.0.pkl" will be "@awesomeAgent/helloWorld4:1.0.0" and dependency "@awesomeAgent/helloWorld3:1.0.0"
    And the data files will be copied to "agents/awesomeAgent/1.0.0/data/awesomeAgent/1.0.0"
    And the package file "awesomeAgent-1.0.0.kdeps" will be created
    And an ai agent on "awesome-agent" folder exists
    And the resources and data folder exists
    And theres a data file
    And it has a workflow file that has name property "xAgent" and version property "1.0.0" and default action "fooBar" and workspaces "@awesomeAgent/helloWorld"
    And it has a "resource1.pkl" file with no dependency with id property "fooBar"
    And it has a "resource2.pkl" file with id property "fooBar2" and dependent on "fooBar, @awesomeAgent/helloWorld:1.0.0"
    And it has a "resource3.pkl" file with id property "fooBar3" and dependent on "fooBar2, fooBar, @awesomeAgent/helloWorld3:1.0.0"
    And it has a "resource4.pkl" file with id property "fooBar4" and dependent on "nonexistent, fooBar3, @awesomeAgent/helloWorld2:1.0.0, fooBar2, fooBar, @awesomeAgent/helloWorld4:1.0.0"
    And the project is valid
    And the pkl files is valid
    And the project is compiled
    And it will be stored to "agents/xAgent/1.0.0/workflow.pkl"
    And the workflow action configuration will be rewritten to "@xAgent/fooBar:1.0.0"
    And the resource id for "xAgent_fooBar-1.0.0.pkl" will be rewritten to "@xAgent/fooBar:1.0.0"
    And the resource id for "xAgent_fooBar2-1.0.0.pkl" will be "@xAgent/fooBar2:1.0.0" and dependency "@awesomeAgent/helloWorld:1.0.0"
    And the resource id for "xAgent_fooBar2-1.0.0.pkl" will be "@xAgent/fooBar2:1.0.0" and dependency "@xAgent/fooBar:1.0.0"
    And the resource id for "xAgent_fooBar3-1.0.0.pkl" will be "@xAgent/fooBar3:1.0.0" and dependency "@awesomeAgent/helloWorld3:1.0.0"
    And the resource id for "xAgent_fooBar3-1.0.0.pkl" will be "@xAgent/fooBar3:1.0.0" and dependency "@awesomeAgent/helloWorld3:1.0.0"
    And the resource id for "xAgent_fooBar4-1.0.0.pkl" will be rewritten to "@xAgent/fooBar4:1.0.0"
    And the resource id for "xAgent_fooBar4-1.0.0.pkl" will be "@xAgent/fooBar4:1.0.0" and dependency "@awesomeAgent/helloWorld2:1.0.0"
    And the data files will be copied to "agents/xAgent/1.0.0/data/xAgent/1.0.0"
    And the package file "xAgent-1.0.0.kdeps" will be created
    And the resource file "awesomeAgent_helloWorld4-1.0.0.pkl" exists in the "xAgent" agent "1.0.0"
    And the data files will be copied to "agents/xAgent/1.0.0/data/awesomeAgent/1.0.0"
    And an ai agent on "agentxyz" folder exists
    And the resources and data folder exists
    And theres a data file
    And it has a workflow file that has name property "abcAgent" and version property "1.0.0" and default action "helloWorld"
    And it has a "resource1.pkl" file with no dependency with id property "helloWorld"
    And it has a "resource2.pkl" file with id property "helloWorld2" and dependent on "helloWorld"
    And it has a "resource3.pkl" file with id property "helloWorld3" and dependent on "helloWorld2, helloWorld"
    And it has a "resource4.pkl" file with id property "helloWorld4" and dependent on "nonexistent, helloWorld3, helloWorld2, helloWorld"
    Given the project is valid
    And the pkl files is valid
    When the project is compiled
    And it will be stored to "agents/abcAgent/1.0.0/workflow.pkl"
    And the workflow action configuration will be rewritten to "@abcAgent/helloWorld:1.0.0"
    And the resource id for "abcAgent_helloWorld-1.0.0.pkl" will be rewritten to "@abcAgent/helloWorld:1.0.0"
    And the resource id for "abcAgent_helloWorld2-1.0.0.pkl" will be "@abcAgent/helloWorld2:1.0.0" and dependency "@abcAgent/helloWorld:1.0.0"
    And the resource id for "abcAgent_helloWorld3-1.0.0.pkl" will be "@abcAgent/helloWorld3:1.0.0" and dependency "@abcAgent/helloWorld2:1.0.0"
    And the resource id for "abcAgent_helloWorld4-1.0.0.pkl" will be "@abcAgent/helloWorld4:1.0.0" and dependency "@abcAgent/helloWorld3:1.0.0"
    And the data files will be copied to "agents/abcAgent/1.0.0/data/abcAgent/1.0.0"
    And the package file "abcAgent-1.0.0.kdeps" will be created
    And an ai agent on "abc-agent" folder exists
    And the resources and data folder exists
    And theres a data file
    And it has a workflow file that has name property "myAgent" and version property "1.0.0" and default action "helloWorld" and workspaces "@abcAgent"
    And it has a "resource1.pkl" file with no dependency with id property "helloWorld"
    And it has a "resource2.pkl" file with id property "helloWorld2" and dependent on "helloWorld, @abcAgent/helloWorld:1.0.0"
    And it has a "resource3.pkl" file with id property "helloWorld3" and dependent on "helloWorld2, helloWorld, @abcAgent/helloWorld3:1.0.0"
    And it has a "resource4.pkl" file with id property "helloWorld4" and dependent on "nonexistent, helloWorld3, @abcAgent/helloWorld2:1.0.0, helloWorld2, helloWorld, @abcAgent/helloWorld4:1.0.0"
    And the project is valid
    And the pkl files is valid
    And the project is compiled
    And it will be stored to "agents/myAgent/1.0.0/workflow.pkl"
    And the workflow action configuration will be rewritten to "@myAgent/helloWorld:1.0.0"
    And the resource id for "myAgent_helloWorld-1.0.0.pkl" will be rewritten to "@myAgent/helloWorld:1.0.0"
    And the resource id for "myAgent_helloWorld2-1.0.0.pkl" will be "@myAgent/helloWorld2:1.0.0" and dependency "@abcAgent/helloWorld:1.0.0"
    And the resource id for "myAgent_helloWorld2-1.0.0.pkl" will be "@myAgent/helloWorld2:1.0.0" and dependency "@myAgent/helloWorld:1.0.0"
    And the resource id for "myAgent_helloWorld3-1.0.0.pkl" will be "@myAgent/helloWorld3:1.0.0" and dependency "@abcAgent/helloWorld3:1.0.0"
    And the resource id for "myAgent_helloWorld3-1.0.0.pkl" will be "@myAgent/helloWorld3:1.0.0" and dependency "@abcAgent/helloWorld3:1.0.0"
    And the resource id for "myAgent_helloWorld4-1.0.0.pkl" will be rewritten to "@myAgent/helloWorld4:1.0.0"
    And the resource id for "myAgent_helloWorld4-1.0.0.pkl" will be "@myAgent/helloWorld4:1.0.0" and dependency "@abcAgent/helloWorld2:1.0.0"
    And the data files will be copied to "agents/myAgent/1.0.0/data/myAgent/1.0.0"
    And the package file "myAgent-1.0.0.kdeps" will be created
    And the resource file "abcAgent_helloWorld4-1.0.0.pkl" exists in the "myAgent" agent "1.0.0"
    And the data files will be copied to "agents/myAgent/1.0.0/data/abcAgent/1.0.0"
    And an ai agent on "ai-agent" folder exists
    And the resources and data folder exists
    And theres a data file
    And it has a workflow file that has name property "newAgent" and version property "1.0.0" and default action "helloWorld"
    And it has a "resource1.pkl" file with no dependency with id property "helloWorld"
    And it has a "resource2.pkl" file with id property "helloWorld2" and dependent on "helloWorld"
    And it has a "resource3.pkl" file with id property "helloWorld3" and dependent on "helloWorld2, helloWorld"
    And it has a "resource4.pkl" file with id property "helloWorld4" and dependent on "nonexistent, helloWorld3, helloWorld2, helloWorld"
    Given the project is valid
    And the pkl files is valid
    When the project is compiled
    And it will be stored to "agents/newAgent/1.0.0/workflow.pkl"
    And the workflow action configuration will be rewritten to "@newAgent/helloWorld:1.0.0"
    And the resource id for "newAgent_helloWorld-1.0.0.pkl" will be rewritten to "@newAgent/helloWorld:1.0.0"
    And the resource id for "newAgent_helloWorld2-1.0.0.pkl" will be "@newAgent/helloWorld2:1.0.0" and dependency "@newAgent/helloWorld:1.0.0"
    And the resource id for "newAgent_helloWorld3-1.0.0.pkl" will be "@newAgent/helloWorld3:1.0.0" and dependency "@newAgent/helloWorld2:1.0.0"
    And the resource id for "newAgent_helloWorld4-1.0.0.pkl" will be "@newAgent/helloWorld4:1.0.0" and dependency "@newAgent/helloWorld3:1.0.0"
    And the data files will be copied to "agents/newAgent/1.0.0/data/newAgent/1.0.0"
    And the package file "newAgent-1.0.0.kdeps" will be created
    And an ai agent on "awesome-agent" folder exists
    And the resources and data folder exists
    And theres a data file
    And it has a workflow file that has name property "acmeCoAgent" and version property "1.0.0" and default action "fooBar" and workspaces "@newAgent:1.0.0"
    And it has a "resource1.pkl" file with no dependency with id property "fooBar"
    And it has a "resource2.pkl" file with id property "fooBar2" and dependent on "fooBar, @newAgent/helloWorld:1.0.0"
    And it has a "resource3.pkl" file with id property "fooBar3" and dependent on "fooBar2, fooBar, @newAgent/helloWorld3:1.0.0"
    And it has a "resource4.pkl" file with id property "fooBar4" and dependent on "nonexistent, fooBar3, @newAgent/helloWorld2:1.0.0, fooBar2, fooBar, @newAgent/helloWorld4:1.0.0"
    And the project is valid
    And the pkl files is valid
    And the project is compiled
    And it will be stored to "agents/acmeCoAgent/1.0.0/workflow.pkl"
    And the workflow action configuration will be rewritten to "@acmeCoAgent/fooBar:1.0.0"
    And the resource id for "acmeCoAgent_fooBar-1.0.0.pkl" will be rewritten to "@acmeCoAgent/fooBar:1.0.0"
    And the resource id for "acmeCoAgent_fooBar2-1.0.0.pkl" will be "@acmeCoAgent/fooBar2:1.0.0" and dependency "@newAgent/helloWorld:1.0.0"
    And the resource id for "acmeCoAgent_fooBar2-1.0.0.pkl" will be "@acmeCoAgent/fooBar2:1.0.0" and dependency "@acmeCoAgent/fooBar:1.0.0"
    And the resource id for "acmeCoAgent_fooBar3-1.0.0.pkl" will be "@acmeCoAgent/fooBar3:1.0.0" and dependency "@newAgent/helloWorld3:1.0.0"
    And the resource id for "acmeCoAgent_fooBar3-1.0.0.pkl" will be "@acmeCoAgent/fooBar3:1.0.0" and dependency "@newAgent/helloWorld3:1.0.0"
    And the resource id for "acmeCoAgent_fooBar4-1.0.0.pkl" will be rewritten to "@acmeCoAgent/fooBar4:1.0.0"
    And the resource id for "acmeCoAgent_fooBar4-1.0.0.pkl" will be "@acmeCoAgent/fooBar4:1.0.0" and dependency "@newAgent/helloWorld2:1.0.0"
    And the data files will be copied to "agents/acmeCoAgent/1.0.0/data/acmeCoAgent/1.0.0"
    And the package file "acmeCoAgent-1.0.0.kdeps" will be created
    And the resource file "newAgent_helloWorld4-1.0.0.pkl" exists in the "acmeCoAgent" agent "1.0.0"
    And the data files will be copied to "agents/acmeCoAgent/1.0.0/data/newAgent/1.0.0"
    And an ai agent on "mega-agent" folder exists
    And the resources and data folder exists
    And theres a data file
    And it has a workflow file that has name property "abcAgent" and version property "1.0.0" and default action "helloWorld"
    And it has a "resource1.pkl" file with no dependency with id property "helloWorld"
    And it has a "resource2.pkl" file with id property "helloWorld2" and dependent on "helloWorld"
    And it has a "resource3.pkl" file with id property "helloWorld3" and dependent on "helloWorld2, helloWorld"
    And it has a "resource4.pkl" file with id property "helloWorld4" and dependent on "nonexistent, helloWorld3, helloWorld2, helloWorld"
    Given the project is valid
    And the pkl files is valid
    When the project is compiled
    And it will be stored to "agents/abcAgent/1.0.0/workflow.pkl"
    And the workflow action configuration will be rewritten to "@abcAgent/helloWorld:1.0.0"
    And the resource id for "abcAgent_helloWorld-1.0.0.pkl" will be rewritten to "@abcAgent/helloWorld:1.0.0"
    And the resource id for "abcAgent_helloWorld2-1.0.0.pkl" will be "@abcAgent/helloWorld2:1.0.0" and dependency "@abcAgent/helloWorld:1.0.0"
    And the resource id for "abcAgent_helloWorld3-1.0.0.pkl" will be "@abcAgent/helloWorld3:1.0.0" and dependency "@abcAgent/helloWorld2:1.0.0"
    And the resource id for "abcAgent_helloWorld4-1.0.0.pkl" will be "@abcAgent/helloWorld4:1.0.0" and dependency "@abcAgent/helloWorld3:1.0.0"
    And the data files will be copied to "agents/abcAgent/1.0.0/data/abcAgent/1.0.0"
    And the package file "abcAgent-1.0.0.kdeps" will be created
    And an ai agent on "abc-agent" folder exists
    And the resources and data folder exists
    And theres a data file
    And it has a workflow file that has name property "xyzAgent" and version property "1.0.0" and default action "fooBar" and workspaces "@acmeCoAgent,@xAgent,@abcAgent,@myAgent/helloWorld4:1.0.0,@newAgent:1.0.0,@awesomeAgent/helloWorld"
    And it has a "resource1.pkl" file with no dependency with id property "fooBar"
    And it has a "resource2.pkl" file with id property "fooBar2" and dependent on "fooBar, @abcAgent/helloWorld:1.0.0"
    And it has a "resource3.pkl" file with id property "fooBar3" and dependent on "fooBar2, fooBar, @abcAgent/helloWorld3:1.0.0"
    And it has a "resource4.pkl" file with id property "fooBar4" and dependent on "nonexistent, fooBar3, @abcAgent/helloWorld2:1.0.0, fooBar2, fooBar, @abcAgent/helloWorld4:1.0.0"
    And the project is valid
    And the pkl files is valid
    And the project is compiled
    And it will be stored to "agents/xyzAgent/1.0.0/workflow.pkl"
    And the workflow action configuration will be rewritten to "@xyzAgent/fooBar:1.0.0"
    And the resource id for "xyzAgent_fooBar-1.0.0.pkl" will be rewritten to "@xyzAgent/fooBar:1.0.0"
    And the resource id for "xyzAgent_fooBar2-1.0.0.pkl" will be "@xyzAgent/fooBar2:1.0.0" and dependency "@abcAgent/helloWorld:1.0.0"
    And the resource id for "xyzAgent_fooBar2-1.0.0.pkl" will be "@xyzAgent/fooBar2:1.0.0" and dependency "@xyzAgent/fooBar:1.0.0"
    And the resource id for "xyzAgent_fooBar3-1.0.0.pkl" will be "@xyzAgent/fooBar3:1.0.0" and dependency "@abcAgent/helloWorld3:1.0.0"
    And the resource id for "xyzAgent_fooBar3-1.0.0.pkl" will be "@xyzAgent/fooBar3:1.0.0" and dependency "@abcAgent/helloWorld3:1.0.0"
    And the resource id for "xyzAgent_fooBar4-1.0.0.pkl" will be rewritten to "@xyzAgent/fooBar4:1.0.0"
    And the resource id for "xyzAgent_fooBar4-1.0.0.pkl" will be "@xyzAgent/fooBar4:1.0.0" and dependency "@abcAgent/helloWorld2:1.0.0"
    And the data files will be copied to "agents/xyzAgent/1.0.0/data/xyzAgent/1.0.0"
    And the package file "xyzAgent-1.0.0.kdeps" will be created
    And the resource file "abcAgent_helloWorld-1.0.0.pkl" exists in the "xyzAgent" agent "1.0.0"
    And the data files will be copied to "agents/xyzAgent/1.0.0/data/abcAgent/1.0.0"
    And the resource file "myAgent_helloWorld2-1.0.0.pkl" exists in the "xyzAgent" agent "1.0.0"
    And the data files will be copied to "agents/xyzAgent/1.0.0/data/myAgent/1.0.0"
    And the resource file "awesomeAgent_helloWorld3-1.0.0.pkl" exists in the "xyzAgent" agent "1.0.0"
    And the data files will be copied to "agents/xyzAgent/1.0.0/data/awesomeAgent/1.0.0"
    And the resource file "newAgent_helloWorld4-1.0.0.pkl" exists in the "xyzAgent" agent "1.0.0"
    And the data files will be copied to "agents/xyzAgent/1.0.0/data/newAgent/1.0.0"
    And the resource file "acmeCoAgent_fooBar3-1.0.0.pkl" exists in the "xyzAgent" agent "1.0.0"
    And the data files will be copied to "agents/xyzAgent/1.0.0/data/acmeCoAgent/1.0.0"
    And the resource file "xAgent_fooBar2-1.0.0.pkl" exists in the "xyzAgent" agent "1.0.0"
    And the data files will be copied to "agents/xyzAgent/1.0.0/data/xAgent/1.0.0"

  Scenario: Kdeps packages will be extracted
    When a kdeps archive "xyzAgent-1.0.0.kdeps" is opened
    Then the content of that archive file will be extracted to "agents/xyzAgent/1.0.0"

  Scenario: Valid ai agent will be packaged
    When the project is valid
    When the pkl files is valid
    Then the project will be archived to "myAgent-1.0.0.kdeps"

  Scenario: Project with resource run blocks has two or more chat, httpClient and exec declarations but null will be valid
    Given the ".kdeps" system folder exists
    And an ai agent on "ai-agent-chatter1" folder exists
    And the resources and data folder exists
    And theres a data file
    And it has a workflow file that has name property "newAgentChatter1" and version property "1.0.0" and default action "helloWorld"
    And it has a "resource1.pkl" file with no dependency with id property "helloWorld"
    And it has a "resource2.pkl" file with id property "helloWorld2" and dependent on "helloWorld"
    And it has a "resource3.pkl" file with id property "helloWorld3" and dependent on "helloWorld2, helloWorld" with run block "chat, exec" and is null
    And it has a "resource9.pkl" file with id property "helloWorld9" and dependent on "helloWorld2, helloWorld" with run block "chat" and is not null
    And the project is valid
    And the pkl files is valid
    Then the project will be archived to "newAgentChatter1-1.0.0.kdeps"

  Scenario: Project with resource run blocks has two or more chat, httpClient and exec declarations will be invalid
    Given the ".kdeps" system folder exists
    And an ai agent on "ai-agent-chatter2" folder exists
    And the resources and data folder exists
    And theres a data file
    And it has a workflow file that has name property "newAgentChatter2" and version property "1.0.0" and default action "helloWorld"
    And it has a "resource1.pkl" file with no dependency with id property "helloWorld"
    And it has a "resource2.pkl" file with id property "helloWorld2" and dependent on "helloWorld"
    And it has a "resource3.pkl" file with id property "helloWorld3" and dependent on "helloWorld2, helloWorld" with run block "chat, exec" and is null
    And it has a "resource4.pkl" file with id property "helloWorld4" and dependent on "helloWorld2, helloWorld" with run block "chat" and is not null
    And it has a "resource9.pkl" file with id property "helloWorld9" and dependent on "helloWorld2, helloWorld" with run block "chat, exec" and is not null
    And the project is invalid
    Then the project will not be archived to "newAgentChatter2-1.0.0.kdeps"

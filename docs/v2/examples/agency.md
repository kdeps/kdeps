# Compose two agents into an agency

*Applies to workflow mode.*

## Overview

In this tutorial you build an agency: two agents in one project, where the
entry-point agent calls the other and returns the combined result. Each agent
is its own `workflow.yaml`; an `agency.yaml` manifest ties them together.

This tutorial is for developers who have completed the
[quickstart](/getting-started/quickstart) and read
[AI agencies](/concepts/agency). It assumes you know:

- Basic YAML
- How a single kdeps workflow runs

By the end you will be able to:

- Write an `agency.yaml` manifest with a `targetAgentId`
- Have one agent call another with the `agent:` resource
- Forward data to a sub-agent with `params:` and read it with `get()`

## Background

An agency is like calling functions across modules. Each agent is a full
workflow - its own resources, routes, and settings. The `agent:` resource runs
another agent's entire pipeline and returns its `apiResponse`. Only the
entry-point agent (`targetAgentId`) serves HTTP; the others are called
internally.

## Before you start

- kdeps installed (`kdeps --version`).
- A working directory for the project.

## Step 1: create the structure

```bash
mkdir -p greeter-agency/agents/greeter
mkdir -p greeter-agency/agents/responder
cd greeter-agency
```

## Step 2: write the manifest

Create `agency.yaml`:

```yaml
# agency.yaml
apiVersion: kdeps.io/v1
kind: Agency

metadata:
  name: greeter-agency
  version: "1.0.0"
  targetAgentId: greeter-agent   # the agent that serves HTTP

agents:
  - agents/greeter               # relative to this file
  - agents/responder
```

## Step 3: the helper agent

Create `agents/responder/workflow.yaml`:

<div v-pre>

```yaml
# agents/responder/workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: responder-agent          # how other agents address it
  version: "1.0.0"
  targetActionId: respond

settings:
  agentSettings:
    timezone: UTC

resources:
  - actionId: respond
    name: Respond
    apiResponse:
      success: true
      response: "Hello, {{ get('name', 'World') }}! (from responder-agent)"
```

</div>

This agent has no `apiServer:` - it only runs when another agent calls it.
`get('name')` reads the value the caller forwards.

## Step 4: the entry-point agent

Create `agents/greeter/workflow.yaml`:

<div v-pre>

```yaml
# agents/greeter/workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: greeter-agent
  version: "1.0.0"
  targetActionId: greet

settings:
  apiServer:
    portNum: 17100
    routes:
      - path: /api/v1/greet
        methods: [GET]

resources:
  - actionId: callResponder
    name: Call responder
    agent:
      name: responder-agent      # match responder's metadata.name
      params:
        name: "{{ get('name') }}" # forwarded; readable as get('name') there

  - actionId: greet
    name: Greet
    requires: [callResponder]
    apiResponse:
      success: true
      response: "{{ output('callResponder') }}"   # the responder's reply
```

</div>

## Step 5: validate and run

```bash
kdeps validate .
export KDEPS_API_AUTH_TOKEN=dev-token
kdeps run agency.yaml
```

```bash
curl "http://localhost:17100/api/v1/greet?name=Alice" \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN"
```

Response:

```json
{
  "success": true,
  "data": "Hello, Alice! (from responder-agent)"
}
```

## Step 6: package the agency (optional)

```bash
kdeps bundle package .            # -> greeter-agency-1.0.0.kagency
kdeps run greeter-agency-1.0.0.kagency
kdeps bundle build greeter-agency-1.0.0.kagency   # Docker image
```

## Summary

You built an agency that:

- Declares two agents and an entry point in `agency.yaml`
- Calls one agent from another with the `agent:` resource
- Forwards a value with `params:` and reads it with `get()`
- Returns the sub-agent's output with `output()`

## Next steps

- [AI agencies](/concepts/agency) - discovery, packaging, `.kagency`
- [Agent resource](/resources/delegation/agent) - the `agent:` reference
- [Packaging commands](/reference/cli/packaging) - `.kdeps` and `.kagency`
- [Docker deployment](/deployment/docker) - ship the agency as an image

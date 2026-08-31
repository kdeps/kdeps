# Build a reusable component

*Applies to workflow mode.*

## Overview

In this tutorial you build a custom component - a bundle of resources with a
typed input interface - and call it from a workflow. Components are how you
package reusable logic and share it across projects.

This tutorial is for developers who have completed the
[quickstart](/getting-started/quickstart) and read
[Components](/concepts/components). It assumes you know:

- Basic YAML

By the end you will be able to:

- Write a `component.yaml` with an `interface.inputs` schema
- Auto-discover a component from a `components/` directory
- Call it with `component:` and pass typed inputs with `with:`
- Read a component's output with `output()`

## Background

A component is a `component.yaml` plus its resources. Drop it in a
`components/<name>/` directory and kdeps loads it automatically - no change to
`workflow.yaml`. A resource invokes it with `component:`, passing inputs under
`with:`; those are validated against `interface.inputs`.

## Before you start

- kdeps installed (`kdeps --version`).
- A working directory for the project.

## Step 1: create the structure

```bash
mkdir -p greeter/components/formatter
mkdir greeter/resources
cd greeter
```

## Step 2: write the component

Create `components/formatter/component.yaml`:

<div v-pre>

```yaml
# components/formatter/component.yaml
apiVersion: kdeps.io/v1
kind: Component

metadata:
  name: formatter
  version: "1.0.0"

interface:
  inputs:
    - name: name
      type: string
      required: true
      description: The name to greet
    - name: shout
      type: string
      required: false
      description: "'true' to upper-case the greeting"

resources:
  - actionId: formatGreeting
    name: Format greeting
    after:
      - "set('greeting', 'Hello, ' + get('name') + '!')"
      - "set('final', get('shout') == 'true' ? upper(get('greeting')) : get('greeting'))"
    apiResponse:
      success: true
      response:
        text: "{{ get('final') }}"
```

</div>

Inside the component, `get('name')` and `get('shout')` are the inputs passed by
the caller.

## Step 3: call the component

Create `resources/greet.yaml`:

<div v-pre>

```yaml
# resources/greet.yaml
actionId: greet
name: Greet
component:
  name: formatter          # the component's metadata.name
  with:
    name: "{{ get('who', 'World') }}"
    shout: "{{ get('loud', 'false') }}"
```

</div>

## Step 4: return the result

Create `resources/response.yaml`:

<div v-pre>

```yaml
# resources/response.yaml
actionId: response
name: Response
requires: [greet]
apiResponse:
  success: true
  response:
    greeting: "{{ output('greet').text }}"
```

</div>

## Step 5: add the route and target

Create `workflow.yaml`:

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: greeter
  version: "1.0.0"
  targetActionId: response

settings:
  apiServer:
    portNum: 16395
    routes:
      - path: /greet
        methods: [GET]
```

## Step 6: validate and run

```bash
kdeps validate .
export KDEPS_API_AUTH_TOKEN=dev-token
kdeps run .
```

```bash
curl "http://localhost:16395/greet?who=Ada&loud=true" \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN"
```

Response:

```json
{ "success": true, "data": { "greeting": "HELLO, ADA!" } }
```

## Summary

You built and used a component that:

- Declares typed inputs in `interface.inputs`
- Is auto-discovered from `components/formatter/`
- Is called with `component:` + `with:`
- Returns a value read with `output('greet').text`

## Next steps

- [Components](/concepts/components) - registry components, `componentTools:`
- [Components reference](/reference/components) - full schema, env var derivation, packaging
- [Registry commands](/reference/cli/registry) - install and publish components
- [Two-agent agency tutorial](/examples/agency) - composing whole agents instead

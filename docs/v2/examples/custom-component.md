# Build a reusable component

Build a custom component - a bundle of resources with a
typed input interface - and call it from a workflow. Components are how you
package reusable logic and share it across projects.

A component is a `component.yaml` plus its resources. Drop it in a
`components/<name>/` directory and kdeps loads it automatically - no change to
`workflow.yaml`. A resource invokes it with `component:`, passing inputs under
`with:`; those are validated against `interface.inputs`.

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

## Next steps

- [Components](/agencies/components) - registry components, `componentTools:`
- [Components reference](/agencies/component-reference) - full schema, env var derivation, packaging
- [Registry commands](/registry/cli) - install and publish components
- [Two-agent agency tutorial](/examples/agency) - composing whole agents instead

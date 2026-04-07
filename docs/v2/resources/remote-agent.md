# Remote Agent Resource (UAF)

> **Note**: This capability is now provided as an installable component. See the [Components guide](../concepts/components) for how to install and use it.
>
> Install: `kdeps component install remoteagent`
>
> Usage: `run: { component: { name: remoteagent, with: { url: "...", query: "..." } } }`

The Remote Agent component invokes a remote kdeps agent over HTTP, sending a query and returning the agent's response.

> **Note**: The component uses simple HTTP invocation (`url` + `query`). For the full UAF federation protocol (Ed25519 signing, URN-based identity, receipts, trust levels, fallback agents), use the `kdeps federation` CLI commands and configure the federation workflow directly.

## Component Inputs

| Input | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `url` | string | yes | — | Base URL of the remote kdeps agent (e.g. `http://host:16394`) |
| `query` | string | yes | — | Query or request body to send to the agent |

## Using the Remote Agent Component

```yaml
run:
  component:
    name: remoteagent
    with:
      url: "https://remote-agent.example.com"
      query: "Translate 'Hello, world!' to French"
```

Access the result via `output('<callerActionId>')`. The result includes the agent's response.

---

## Result Map

| Key | Type | Description |
|-----|------|-------------|
| `success` | bool | `true` if the remote agent responded successfully. |
| `response` | any | The agent's response payload. |

---

## Expression Support

All fields support [KDeps expressions](../concepts/expressions):

<div v-pre>

```yaml
run:
  component:
    name: remoteagent
    with:
      url: "{{ env('REMOTE_AGENT_URL') }}"
      query: "{{ get('user_query') }}"
```

</div>

---

## Full Example: Multi-Agent Pipeline

<div v-pre>

```yaml
# Step 1: Call a remote translation agent
- apiVersion: kdeps.io/v1
  kind: Resource

  metadata:
    actionId: translate
    name: Translate via Remote Agent

  run:
    component:
      name: remoteagent
      with:
        url: "{{ env('TRANSLATOR_AGENT_URL') }}"
        query: "{{ get('text_to_translate') }}"

# Step 2: Use the translation result
- apiVersion: kdeps.io/v1
  kind: Resource

  metadata:
    actionId: respond
    name: Return Translation
    requires:
      - translate

  run:
    apiResponse:
      success: true
      response:
        translation: "{{ output('translate').response }}"
```

</div>

---

## Federation CLI Reference

For advanced UAF federation (key management, agent registration, receipts):

| Command | Description |
|---------|-------------|
| `kdeps federation keygen --org <name>` | Generate Ed25519 keypair |
| `kdeps federation register ...` | Register agent in UAF registry |
| `kdeps federation trust add ...` | Add registry trust anchor |
| `kdeps federation mesh list` | List remote agents in project |
| `kdeps federation receipt verify ...` | Verify signed receipt |

## See Also

- [`agent:`](./overview.md#agent) - In-process delegation within an agency
- [Federation Demo](https://github.com/kdeps/kdeps/tree/main/examples/federation-demo) - Working end-to-end example

# Remote Agent Resource (UAF)

The `remoteAgent` resource invokes a remote federated agent using the **Universal Agent Federation (UAF)** protocol. Requests are cryptographically signed with Ed25519, and responses include a signed receipt that verifies the callee's identity and output integrity.

## Basic Usage

<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: callExternalAgent
  description: Call a remote translation agent

run:
  remoteAgent:
    urn: "urn:agent:api.partner.io/io.partner:translator@v2.1.0#sha256:a1b2c3d4e5f6..."
    input:
      text: "{{ get('q') }}"
      targetLanguage: "es"
    timeout: "30s"

  apiResponse:
    success: true
    response:
      translation: "{{ get('callExternalAgent').translated }}"
```

</div>

## URN Format

Every remote agent is identified by a URN that cryptographically binds its identity to its spec content:

```
urn:agent:<authority>/<namespace>:<name>@<version>#<algorithm>:<content-hash>
```

| Component | Example | Description |
|-----------|---------|-------------|
| `authority` | `api.partner.io` | Hostname (+ optional port) of the registry or agent |
| `namespace` | `io.partner` | Reverse-DNS namespace of the agent's owner |
| `name` | `translator` | Agent name (alphanumeric + hyphens) |
| `version` | `v2.1.0` | Semantic version |
| `algorithm` | `sha256` | Hash algorithm: `sha256`, `sha512` |
| `content-hash` | `a1b2c3...` | Hex-encoded hash of the canonicalized workflow YAML |

Compute the hash of any spec:

```bash
# kdeps canonicalizes YAML → JSON → SHA256 before comparing
kdeps federation register --urn ... --spec workflow.yaml ...
```

## Configuration Reference

```yaml
run:
  remoteAgent:
    # Required: URN of the remote agent
    urn: "urn:agent:authority/namespace:name@version#sha256:hash"

    # Required: Input parameters to pass to the remote agent
    # Values support kdeps expressions
    input:
      key: "{{ get('someValue') }}"
      literal: "static-value"

    # Optional: Invocation timeout (default: 60s)
    timeout: "30s"

    # Optional: Minimum trust level required
    # Values: self-attested | verified | certified
    requireTrustLevel: "verified"

    # Optional: Cache the remote spec locally (default: true)
    cacheSpec: true

    # Optional: Fallback agents tried in order if primary fails
    fallback:
      - urn: "urn:agent:backup.io/ns:agent@v1.0.0#sha256:..."
        timeout: "15s"
      - urn: "urn:agent:fallback.io/ns:agent@v1.0.0#sha256:..."
```

## Trust Levels

| Level | Description |
|-------|-------------|
| `self-attested` | The agent claims its own identity. No third-party verification. |
| `verified` | A UAF registry has verified the agent's spec hash. |
| `certified` | A third-party auditor has certified the agent. |

When `requireTrustLevel` is set, kdeps rejects agents that don't meet the minimum level.

## Accessing the Response

The response from the remote agent is available via `get('actionId')`. The remote agent's outputs are mapped directly:

<div v-pre>

```yaml
run:
  remoteAgent:
    urn: "urn:agent:example.com/io.example:classifier@v1.0.0#sha256:..."
    input:
      text: "{{ get('q') }}"

  apiResponse:
    success: true
    response:
      category: "{{ get('myRemoteCall').category }}"
      confidence: "{{ get('myRemoteCall').confidence }}"
```

</div>

## Fallback Agents

If the primary agent fails (network error, timeout, HTTP error), kdeps automatically tries the fallback list in order:

```yaml
run:
  remoteAgent:
    urn: "urn:agent:primary.io/ns:agent@v1.0.0#sha256:abc..."
    input:
      data: "{{ get('input') }}"
    timeout: "20s"
    fallback:
      - urn: "urn:agent:secondary.io/ns:agent@v1.0.0#sha256:def..."
        timeout: "15s"
      - urn: "urn:agent:tertiary.io/ns:agent@v1.0.0#sha256:ghi..."
        timeout: "10s"
```

## Getting Started with Federation

### 1. Generate Keys

```bash
kdeps federation keygen --org my-org
# → ~/.config/kdeps/keys/my-org.key
# → ~/.config/kdeps/keys/my-org.key.pub
```

### 2. Register Your Agent

```bash
kdeps federation register \
  --urn "urn:agent:registry.kdeps.io/io.myorg:my-agent@v1.0.0#sha256:<hash>" \
  --spec workflow.yaml \
  --registry https://registry.kdeps.io \
  --contact admin@myorg.io
```

### 3. Add Trust Anchors

```bash
kdeps federation trust add \
  --registry https://registry.kdeps.io \
  --public-key registry-pub.pem
```

### 4. Inspect Your Mesh

```bash
# List all remote agents used in current project
kdeps federation mesh list

# Preview registration manifest (dry-run)
kdeps federation mesh publish
```

### 5. Verify Receipts

```bash
kdeps federation receipt verify \
  --receipt receipt.json \
  --callee-urn "urn:agent:..." \
  --caller-urn "urn:agent:..." \
  --public-key callee.pub
```

## Security Model

UAF uses **Ed25519** for all cryptographic operations:

1. **Caller signs** the `InvocationRequest` body with its private key
2. **Callee verifies** the request (optional, configurable)
3. **Callee signs** the `Receipt` JSON with its private key
4. **Caller verifies** the receipt signature using the callee's public key (fetched from registry)
5. **Message ID** in the receipt must match the original request

The receipt includes:
- `messageId` — UUID from the original request
- `callee` / `caller` — URNs of both parties
- `execution.status` — `success`, `error`, or `timeout`
- `execution.outputs` — the agent's outputs
- `timestamp` — ISO 8601 execution time

## Wire Format

```http
POST /.uaf/v1/invoke HTTP/2
Content-Type: application/json
X-UAF-Version: 1.0
X-UAF-Message-Id: <uuid>
X-UAF-Caller-Urn: urn:agent:...
X-UAF-Caller-Public-Key: ed25519:<base64>
X-UAF-Signature: <hex-signature>

{ "messageId": "...", "timestamp": "...", "caller": {...}, "callee": {...}, "payload": {...} }

HTTP/2 200 OK
X-UAF-Receipt: <base64(receipt-json)>
X-UAF-Receipt-Signature: <hex-ed25519-sig>
```

## CLI Reference

| Command | Description |
|---------|-------------|
| `kdeps federation keygen --org <name>` | Generate Ed25519 keypair |
| `kdeps federation register ...` | Register agent in registry |
| `kdeps federation key-rotate --org <name>` | Rotate keys (dual-key period) |
| `kdeps federation trust add ...` | Add registry trust anchor |
| `kdeps federation trust list` | List trust anchors |
| `kdeps federation trust remove --host <h>` | Remove trust anchor |
| `kdeps federation mesh list` | List remote agents in project |
| `kdeps federation mesh publish` | Preview registration manifest |
| `kdeps federation receipt verify ...` | Verify signed receipt |

## See Also

- [`agent:`](./overview.md#agent) — In-process delegation within an agency
- [Federation Demo](../../examples/federation-demo/) — Working end-to-end example
- [Getting Started](../getting-started/quickstart.md) — kdeps quickstart

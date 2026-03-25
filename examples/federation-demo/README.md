# Federation Demo: Agent A → Agent B

This example demonstrates Universal Agent Federation (UAF): **agent-a** calls **agent-b** over HTTP with cryptographic request signing and receipt verification.

## Architecture

```
HTTP Client
    │  POST /api/v1/call  { "message": "Hello!" }
    ▼
agent-a (port 16395)
    │  UAF invocation: signed InvocationRequest
    │  POST /.uaf/v1/invoke
    ▼
agent-b (port 16396)
    │  Returns: signed Receipt + outputs
    ▼
agent-a verifies receipt signature, returns result
    │
    ▼
HTTP Client receives echoed response
```

## Prerequisites

- `kdeps` CLI installed (`curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/main/install.sh | sh`)
- Two terminal windows

## Step 1: Generate Federation Keys

```bash
# Generate an Ed25519 keypair for the demo
kdeps federation keygen --org demo
# Creates: ~/.config/kdeps/keys/demo.key and demo.key.pub
```

## Step 2: Start Agent B (Echo Server)

```bash
cd examples/federation-demo/agent-b
kdeps run
# Starts on port 16396
```

## Step 3: Register Agent B

In a new terminal, compute the URN hash and register agent-b with a local registry (or skip for local testing):

```bash
# Compute hash of agent-b spec
HASH=$(kdeps federation mesh publish 2>/dev/null | grep sha256 | awk '{print $NF}')

# Register (requires a UAF registry endpoint)
kdeps federation register \
  --urn "urn:agent:localhost:16396/demo:federation-agent-b@1.0.0#sha256:${HASH}" \
  --spec examples/federation-demo/agent-b/workflow.yaml \
  --registry http://localhost:16396 \
  --contact demo@example.com
```

## Step 4: Start Agent A (Caller)

```bash
cd examples/federation-demo/agent-a
kdeps run
# Starts on port 16395
```

## Step 5: Test the Federation

```bash
curl -X POST http://localhost:16395/api/v1/call \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello from federation!"}'
```

Expected response:

```json
{
  "success": true,
  "data": {
    "echoed": "Hello from federation!",
    "timestamp": "2026-03-25T04:00:00Z",
    "agentId": "federation-agent-b"
  }
}
```

## URN Format

The URN used in agent-a's `call_remote.yaml` follows this format:

```
urn:agent:<authority>/<namespace>:<name>@<version>#<algorithm>:<content-hash>
         └──────────┘ └─────────┘ └────┘  └───────┘  └───────┘  └───────────────────────────────────────────────────────────────┘
         localhost:16396  demo   agent-b   1.0.0       sha256    SHA256(canonicalized workflow.yaml)
```

Compute the hash of any workflow:

```bash
# Using kdeps canonical hash tool
kdeps federation keygen --org demo  # ensure keys exist
# Then run: sha256 of JSON-canonicalized YAML
```

## Security Model

1. **agent-a** generates an `InvocationRequest` with a UUID message ID, timestamp, and caller identity
2. **agent-a** signs the request body with its Ed25519 private key (`~/.config/kdeps/keys/installation.key`)
3. **agent-b** processes the request and returns a signed `Receipt`
4. **agent-a** verifies the receipt signature using **agent-b**'s public key (fetched from registry)
5. **agent-a** validates the message ID matches the original request

## Trust Levels

| Level | Meaning |
|-------|---------|
| `self-attested` | Agent claims its own identity (no third-party verification) |
| `verified` | Registry verified the agent's spec hash |
| `certified` | Third-party auditor certified the agent |

## Key Concepts

- **URN**: Unique, content-addressed identifier binding agent identity to spec hash
- **Receipt**: Signed proof of execution from callee
- **Fallback**: If primary agent fails, try alternate URN(s)
- **Registry**: Resolves URNs to HTTP endpoints and public keys

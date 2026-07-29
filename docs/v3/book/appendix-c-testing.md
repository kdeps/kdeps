# Appendix C: Testing Your Agent

Testing a kdeps agent is testing an HTTP API plus an LLM. The HTTP API part is straightforward and deterministic. The LLM part is not. This appendix covers how to test both, what can be automated, and what cannot.

---

## What You Can Test Deterministically

These behaviors do not depend on LLM output and can be fully automated:

- Input validation rejects bad input with the correct status code and message
- Validation passes for correctly formed requests
- DAG ordering is correct (no cycles, target is reachable)
- SQL queries run and return the expected shape
- HTTP client resources reach their endpoints and parse responses
- `apiResponse:` returns the expected JSON shape
- `onError:` produces fallback output when an upstream resource fails
- Session values persist across requests in the same session

## What Requires Human Judgment or Statistical Evaluation

- Whether LLM output is correct, coherent, or useful
- Whether prompt engineering changes improved or degraded response quality
- Whether the agent reaches the right conclusion in a complex multi-step task

Automate the structural tests. Evaluate LLM quality manually during development.

---

## Smoke Testing With curl

The fastest test is a curl one-liner run after every change:

```bash
# Happy path
$ curl -s -X POST http://localhost:16395/api/v1/chat \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"q": "What is 2 + 2?"}' | jq .

# Expected shape
{
  "success": true,
  "data": {
    "answer": "..."
  }
}
```

```bash
# Validation: empty input should return 400
$ curl -s -o /dev/null -w "%{http_code}" -X POST http://localhost:16395/api/v1/chat \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"q": ""}'
# Expected: 400

# Validation: wrong method should return 405
$ curl -s -o /dev/null -w "%{http_code}" -X POST http://localhost:16395/api/v1/chat \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN"
# Expected: 405
```

Run these manually while developing. Automate them in CI once the workflow stabilizes.

---

## Shell-Based Integration Test Script

For a workflow that must meet specific structural guarantees, write a test script:

```bash
#!/usr/bin/env bash
# test_agent.sh
set -euo pipefail

BASE="http://localhost:16395"
AUTH="Authorization: Bearer ${KDEPS_API_AUTH_TOKEN:?set KDEPS_API_AUTH_TOKEN}"
PASS=0
FAIL=0

check() {
  local desc="$1"
  local expected="$2"
  local actual="$3"
  if [ "$actual" = "$expected" ]; then
    echo "  PASS: $desc"
    PASS=$((PASS + 1))
  else
    echo "  FAIL: $desc"
    echo "        expected: $expected"
    echo "        actual:   $actual"
    FAIL=$((FAIL + 1))
  fi
}

echo "=== Starting agent tests ==="

# Test 1: happy path returns 200
STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
  -X POST "$BASE/api/v1/chat" \
  -H "$AUTH" \
  -H "Content-Type: application/json" \
  -d '{"q": "What is kdeps?"}')
check "happy path returns 200" "200" "$STATUS"

# Test 2: response has expected shape
BODY=$(curl -s -X POST "$BASE/api/v1/chat" \
  -H "$AUTH" \
  -H "Content-Type: application/json" \
  -d '{"q": "What is kdeps?"}')
SUCCESS=$(echo "$BODY" | jq -r '.success')
check "response.success is true" "true" "$SUCCESS"
HAS_ANSWER=$(echo "$BODY" | jq 'has("data") and (.data | has("answer"))')
check "response has data.answer" "true" "$HAS_ANSWER"

# Test 3: empty q returns 400
STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
  -X POST "$BASE/api/v1/chat" \
  -H "$AUTH" \
  -H "Content-Type: application/json" \
  -d '{"q": ""}')
check "empty q returns 400" "400" "$STATUS"

# Test 4: missing q returns 400
STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
  -X POST "$BASE/api/v1/chat" \
  -H "$AUTH" \
  -H "Content-Type: application/json" \
  -d '{}')
check "missing q returns 400" "400" "$STATUS"

# Test 5: wrong method returns 405
STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
  -H "$AUTH" \
  "$BASE/api/v1/chat")
check "GET returns 405" "405" "$STATUS"

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
[ "$FAIL" -eq 0 ] || exit 1
```

Run it:

```bash
# Start the agent in another terminal
$ export KDEPS_API_AUTH_TOKEN=dev-token
$ kdeps run workflow.yaml

# Run the test suite
$ bash test_agent.sh
```

This script tests structure and status codes — the parts that are fully deterministic. Add a test case for every validation rule and every `onError` path you define.

---

## Testing With `--dev` Hot Reload

During development, run the workflow with `--dev` so changes to resource files take effect without restarting:

```bash
$ kdeps run workflow.yaml --dev
```

Then edit a resource file and re-run your curl command. The server reloads the workflow automatically. This cuts the feedback loop from "restart, wait, curl" to "save, curl".

---

## Testing Validation Rules

For every `check:` expression in a `validations:` block, write a test case that verifies the rejection:

```yaml
# resources/validate.yaml
validations:
  check:
    - get('email') matches '^[^@]+@[^@]+\\.[^@]+$'
    - len(get('message')) <= 1000
    - get('priority') in ['low', 'medium', 'high']
  error:
    code: 400
    message: "invalid input"
```

Corresponding test cases:

```bash
# Bad email
curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE/api/v1/submit" \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -d '{"email": "not-an-email", "message": "hello", "priority": "low"}'
# Expected: 400

# Message too long (1001 chars)
LONG=$(python3 -c "print('x' * 1001)")
curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE/api/v1/submit" \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -d "{\"email\": \"a@b.com\", \"message\": \"$LONG\", \"priority\": \"low\"}"
# Expected: 400

# Invalid priority
curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE/api/v1/submit" \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -d '{"email": "a@b.com", "message": "hello", "priority": "urgent"}'
# Expected: 400

# Valid request
curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE/api/v1/submit" \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -d '{"email": "a@b.com", "message": "hello", "priority": "high"}'
# Expected: 200
```

---

## Testing Session Persistence

Session tests require cookie handling:

```bash
#!/usr/bin/env bash
# test_session.sh
COOKIE_JAR=$(mktemp)
BASE="http://localhost:16395"
AUTH="Authorization: Bearer ${KDEPS_API_AUTH_TOKEN:?set KDEPS_API_AUTH_TOKEN}"

# First request: start a session, set a value
curl -s -c "$COOKIE_JAR" -X POST "$BASE/api/v1/chat" \
  -H "$AUTH" \
  -H "Content-Type: application/json" \
  -d '{"message": "my name is Alice"}' > /dev/null

# Second request: same session — should remember the name
REPLY=$(curl -s -b "$COOKIE_JAR" -X POST "$BASE/api/v1/chat" \
  -H "$AUTH" \
  -H "Content-Type: application/json" \
  -d '{"message": "what is my name?"}' | jq -r '.data.answer')

echo "Reply: $REPLY"
echo "$REPLY" | grep -qi "alice" && echo "PASS: session persisted name" \
  || echo "FAIL: session did not persist name"

rm "$COOKIE_JAR"
```

This test relies on LLM behavior (whether the model mentions "Alice") so it is not fully automatable. But it verifies that session values are present in the context, which is the structural guarantee you need.

To test the structural part only — that the session value was written and read — use a resource that returns the session value directly without LLM involvement:

```yaml
# resources/echo-session.yaml (test-only resource)
actionId: echoSession
before:
  - set('stored_name', get('stored_name', 'session'))
apiResponse:
  success: true
  response:
    name: get('stored_name')
```

```bash
# First request: write to session
curl -s -c /tmp/c.txt -X POST "$BASE/api/v1/store" \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -d '{"name": "Alice"}'

# Second request: read from session — LLM not involved
NAME=$(curl -s -b /tmp/c.txt -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  "$BASE/api/v1/echo" | jq -r '.data.name')
[ "$NAME" = "Alice" ] && echo "PASS" || echo "FAIL: got '$NAME'"
```

---

## Testing onError Paths

Test your error handling by injecting known failures. The simplest way is a resource that is designed to fail:

```yaml
# resources/fetch-data.yaml
actionId: fetchData
httpClient:
  url: "&#123;&#123; get('api_url') &#125;&#125;"
  method: GET
onError:
  action: continue
  fallback: {"error": true, "message": "fetch failed"}
```

```bash
# Pass an invalid URL to trigger the onError path
curl -s -X POST "$BASE/api/v1/process" \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -d '{"api_url": "http://does-not-exist.invalid/data"}' | jq .

# Expected: success: true, with fallback data in the response
# (the workflow continued despite the failed fetch)
```

For retry testing, use a URL that reliably returns a 500 status to exhaust retries.

---

## Testing Agent Mode Tool Selection

Agent mode (`kdeps [path]`) is non-deterministic — the LLM chooses which tools to invoke. You cannot assert "tool X was called" reliably. What you can test:

**1. The workflow registered as a tool does not error when invoked directly:**

Run each workflow individually in workflow mode first. If the workflow works in isolation via `kdeps run`, it will work when the agent invokes it as a tool. Test each workflow's happy path and error paths before exposing it to agent mode.

**2. Validate the agent's tool registry:**

```bash
$ kdeps validate ./agents/
```

`kdeps validate` on a directory checks every workflow in it. All must pass before the agent is started.

**3. Test bot mode bot input stateless execution by setting `executionType: stateless` in workflow.yaml:**

In bot source workflows, set `executionType: stateless`, then pipe a JSON message and capture stdout:

```bash
echo '{"message": {"text": "What is kdeps?", "from": {"id": 1}, "chat": {"id": 1}, "platform": "telegram"&#125;&#125;' \
  | kdeps run workflow.yaml
```

This runs the workflow once and exits — no live connection needed.

---

## CI/CD Integration

Add the test script to your CI pipeline. A minimal GitHub Actions example:

```yaml
# .github/workflows/test.yml
name: Agent Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install kdeps
        run: curl -fsSL https://kdeps.com/install.sh | bash

      # No LLM server needed: the default file backend downloads the
      # llamafile on first run. Cache it so CI doesn't re-download 1.1 GB.
      - name: Cache llamafile models
        uses: actions/cache@v4
        with:
          path: ~/.kdeps/models
          key: llamafiles-llama3.2-1b

      - name: Start agent
        run: kdeps run workflow.yaml &
        env:
          DATABASE_URL: $&#123;&#123; secrets.DATABASE_URL &#125;&#125;

      - name: Wait for agent to be ready
        run: |
          for i in $(seq 1 30); do
            curl -sf http://localhost:16395/health && break
            sleep 1
          done

      - name: Run tests
        run: bash test_agent.sh
```

Keep CI tests focused on structural guarantees. Do not assert on LLM output text in CI — it will be flaky.

---

## What Not to Test

- **LLM output content** — it varies between runs, models, and model versions. Test the shape of the response, not the words.
- **Response latency in assertions** — LLM inference time is variable. Use timeouts in curl (`--max-time 30`) but do not assert that response time is under N seconds in a test suite.
- **Exact expression evaluation results** — test expressions in isolation using `kdeps validate`, not by checking live response fields.

The goal of automated testing for an AI agent is to verify that the plumbing works: inputs reach the right place, outputs have the right shape, errors are handled correctly. Whether the LLM said something intelligent is a separate concern.

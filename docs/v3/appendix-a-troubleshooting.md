# Appendix A: Troubleshooting

This appendix covers the most common failure modes in kdeps, organized by symptom. Each entry follows the same format: what you observe, why it happens, and how to fix it.

---

## Resource Did Not Execute

**Symptom:** A resource you expected to run is absent from the logs. Its output is not in the data store.

**Why:** The resource is not reachable from the workflow's `targetActionId` through `requires:` edges. kdeps only executes resources on the path to the target. Resources outside that path are silently skipped.

**Fix:**

1. Run `kdeps validate workflow.yaml` — it will report unreachable resources.
2. Check the `requires:` chain from your suspected resource back to `targetActionId`. Every link must be present.
3. If the resource is a side-effect (logging, audit write) and intentionally has no downstream consumer, add it to `requires:` of the terminal resource:

```yaml
# resources/audit.yaml
actionId: audit
requires: [llm]
sql:
  connectionName: main
  query: "INSERT INTO logs ..."

# resources/respond.yaml
actionId: respond
requires: [llm, audit]    # <-- force audit to run before respond
apiResponse:
  success: true
  response: ...
```

---

## get() Returns null

**Symptom:** An expression like `get('myResource')` evaluates to `null` or `nil`. Downstream resources fail or produce empty output.

**Why (most common):** The resource with `actionId: myResource` has not executed yet when the expression runs, or it failed silently. The data store only contains values for resources that have successfully completed.

**Fix — check execution order:**

```yaml
# Wrong: resource B reads from A but does not declare a dependency
# resources/b.yaml
actionId: B
chat:
  prompt: "&#123;&#123; get('A') &#125;&#125;"   # A may not have run yet
```

```yaml
# Correct: declare the dependency
actionId: B
requires: [A]              # guarantees A completes first
chat:
  prompt: "&#123;&#123; get('A') &#125;&#125;"
```

**Fix — check the actionId spelling:**

`get()` is case-sensitive and matches the exact string in `actionId:`. `get('MyResource')` and `get('myresource')` are different keys.

**Fix — check for upstream failure:**

If resource A failed (non-zero exit, LLM error, SQL error), it produces no output. `get('A')` is null in all downstream resources. Run with `--instrument` to see where execution stopped:

```bash
$ kdeps run workflow.yaml --instrument
```

**Fix — use a default:**

If null is acceptable and you want a fallback:

```yaml
prompt: "&#123;&#123; get('A') or 'no context available' &#125;&#125;"
```

---

## Validation Always Fails (or Never Fires)

**Symptom A:** Every request hits the validation error even with correct input.

**Symptom B:** Validation never fires — bad input passes through.

**Why A:** The expression in `validations.check` references a key that does not exist in the data store at validation time, so it evaluates to `false` or null.

**Why B:** The resource with the `validations:` block is not in the execution path to `targetActionId`.

**Fix A — inspect what is in the data store at validation time:**

```bash
$ kdeps run workflow.yaml --instrument
```

Look for the resource's log line and check what keys are populated. HTTP request body fields are available as top-level keys via `get()`. A POST body `{"q": "hello"}` makes `get('q')` available immediately.

Common pitfall — reading a key before it is set:

```yaml
# Wrong: 'normalized' is set in before: but validations: runs before before:
validations:
  check:
    - get('normalized') != ''    # normalized does not exist yet here

before:
  - set('normalized', trim(get('q')))
```

```yaml
# Correct: validate the raw input directly
validations:
  check:
    - get('q') != ''

before:
  - set('normalized', trim(get('q')))
```

**Fix B:** Add the validating resource to the DAG path. A resource with `validations:` but no path to `targetActionId` is never executed.

---

## LLM Does Not See Context From a Previous Resource

**Symptom:** The LLM response ignores data you fetched in a previous resource (SQL result, HTTP response, scraped text).

**Why:** The prompt expression is referencing the wrong key, the fetched data has an unexpected shape, or the context is too large and was silently truncated.

**Fix — verify the key name and shape:**

```bash
$ kdeps run workflow.yaml --instrument
```

Find the log output for the upstream resource (e.g., `sql`). The output is stored under the resource's `actionId`. If `actionId: lookup` returns `[{"name": "Alice"}]`, then:

```yaml
# Wrong
prompt: "The user is &#123;&#123; get('user') &#125;&#125;"      # key 'user' does not exist

# Correct
prompt: "The user is &#123;&#123; get('lookup')[0].name &#125;&#125;"
```

**Fix — convert structured data to a readable string:**

LLMs perform better with text than with raw JSON objects:

```yaml
before:
  - set('context', json(get('lookup')))    # serialize to JSON string

chat:
  prompt: |
    Here is the relevant data:
    &#123;&#123; get('context') &#125;&#125;

    Question: &#123;&#123; get('q') &#125;&#125;
```

**Fix — check context length:**

Most models have a context window limit. Very long context (>8k tokens) may be silently truncated. Use `get('text')[0:4000]` to cap the input.

---

## DAG Cycle Error

**Symptom:** `kdeps validate workflow.yaml` reports `cycle detected` or `circular dependency`.

**Why:** Resource A requires B, and B requires A (directly or transitively). kdeps cannot determine execution order.

**Fix:** Draw the dependency graph. Find the cycle. Remove the edge that creates it — usually a resource that should not be in `requires:` at all:

```yaml
# Broken: cycle between enrich and summarize
# enrich requires [summarize]
# summarize requires [enrich]
```

If two resources genuinely need each other's output, they must be merged into one resource, or the processing must be rearranged so data flows in one direction.

---

## Session Not Persisting Between Requests

**Symptom:** `set('key', value, 'session')` appears to work but on the next request `get('key')` returns null.

**Why (most common):** The session cookie is not being sent back with subsequent requests. kdeps sets a `Set-Cookie` header on the first response. If the client does not resend that cookie, each request starts a new session.

**Fix — with curl:**

```bash
# First request: capture the cookie
$ curl -c /tmp/session.txt -X POST http://localhost:16395/api/v1/chat \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message": "hello"}'

# Subsequent requests: send the cookie back
$ curl -b /tmp/session.txt -X POST http://localhost:16395/api/v1/chat \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message": "what did I just say?"}'
```

**Fix — check session TTL:**

If the session expired, it is deleted and a new one is created. Check `settings.session.ttl` in `workflow.yaml`. The default is `1h`.

**Fix — verify session type is configured:**

Sessions require explicit configuration:

```yaml
settings:
  session:
    type: sqlite
    path: "/data/sessions.db"
    ttl: "2h"
```

Without this block, every request is stateless and session writes are silently discarded.

---

## HTTP Request Returns 500 With No Useful Message

**Symptom:** `kdeps run workflow.yaml` is running, curl returns `{"success": false, "error": "internal server error"}` with no details.

**Fix — run with trace to see the full error:**

```bash
$ kdeps run workflow.yaml --instrument
```

The trace output shows which resource failed and the exact error message. Common causes:

| Cause | What you see in --instrument |
|---|---|
| SQL connection failed | `failed to connect: dial tcp ... connection refused` |
| LLM backend unreachable | `failed to call LLM: connection refused` or `404` |
| Python script syntax error | `SyntaxError` in the python executor output |
| Shell command not found | `exec: "mycommand": executable file not found` |
| Expression parse error | `failed to parse expression: ...` |
| Missing required field | `resource must specify at least one execution type` |

---

## Expression Evaluation Error: "undefined: X"

**Symptom:** `--instrument` shows `failed to evaluate expression: undefined: myVar`.

**Why:** The variable `myVar` has not been set in the data store at the point the expression runs.

**Fix:** Check where `myVar` is set. If it is set in a `before:` block of the same resource, expressions in `validations.check` run before `before:`, so the variable is not available there. If it is set by another resource, check that the resource has executed (see "get() Returns null" above).

---

## Resource Runs But Output Is Empty or Wrong Shape

**Symptom:** `get('myResource')` returns something but not the structure you expect — it is a string instead of an object, or an array instead of a scalar.

**Why:** Each resource type returns a specific output shape:

| Resource | Output shape |
|---|---|
| `chat:` | string (the LLM's response text) |
| `chat:` with `jsonResponse: true` | object (parsed JSON) |
| `sql:` | array of row objects |
| `sql:` single-row query | `[{"col": "val"}]` — still an array, access `[0]` |
| `httpClient:` | object with `status`, `body`, `headers` |
| `exec:` | object with `stdout`, `stderr`, `exitCode` |
| `python:` | object with `result`, `stdout`, `stderr` |
| `scraper:` | object with `text`, `html`, `url` |
| `embedding:` (search) | array of `{text, score, metadata}` objects |

**Fix — access the correct field:**

```yaml
# sql: returns an array
sql:
  query: "SELECT name FROM users WHERE id = $1"

# Wrong: get('lookup') is the full array
prompt: "The user is &#123;&#123; get('lookup') &#125;&#125;"

# Correct: index into the array
prompt: "The user is &#123;&#123; get('lookup')[0].name &#125;&#125;"
```

```yaml
# httpClient: body is a string; parse it for field access
httpClient:
  url: "https://api.example.com/user"

before:
  - set('user', fromJSON(get('fetch').body))

prompt: "&#123;&#123; get('user').name &#125;&#125;"
```

---

## kdeps validate Passes But Runtime Fails

**Symptom:** `kdeps validate workflow.yaml` reports no errors, but running the workflow fails.

**Why:** Validation checks schema correctness (required fields, valid types, no cycles). It does not verify runtime conditions: whether the database is reachable, whether the LLM model name is valid, whether environment variables are set.

**Fix — use `kdeps doctor`:**

```bash
$ kdeps doctor
```

`kdeps doctor` checks runtime prerequisites: the LLM backend (the models directory for the default llamafile backend, or Ollama connectivity when opted in), Docker availability, required environment variables referenced in `workflow.yaml`. It is the right tool for "will this actually run?" questions.

**Fix — verify environment variables:**

Every `${VAR}` reference in `workflow.yaml` must be set in the environment. Missing variables silently become empty strings:

```bash
$ grep -r '\${' workflow.yaml resources/    # find all variable references
$ env | grep -E "DATABASE_URL|OPENAI_KEY|..."  # verify they are set
```

---

## Deployment: Docker Image Starts But Agent Returns Errors

**Symptom:** The Docker image runs without crashing, but requests return errors that did not occur locally.

**Common causes:**

1. **Environment variables not passed to the container.** Use `-e` or `--env-file`:
   ```bash
   $ docker run -e DATABASE_URL=... -e OPENAI_API_KEY=... myagent:latest
   ```

2. **Volume mounts missing.** If your workflow uses `file:` input or writes to a local SQLite database, those paths do not exist inside the container unless you mount them:
   ```bash
   $ docker run -v /local/data:/data myagent:latest
   ```

3. **Ollama not reachable.** Inside Docker, `localhost` refers to the container, not your host machine. Use the host's Docker bridge IP or a service name if using Docker Compose:
   ```yaml
   # ~/.kdeps/config.yaml inside the image
   ollama:
     baseUrl: http://host.docker.internal:11434   # Mac/Windows
     # or: http://172.17.0.1:11434               # Linux bridge
   ```

4. **Port not exposed.** Ensure `--publish` matches `settings.apiServer.portNum` in `workflow.yaml` (default: `16395`):
   ```bash
   $ docker run -p 16395:16395 myagent:latest
   ```

---

## Getting More Information

When the above steps do not identify the problem:

```bash
# Full trace of every resource execution (DAG ordering, resource inputs/outputs)
$ kdeps run workflow.yaml --instrument

# Debug logging (expression values, data store state)
$ kdeps run workflow.yaml --debug

# Validate schema only (no runtime checks)
$ kdeps validate workflow.yaml

# Runtime prerequisite check
$ kdeps doctor
```

For persistent issues, file a bug at [github.com/kdeps/kdeps/issues](https://github.com/kdeps/kdeps/issues) with the output of `kdeps validate` and `--instrument`.

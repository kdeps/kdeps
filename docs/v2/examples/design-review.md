# Design review panel

Four reviewer resources check a design document in parallel. A rejection ends the reply with a question. The next request answers it, and the session carries the findings across.

```d2
direction: right
req: "POST /api/v1/review\n{doc}" {shape: oval}
req -> scope
req -> clarity
req -> failure-modes
req -> readiness
scope -> reply
clarity -> reply
failure-modes -> reply
readiness -> reply
reply -> out: "passed, findings\nquestion + options" {shape: oval}
```

Full project: [`examples/design-review`](https://github.com/kdeps/kdeps/tree/main/examples/design-review).

## Reviewer

One file per criterion. Each reads only the request, so all four run at the same time.

<div v-pre>

```yaml
# resources/scope.yaml
actionId: scope
validations:
  skip:
    - "get('doc') == nil"   # an answer request carries no doc: skip the review
chat:
  model:
    name: llama3.2:1b
  context:
    prompt: |
      You review design documents for one criterion only.
      Criterion (scope): The design solves the problem it states, no smaller and no larger.
      Reply with JSON only: {"pass": true|false, "findings": ["..."]}

      Design document:
      {{ get('doc') }}
  harness:
    output:
      json: true
      keys: [pass, findings]
```

</div>

## Reply: verdict, question, answer

<div v-pre>

```yaml
# resources/reply.yaml
actionId: reply
flow:
  requires: [scope, clarity, failure-modes, readiness]
before:
  # keep the findings in the session for the answer request
  # (own key names: get('doc') also reads the session)
  - >-
    get('doc') != nil ? set('review_findings', {
      "scope": output('scope').findings,
      "clarity": output('clarity').findings,
      "failure-modes": output('failure-modes').findings,
      "readiness": output('readiness').findings
    }, 'session') : nil
  - >-
    set('passed', get('doc') != nil && all(
      [output('scope'), output('clarity'), output('failure-modes'), output('readiness')],
      {.pass == true}))
apiResponse:
  success: true
  response:
    passed: "{{ get('passed') }}"
    findings: "{{ get('review_findings', 'session') }}"
    question: "{{ get('doc') != nil && !get('passed') ? 'Revise the design or ship it as is?' : '' }}"
    options: "{{ get('doc') != nil && !get('passed') ? ['1. Revise and resubmit', '2. Ship as is'] : [] }}"
    decision: "{{ get('answer') == '2' ? 'Shipping with the open findings above.' : '' }}"
```

</div>

`workflow.yaml` needs a `session:` block (`type: sqlite`) for the findings to survive between the two requests.

## Run

```bash
kdeps run examples/design-review/workflow.yaml
curl -s -c ck -b ck -H 'Content-Type: application/json' -d '{"doc": "Retries: TBD"}' localhost:16400/api/v1/review
curl -s -c ck -b ck -H 'Content-Type: application/json' -d '{"answer": "2"}' localhost:16400/api/v1/review
```

The first reply has `"passed": false`, the findings, a `question` and numbered `options`. The second has the `decision` and the same findings, read from the session.

In agent mode the same panel is the `design-review` [judge preset](/agent/judges), and the model asks with [`ask_question`](/agent/tools).

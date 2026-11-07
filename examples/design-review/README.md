# Design review

Four reviewers (scope, clarity, failure modes, readiness) check a design document at the same time. When any rejects it, the reply ends with a question, and the next request answers it.

```bash
kdeps run examples/design-review/workflow.yaml

# 1. Review a doc. Keep the session cookie: the answer needs it.
curl -s -c ck -b ck -H 'Content-Type: application/json' \
  -d '{"doc": "Retries: TBD"}' localhost:16400/api/v1/review
# {"passed": false, "findings": {...}, "question": "Revise the design or ship it as is?",
#  "options": ["1. Revise and resubmit", "2. Ship as is"], ...}

# 2. Answer the question. The findings come back from the session.
curl -s -c ck -b ck -H 'Content-Type: application/json' \
  -d '{"answer": "2"}' localhost:16400/api/v1/review
# {"decision": "Shipping with the open findings above.", "findings": {...}, ...}
```

- The reviewers only read the request, so kdeps runs them in parallel (up to the `parallel` cap, 4 by default).
- The session keeps the reviewed doc and findings under their own keys (`reviewed_doc`, `review_findings`): `get('doc')` also reads the session, so reusing the request's name would shadow the next doc.
- The agent-mode version is the `design-review` judge preset: `/harness preset design-review`.

# Use conditionals and list operations

*Applies to workflow mode.*

## Overview

In this tutorial you build one resource that demonstrates every control-flow
tool kdeps expressions give you: the ternary operator, the logical operators,
and the list functions `filter`, `map`, `all`, and `any`.

This tutorial is for developers who have completed the
[quickstart](/getting-started/quickstart). It assumes you know:

- Basic YAML
- Basic boolean logic

By the end you will be able to:

- Choose a value with `cond ? a : b`
- Combine booleans with `&&`, `||`, `!`
- Filter, map, and test a list with `filter`, `map`, `all`, `any`

## Background

kdeps expressions run on [expr-lang](https://expr-lang.org/). Inside a
`before:` or `after:` block each line is a bare expression - no `{{ }}`. The
list functions take a predicate in braces where `.` is the current element:
`filter(people, {.age >= 18})`.

## Before you start

- kdeps installed (`kdeps --version`).
- A working directory for the project.

## Step 1: create the project

```bash
mkdir control-flow-demo
cd control-flow-demo
mkdir resources
```

## Step 2: define the route

Create `workflow.yaml`:

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: control-flow-demo
  version: "1.0.0"
  targetActionId: demo

settings:
  apiServer:
    portNum: 16395
    routes:
      - path: /api/demo
        methods: [GET]
```

## Step 3: the demo resource

Create `resources/demo.yaml`:

<div v-pre>

```yaml
# resources/demo.yaml
actionId: demo
name: Control flow demo
validations:
  methods: [GET]
  routes: [/api/demo]

before:
  # Ternary: cond ? then : else
  - "set('label_20', 20 >= 18 ? 'adult' : 'child')"
  - "set('label_15', 15 >= 18 ? 'adult' : 'child')"

  # Logical operators
  - "set('and_', 20 >= 18 && true)"
  - "set('or_',  false || true)"
  - "set('not_', !false)"

  # A list to work with
  - >-
    set('people', [
      {"name": "Alice", "age": 20},
      {"name": "Bob",   "age": 15},
      {"name": "Carol", "age": 30}
    ])

  # List operations - {.field} is the predicate, . is the current element
  - "set('adults',    filter(get('people'), {.age >= 18}))"
  - "set('names',     map(get('people'), {.name}))"
  - "set('allAdults', all(get('people'), {.age >= 18}))"
  - "set('anyMinor',  any(get('people'), {.age < 18}))"

apiResponse:
  success: true
  response:
    ternary:
      age_20: "{{ get('label_20') }}"
      age_15: "{{ get('label_15') }}"
    logical:
      and: "{{ get('and_') }}"
      or:  "{{ get('or_') }}"
      not: "{{ get('not_') }}"
    lists:
      adults:     "{{ get('adults') }}"
      names:      "{{ get('names') }}"
      all_adults: "{{ get('allAdults') }}"
      any_minor:  "{{ get('anyMinor') }}"
```

</div>

## Step 4: validate and run

```bash
kdeps validate .
export KDEPS_API_AUTH_TOKEN=dev-token
kdeps run .
```

```bash
curl http://localhost:16395/api/demo \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN"
```

Response:

```json
{
  "success": true,
  "data": {
    "ternary": { "age_20": "adult", "age_15": "child" },
    "logical": { "and": true, "or": true, "not": true },
    "lists": {
      "adults": [{ "name": "Alice", "age": 20 }, { "name": "Carol", "age": 30 }],
      "names": ["Alice", "Bob", "Carol"],
      "all_adults": false,
      "any_minor": true
    }
  }
}
```

## Summary

You used, in one resource:

- The ternary operator to pick a label
- `&&`, `||`, `!` to combine booleans
- `filter`, `map`, `all`, `any` with `{.field}` predicates over a list

## Next steps

- [Expressions](/concepts/expressions) - where expressions run
- [Expression operators](/reference/expression-operators) - the full operator list
- [Expression functions reference](/reference/expression-functions-reference) - every function
- [Validation and control flow](/concepts/validation-and-control) - `skip` and `check`

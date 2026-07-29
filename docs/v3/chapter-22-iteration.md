# Chapter 22: Iteration — items and loop

kdeps has two iteration mechanisms. `items:` runs a resource once per entry in a fixed list — like a for-each. `loop:` runs a resource repeatedly while a condition is true — like a while loop. Together they make kdeps Turing-complete.

## items: — For-Each Iteration

`items:` attaches to any resource and causes it to execute once per item in the list. Each execution has access to context about its position in the list.

### Basic Usage

```yaml
# resources/classify-messages.yaml
actionId: classifyMessages
items:
  - "Your account has been compromised, click here"
  - "Meeting rescheduled to 3pm tomorrow"
  - "Congratulations, you won a prize!"

chat:
  model: llama3.2:1b
  prompt: "Classify as spam or not-spam: &#123;&#123; get('current') &#125;&#125;"
  jsonResponse: true
```

The resource runs three times. On each run, `get('current')` holds the current item. The resource's final output is a list of all results, one per iteration.

### Dynamic Items from the Data Store

Items do not need to be literal. Read them from a prior resource:

```yaml
# resources/fetch-urls.yaml
actionId: fetchUrls
sql:
  connectionName: main
  query: "SELECT url FROM documents WHERE processed = false LIMIT 50"

# resources/process-each.yaml
actionId: processEach
requires: [fetchUrls]
items: "&#123;&#123; get('fetchUrls') &#125;&#125;"    # items from the SQL result set

scraper:
  url: "&#123;&#123; get('current').url &#125;&#125;"
  timeout: 30
```

`items: "&#123;&#123; get('fetchUrls') &#125;&#125;"` evaluates to the list of rows from the SQL query. Each row is one item.

### Item Context

Inside an items-bound resource, special getters are available:

| Getter | Description |
|---|---|
| `get('current')` | Current item value |
| `get('prev')` | Previous item (null if first) |
| `get('next')` | Next item (null if last) |
| `get('index')` | Current index, 0-based |
| `get('count')` | Total number of items |
| `get('all')` | Array of all items |

Or via the `item` object methods (equivalent):

```yaml
after:
  - set('idx', item.index())
  - set('curr', item.current())
  - set('prev', item.prev())      # nil on first iteration
  - set('next', item.next())      # nil on last iteration
  - set('total', item.count())
  - set('all', item.values())     # the full items array
```

`item.values()` is useful when you need to reference the full list from within an iteration — for example, to check if the current item is unique or to build a summary that references earlier items.

### Item-Scoped Storage

Each iteration can store values that survive to the next iteration using `'item'` scope:

```yaml
items: "&#123;&#123; get('records') &#125;&#125;"

before:
  - set('running_total', get('running_total', 'item') or 0)
  - set('running_total', get('running_total') + get('current').amount, 'item')

chat:
  prompt: "Record &#123;&#123; item.index() + 1 &#125;&#125; of &#123;&#123; item.count() &#125;&#125;: &#123;&#123; get('current').description &#125;&#125;. Running total: &#123;&#123; get('running_total') &#125;&#125;"
```

Item-scoped values persist across iterations of the same resource but do not leak into the parent workflow's data store.

### Collecting Results

After an items-bound resource completes, `get('resourceActionId')` returns an **array** containing all iteration outputs:

```yaml
# resources/score-items.yaml
actionId: scoreItems
items: "&#123;&#123; get('candidates') &#125;&#125;"
chat:
  model: llama3.2:1b
  jsonResponse: true
  prompt: "Score from 0-100: &#123;&#123; get('current').text &#125;&#125;"

# resources/rank.yaml
actionId: rank
requires: [scoreItems]
python:
  script: |
    import json
    scores = &#123;&#123; get('scoreItems') &#125;&#125;    # array of all per-item results
    ranked = sorted(scores, key=lambda x: x.get('score', 0), reverse=True)
    print(json.dumps(ranked))
```

### Batch Processing Pattern

```yaml
# resources/validate-batch.yaml
actionId: validateBatch
validations:
  check:
    - get('documents') != null
    - len(get('documents')) > 0
    - len(get('documents')) <= 100
  error:
    code: 400
    message: "documents array required (max 100)"
exec:
  command: "echo ok"

# resources/extract-each.yaml
actionId: extractEach
requires: [validateBatch]
items: "&#123;&#123; get('documents') &#125;&#125;"
chat:
  model: llama3.2:1b
  jsonResponse: true
  prompt: |
    Extract: title, author, date from this document text.
    Return JSON only.
    
    &#123;&#123; get('current').text[0:2000] &#125;&#125;

# resources/respond.yaml
actionId: respond
requires: [extractEach]
apiResponse:
  success: true
  response:
    processed: len(get('documents'))
    results: get('extractEach')
```

### Image Batch Processing

`items:` works with vision-capable models. Pass a list of file paths and attach each as a `files:` upload:

```yaml
# resources/analyze-images.yaml
actionId: analyzeImages
items:
  - "/uploads/photo1.jpg"
  - "/uploads/photo2.jpg"
  - "/uploads/photo3.jpg"
chat:
  model: llama3.2-vision
  prompt: "Describe what is in this image. Be specific about objects, people, and context."
  files:
    - "&#123;&#123; get('current') &#125;&#125;"    # current item is the file path for this iteration
```

For dynamic image lists from the request:

```yaml
items: "&#123;&#123; get('images', 'filepath') &#125;&#125;"    # all uploaded image files
chat:
  model: llama3.2-vision
  prompt: "Classify this image: &#123;&#123; get('current') &#125;&#125;"
  files:
    - "&#123;&#123; get('current') &#125;&#125;"
```

The result array contains one description per image, in the same order as the input list. Access them with `get('analyzeImages')[0]`, `get('analyzeImages')[1]`, etc.

---

## loop: — While-Loop Iteration

`loop:` repeats a resource body while an expression is true (or a fixed number of times). It enables patterns that `items:` cannot: iterating until a condition is met, retrying until success, accumulating data across an unknown number of steps.

### Basic Usage

```yaml
# resources/count.yaml
actionId: count
loop:
  while: "loop.index() < 5"
  maxIterations: 1000    # safety cap
after:
  - set('result', loop.count())
apiResponse:
  success: true
  response:
    count: get('result')
```

`loop.index() < 5` runs for indices 0, 1, 2, 3, 4 — five iterations. `maxIterations` is a safety cap to prevent infinite loops.

### Loop Context

| Callable | Description |
|---|---|
| `loop.index()` | Current iteration index, 0-based |
| `loop.count()` | Current iteration count, 1-based |
| `loop.results()` | Array of outputs from all prior iterations |

### Loop-Scoped Storage

```yaml
# Store values that persist across loop iterations
before:
  - set('accumulated', get('accumulated', 'loop') or [])

after:
  - set('accumulated', get('accumulated') + [get('loopResult')], 'loop')
```

`set('key', value, 'loop')` and `get('key', 'loop')` scope to the current loop execution.

### Iterating Until a Condition

```yaml
# resources/search-until-found.yaml
actionId: searchUntilFound
loop:
  while: "get('found', 'loop') != true and loop.index() < 10"
  maxIterations: 10

before:
  - set('page', loop.index() + 1)

httpClient:
  method: GET
  url: "https://api.example.com/search?q=&#123;&#123; get('query') &#125;&#125;&page=&#123;&#123; get('page') &#125;&#125;"

after:
  - set('found', len(get('searchUntilFound').results) > 0 and
      get('searchUntilFound').results[0].score > 0.9, 'loop')
  - set('best_match', get('searchUntilFound').results[0] or null, 'loop')
```

The loop continues fetching pages until a high-confidence result is found or 10 pages are exhausted.

### Polling Pattern

```yaml
# resources/poll-job.yaml
actionId: pollJob
loop:
  while: "get('status', 'loop') not in ['completed', 'failed']"
  maxIterations: 60
  every: "5s"           # wait 5 seconds between iterations (scheduled polling)

httpClient:
  method: GET
  url: "https://api.example.com/jobs/&#123;&#123; get('jobId') &#125;&#125;"

after:
  - set('status', get('pollJob').status, 'loop')
  - set('result', get('pollJob').result, 'loop')
```

`every: "5s"` pauses 5 seconds between iterations. This turns the loop into a polling mechanism without blocking resources.

### Accumulating LLM Conversations

```yaml
# resources/chain-of-thought.yaml
actionId: chainOfThought
loop:
  while: "get('done', 'loop') != true and loop.index() < 10"
  maxIterations: 10

before:
  - set('history', get('history', 'loop') or [])
  - set('history', get('history') + [{"role": "user", "content": get('next_prompt') or get('q')}], 'loop')

chat:
  model: llama3.2:1b
  messages: "&#123;&#123; get('history') &#125;&#125;"

after:
  - set('history', get('history') + [{"role": "assistant", "content": get('chainOfThought')}], 'loop')
  - set('done', get('chainOfThought') contains 'FINAL ANSWER', 'loop')
  - set('next_prompt', 'Continue your reasoning.', 'loop')
```

Each iteration adds to the conversation history. The loop terminates when the LLM includes "FINAL ANSWER" in its response.

### maxIterations Safety

Always set `maxIterations`. A `loop:` without it uses a default cap of 1000. For tight polling loops (checking a job status), 60–120 is sensible. For reasoning chains, 5–15 is usually enough. A loop that runs 1000 iterations against an LLM API will cost real money and take a long time.

## items vs. loop: Choosing the Right Tool

| `items:` | `loop:` |
|---|---|
| You have a fixed list to iterate over | You do not know in advance how many iterations are needed |
| Each item is independent | Each iteration can depend on the previous |
| Batch processing | Polling, retrying, chain-of-thought |
| Result is a list of outputs | Result is derived from loop-scoped state |

When in doubt, reach for `items:`. It is simpler, easier to reason about, and produces predictable output. Use `loop:` only when the termination condition is genuinely dynamic.

X> ## Exercise
X>
X> Build a workflow that classifies a batch of customer support tickets and produces a summary report.
X>
X> The endpoint accepts `POST /api/v1/triage` with a JSON body:
X> ```json
X> {
X>   "tickets": [
X>     "My order hasn't arrived after 3 weeks",
X>     "How do I reset my password?",
X>     "The app crashes when I open settings",
X>     "Can I change my subscription plan?",
X>     "I was charged twice for the same order"
X>   ]
X> }
X> ```
X>
X> Requirements:
X>
X> 1. Use `items: get('tickets')` on a `chat:` resource to classify each ticket into one of: `billing`, `technical`, `shipping`, `account`. Store each result.
X> 2. After all tickets are classified, use a `python:` resource (no `items:`) to aggregate the results: count how many tickets fall into each category.
X> 3. Return the per-ticket classifications and the category counts in the response.
X>
X> Verify that all 5 tickets are classified and the counts add up to 5.
X>
X> Then add a `loop:` variant: instead of processing a fixed list, poll for new unprocessed tickets from a SQLite table every iteration. The loop terminates when the query returns an empty result set. Compare the two approaches — which is simpler for your use case?
X>
X> **Stretch goal:** Add `onError:` to the classification resource so a single LLM failure on one ticket does not abort the entire batch, substituting `"unknown"` as the fallback category.

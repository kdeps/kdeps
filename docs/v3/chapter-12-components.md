# Chapter 12: Components

A component is a reusable, self-contained resource bundle. Where a workflow is a complete runnable agent with an HTTP server, a component is a building block — callable from any workflow, encapsulating a capability like web scraping, search, or browser automation.

kdeps has two kinds of components: registry components you install from the kdeps component registry, and custom components you build yourself.

## Why Components

Without components, capabilities get duplicated. Every workflow that needs web search copies the same `searchWeb:` resource config. Every workflow that needs scraping copies the same scraper setup. When the underlying implementation changes, you update dozens of files.

Components solve this:
- Define a capability once, in one place
- Install it and use it in any workflow with a single `component:` declaration
- Share components across projects via the registry

Components also enable **LLM tool composition** — when exposed via `componentTools:` on a `chat:` resource, the LLM can call components as function-calling tools, turning a static prompt into a dynamic research session.

## Registry Components

The kdeps component registry distributes pre-built capability components. Install them once; they are available to all workflows.

```bash
# Install available components
$ kdeps registry install scraper     # web / document text extraction
$ kdeps registry install search      # web search
$ kdeps registry install embedding   # vector store operations
$ kdeps registry install browser     # browser automation
```

List installed components:

```bash
$ kdeps registry list
```

Uninstall:

```bash
$ kdeps registry uninstall scraper
```

### Using a Registry Component

```yaml
# resources/fetch-article.yaml
actionId: fetchArticle
component:
  name: scraper
  with:
    url: "&#123;&#123; get('articleUrl') &#125;&#125;"
    selector: "article.content"
```

The `component:` block replaces the action field (`chat:`, `sql:`, etc.) — it IS the action. `with:` maps to the component's declared inputs. After execution, the result is available via `output('fetchArticle')`.

### Component Inputs

Each registry component declares a typed input schema. Pass inputs via `with:`:

```yaml
# scraper
component:
  name: scraper
  with:
    url: "https://example.com"
    selector: ".content"    # optional

# search
component:
  name: search
  with:
    query: "&#123;&#123; get('topic') &#125;&#125;"
    maxResults: 5            # optional, default 5

# embedding
component:
  name: embedding
  with:
    operation: search         # index | search | upsert | delete
    text: "&#123;&#123; get('q') &#125;&#125;"
    collection: "my-docs"

# browser
component:
  name: browser
  with:
    url: "&#123;&#123; get('targetUrl') &#125;&#125;"
    engine: chromium
    actions:
      - action: evaluate
        script: "document.title"
```

### Reading Component Output

```yaml
# reads scraper output
after:
  - set('article_text', output('fetchArticle'))

# reads search results (array)
after:
  - set('top_url', output('searchResult')[0].url)

# reads embedding search results
after:
  - set('relevant_docs', output('retrieveContext'))
```

## Components as LLM Tools

This is where components become powerful. The `componentTools:` field on a `chat:` resource registers installed components as function-calling tools:

```yaml
# resources/research.yaml
actionId: research
chat:
  model: llama3.2:1b
  prompt: "Research this topic thoroughly and give me a detailed answer: &#123;&#123; get('topic') &#125;&#125;"
  componentTools:
    - search
    - scraper
    - embedding
```

When the LLM processes this prompt, it has access to three tools:
- `search(query, maxResults)` — web search
- `scraper(url, selector)` — fetch and extract page content
- `embedding(operation, text, collection)` — search the local knowledge base

The LLM decides:
1. What to search for
2. Which results to scrape
3. Whether to check the local knowledge base
4. How to combine the results into an answer

The component's declared input schema becomes the tool's parameter schema. The LLM sees the parameter names, types, and descriptions and uses them to construct correct calls.

**Only opt-in components become tools.** By default, no installed components are registered as tools. You list exactly which components a particular `chat:` resource can use. This limits the LLM's tool scope to what is relevant for the task.

## Custom Components

Build your own components when you have domain-specific functionality you want to reuse across workflows.

### Directory Structure

```
my-workflow/
├── workflow.yaml
└── components/                    # auto-discovered at runtime
    └── data-enricher/
        ├── component.yaml         # component manifest
        └── resources/             # component-specific resources
            ├── fetch.yaml
            ├── transform.yaml
            └── respond.yaml
```

kdeps auto-discovers `components/` at runtime. No registration step, no changes to `workflow.yaml`.

### Component Manifest

```yaml
# components/data-enricher/component.yaml
apiVersion: kdeps.io/v1
kind: Component

metadata:
  name: data-enricher
  version: "1.0.0"
  description: "Enriches a company name with CRM data and recent news"

interface:
  inputs:
    company_name:
      type: string
      required: true
      description: "The company name to enrich"
    include_news:
      type: boolean
      required: false
      default: true
      description: "Whether to fetch recent news"
  
  output:
    type: object
    description: "Enriched company record with CRM data and news"
```

### Component Resources

Component resources are identical to workflow resources, but they cannot contain `settings:` (no server bindings, no port). They are pure resource bundles:

```yaml
# components/data-enricher/resources/crm-lookup.yaml
actionId: crmLookup
sql:
  connectionName: crm
  query: "SELECT * FROM companies WHERE name ILIKE $1 LIMIT 1"
  params:
    - get('company_name')
```

```yaml
# components/data-enricher/resources/news-fetch.yaml
actionId: newsFetch
validations:
  skip:
    - get('include_news') == false
requires: [crmLookup]
searchWeb:
  query: "&#123;&#123; get('company_name') &#125;&#125; news 2024"
  maxResults: 3
```

```yaml
# components/data-enricher/resources/enrich.yaml
actionId: enrich
requires: [crmLookup, newsFetch]
chat:
  model: llama3.2:1b
  prompt: |
    CRM record: &#123;&#123; get('crmLookup') &#125;&#125;
    Recent news: &#123;&#123; get('newsFetch') &#125;&#125;
    
    Summarize the company in 2 sentences.
```

### Using a Custom Component

```yaml
# resources/process-lead.yaml
actionId: processLead
component:
  name: data-enricher
  with:
    company_name: "&#123;&#123; get('company') &#125;&#125;"
    include_news: true
```

The component's internal resources run as a sub-DAG. Their outputs are scoped to the component invocation — they do not pollute the parent workflow's data store. The component's final output is available via `output('processLead')`.

### Packaging Components for Distribution

Custom components can be packaged as `.komponent` archives for sharing:

```bash
$ kdeps registry package ./components/data-enricher/
# Creates data-enricher-1.0.0.komponent

$ kdeps registry publish data-enricher-1.0.0.komponent
# Publishes to the kdeps registry (requires registry credentials)
```

Others install it with:

```bash
$ kdeps registry install data-enricher
```

## Design Principles for Components

**Single responsibility.** A component should do one thing. `web-search`, `page-reader`, `email-sender` — not `do-research-and-write-report`. Composition happens at the workflow or agent level.

**Explicit input schema.** Declare every input with its type, whether it's required, and a description. The description becomes the tool description the LLM sees when the component is used as a tool.

**No side effects by default.** Components that make writes (database inserts, API calls that change state) should be explicitly named to indicate this: `user-creator`, `email-sender`, not generic names. Callers should know they are calling something that has side effects.

**Version your components.** Use semantic versioning in `metadata.version`. When you change a component's input schema in a breaking way, bump the major version. Workflows that depend on the old version continue to work if the registry supports multiple versions.

## The Component Registry at kdeps.io

The official kdeps component registry lives at [kdeps.io](https://kdeps.io). It hosts pre-built components for common capabilities. As the ecosystem grows, community-contributed components become available for domain-specific tasks: Slack integration, GitHub API, Stripe payments, Twilio SMS, and more.

Browse available components:

```bash
$ kdeps registry search email
$ kdeps registry search database
$ kdeps registry info scraper
```

## Publishing to the Registry

Anyone can publish a component, workflow, or agency to the registry. Publishing uses a **formula** — a YAML file submitted as a pull request to `github.com/kdeps/registry`.

### Step 1: Tag a release

```bash
$ git tag v1.0.0 && git push --tags
```

### Step 2: Generate the formula

Run `kdeps registry submit` from your package directory. It downloads the release tarball, computes the SHA256, and prints the formula YAML:

```bash
$ kdeps registry submit --tag v1.0.0
```

Output:
```yaml
name: my-component
version: 1.0.0
type: component          # component | workflow | agency
github: owner/repo
tarball: https://github.com/owner/repo/archive/refs/tags/v1.0.0.tar.gz
sha256: abc123def456...
description: Does one specific thing well.
tags: [llm, extraction]
license: Apache-2.0
```

### Step 3: Submit a PR

Save the formula as `formulas/my-component.yaml` and open a pull request to `github.com/kdeps/registry`. The PR is reviewed for:

- Unique name (no collision with existing packages)
- Valid SHA256 (tarball matches)
- `kind:` in the manifest matches the formula `type:`
- Description present and meaningful

Once merged, anyone can install:

```bash
$ kdeps registry install my-component
```

**Formula field reference:**

| Field | Required | Description |
|---|---|---|
| `name` | yes | Unique package name; lowercase, alphanumeric + hyphens |
| `version` | yes | Semantic version (`1.0.0`) |
| `type` | yes | `component`, `workflow`, or `agency` |
| `github` | yes | `owner/repo` string |
| `tarball` | yes | Full URL to the release tarball |
| `sha256` | yes | Hex SHA256 of the tarball (computed by `kdeps registry submit`) |
| `description` | yes | One sentence describing the package |
| `tags` | no | Search keywords |
| `license` | no | SPDX identifier (`MIT`, `Apache-2.0`, etc.) |

In the next chapter, we take composition one level further — agencies, where multiple complete agents collaborate on complex tasks.

X> ## Exercise
X>
X> Install a component from the kdeps registry and wire it into a workflow that answers questions by searching the web first.
X>
X> 1. Run `kdeps registry search web` to find a web search component. Install it with `kdeps registry install <name>`.
X> 2. Create a new workflow project. In `workflow.yaml`, declare the installed component in `components:`.
X> 3. Wire the component into your pipeline: accept a question via `POST /api/v1/search-answer`, call the search component to retrieve relevant results, then pass those results as context to a `chat:` LLM resource.
X> 4. Return both the search snippets and the LLM's answer in the response.
X>
X> Run `kdeps registry list` to verify the component appears as installed. Test the endpoint:
X> ```bash
X> curl -X POST localhost:16395/api/v1/search-answer \
X>   -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
X>   -H "Content-Type: application/json" \
X>   -d '{"q":"What was the GDP of the Netherlands in 2023?"}'
X> ```
X>
X> **Stretch goal:** Build a custom local component in `components/` that wraps a currency conversion `httpClient:` call. Use it in the same workflow to convert a dollar amount in the search results to euros.

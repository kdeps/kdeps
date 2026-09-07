# docs/v2 → focused products: migration plan

Status: proposal. Nothing moves until this is approved.

Decisions locked in:
- **One VitePress site**, per-product sidebars selected by URL prefix, product switcher in the top nav. One build, one search, one deploy.
- **6 products**: agent · workflow · agencies · llm-server · deploy · registry.
- Thin features (browser, RAG, messaging/bots, code-intelligence) are **sections inside a product**, not products.

---

## 1. URL scheme

| Prefix | Product | Landing |
|---|---|---|
| `/agent/` | kdeps agent | `/agent/` |
| `/workflow/` | kdeps workflow | `/workflow/` |
| `/agencies/` | kdeps agencies | `/agencies/` |
| `/llm-server/` | kdeps LLM server | `/llm-server/` |
| `/deploy/` | kdeps deploy | `/deploy/` |
| `/registry/` | kdeps registry | `/registry/` |
| `/start/` | shared front door + "which product?" router | `/start/` |
| `/reference/` | shared reference (expression language, CLI, glossary, security) | `/reference/` |
| `/examples/` | shared, each example tagged with its product | `/examples/` |

Site `index.md` stays the marketing home; its 6 feature cards map 1:1 to the 6 products.

---

## 2. Per-product page shape

Every `/(product)/index.md`:
1. **What it is** — one sentence.
2. **The one command** — the single entry point, runnable.
3. **60-second example** — copy-paste, produces output.
4. **When NOT to use this** — points at the sibling product.
5. **Sidebar TOC** for that product only.

---

## 3. File-move mapping (all 141)

### → `/agent/` (kdeps agent)
```
modes/agent-loop-mode.md            → agent/index.md            (rewrite intro as the product landing)
modes/agent-loop-repl.md            → agent/repl.md
modes/agent-loop-commands.md        → agent/commands.md
modes/agent-loop-tools.md           → agent/tools.md
modes/agent-loop-models.md          → agent/models.md
modes/agent-loop-skills.md          → agent/skills.md
modes/agent-loop-goals.md           → agent/goals.md
modes/agent-loop-judges.md          → agent/judges.md
modes/agent-loop-approvals.md       → agent/approvals.md
modes/agent-loop-monitoring.md      → agent/monitoring.md
modes/agent-loop-shell.md           → agent/shell.md
modes/agent-loop-turo.md            → agent/turo.md
modes/agent-loop-registries.md      → agent/registries.md       (cross-links /registry/)
getting-started/local-agent.md      → agent/quickstart.md
getting-started/agent-skills.md     → agent/ai-assisted-authoring.md
concepts/memory.md                  → agent/memory.md
concepts/memory-internals.md        → agent/memory-internals.md
```

### → `/workflow/` (kdeps workflow)
```
modes/workflow-mode.md              → workflow/index.md         (rewrite as product landing)
resources/overview.md               → workflow/resources.md
resources/api-response.md           → workflow/resources/api-response.md
resources/sql.md                    → workflow/resources/sql.md
resources/llm/index.md              → workflow/resources/llm.md
resources/llm/backends.md           → workflow/resources/llm-backends.md   (also linked from /llm-server/)
resources/llm/routing.md            → workflow/resources/llm-routing.md
resources/scripting/index.md        → workflow/resources/scripting.md
resources/scripting/exec.md         → workflow/resources/exec.md
resources/scripting/python.md       → workflow/resources/python.md
resources/files/index.md            → workflow/resources/files.md
resources/files/file.md             → workflow/resources/file.md
resources/files/git.md              → workflow/resources/git.md
resources/web/index.md              → workflow/resources/web.md
resources/web/http-client.md        → workflow/resources/http-client.md
resources/web/browser.md            → workflow/resources/browser.md
resources/web/scraper.md            → workflow/resources/scraper.md
resources/search/index.md           → workflow/resources/search.md
resources/search/searchweb.md       → workflow/resources/search-web.md
resources/search/searchlocal.md     → workflow/resources/search-local.md
resources/rag/index.md              → workflow/resources/rag.md
resources/rag/embedding.md          → workflow/resources/embedding.md
resources/rag/vector-store.md       → workflow/resources/vector-store.md
resources/rag/loader.md             → workflow/resources/loader.md
resources/media/index.md            → workflow/resources/media.md
resources/media/ocr.md              → workflow/resources/ocr.md
resources/media/transcribe.md       → workflow/resources/transcribe.md
resources/messaging/index.md        → workflow/resources/messaging.md
resources/messaging/bot-reply.md    → workflow/resources/bot-reply.md
resources/messaging/email.md        → workflow/resources/email.md
resources/messaging/telephony.md    → workflow/resources/telephony.md
resources/code-intelligence/index.md      → workflow/resources/code-intelligence.md
resources/code-intelligence/graph.md      → workflow/resources/code-graph.md
resources/code-intelligence/navigation.md → workflow/resources/code-navigation.md
concepts/expressions.md             → workflow/expressions.md
concepts/expression-helpers.md      → workflow/expression-helpers.md
concepts/jinja2-templates.md        → workflow/templates.md
concepts/unified-api.md             → workflow/data-access.md
concepts/tools.md                   → workflow/tools.md
concepts/error-handling.md          → workflow/error-handling.md
concepts/validation-and-control.md  → workflow/validation.md
concepts/items.md                   → workflow/items.md
concepts/loop.md                    → workflow/loop.md
concepts/input-sources.md           → workflow/input-sources.md
concepts/inline-resources.md        → workflow/inline-resources.md
configuration/workflow.md           → workflow/configuration.md
configuration/session.md            → workflow/sessions.md
configuration/cors.md               → workflow/cors.md
configuration/route-restrictions.md → workflow/route-restrictions.md
getting-started/quickstart.md       → workflow/quickstart.md
getting-started/workflow-as-tool.md → workflow/as-a-tool.md
guides/execution-flow.md            → workflow/execution-flow.md
```

### → `/agencies/` (kdeps agencies)
```
concepts/agency.md            → agencies/index.md          (rewrite as landing)
concepts/components.md        → agencies/components.md
resources/delegation/index.md → agencies/delegation.md
resources/delegation/agent.md → agencies/agent-resource.md
resources/delegation/component.md → agencies/component-resource.md
reference/components.md       → agencies/component-reference.md
```

### → `/llm-server/` (kdeps LLM server)
```
deployment/llm-server.md          → llm-server/index.md      (rewrite as landing)
reference/cli/llm.md              → llm-server/cli.md
reference/llm-providers.md        → llm-server/providers.md
reference/llm-providers-m365.md   → llm-server/m365.md
```
(`workflow/resources/llm-backends.md` is cross-linked here; the m365 `proxy` subcommand doc lives here.)

### → `/deploy/` (kdeps deploy)
```
deployment/docker.md          → deploy/docker.md
deployment/kubernetes.md      → deploy/kubernetes.md
deployment/prepackage.md      → deploy/prepackage.md
deployment/tls-https.md       → deploy/tls-https.md
deployment/webserver.md       → deploy/webserver.md
guides/deployment-guide.md    → deploy/index.md            (rewrite as landing)
reference/docker-reference.md → deploy/docker-reference.md
reference/cli/packaging.md    → deploy/cli.md
```

### → `/registry/` (kdeps registry)
```
reference/cli/registry.md         → registry/cli.md
reference/registry-formula-spec.md → registry/formula-spec.md
```
New: `registry/index.md` (landing — what the registry is, `kdeps registry search/install/publish`, kdeps.io link).

### → `/start/` (shared front door)
```
getting-started/introduction.md → start/index.md           (+ a "which product do I need?" decision table)
getting-started/installation.md → start/installation.md
getting-started/local-models.md → start/local-models.md
concepts/why-kdeps.md           → start/why-kdeps.md
concepts/overview.md            → start/concepts.md
```

### → `/reference/` (shared, stays)
```
reference/cli/index.md                       → reference/cli.md
reference/cli/dev.md                          → reference/cli-dev.md
reference/glossary.md                         → reference/glossary.md        (unchanged)
reference/security.md                         → reference/security.md        (unchanged)
reference/management-api.md                   → reference/management-api.md  (unchanged)
reference/expression-functions-reference.md   → reference/expression-functions.md
reference/expression-operators.md             → reference/expression-operators.md
reference/expr-blocks.md                      → reference/expr-blocks.md
reference/items-reference.md                  → reference/items.md
reference/tools-reference.md                  → reference/tools.md
reference/browser-actions.md                  → reference/browser-actions.md
reference/validation-examples.md              → reference/validation-examples.md
reference/http-client-examples.md             → reference/http-client-examples.md
reference/python-examples.md                  → reference/python-examples.md
reference/sql-examples.md                     → reference/sql-examples.md
guides/faq.md                                 → reference/faq.md
guides/troubleshooting.md                     → reference/troubleshooting.md
configuration/advanced.md                     → reference/advanced-config.md
CONTRIBUTING-DOCS.md                           → reference/contributing-docs.md
STYLE-GUIDE.md                                 → reference/style-guide.md
```

### `/examples/` (stays; add a `product:` front-matter tag to each)
All 27 `examples/**` files stay in place. `examples/index.md` gains a per-product filter/grouping. Each example's front matter gets `product: workflow|agent|agencies`.

---

## 4. Redirects

VitePress supports `rewrites` (build-time) but not runtime redirects. Two options:
- **A.** A `public/_redirects` file (Netlify/Cloudflare Pages) mapping every old path → new. ~120 lines, generated from section 3.
- **B.** Leave a 1-line stub `.md` at each old path with front-matter `redirect: /new/path` handled by a tiny theme hook.

Recommendation: **A** if the docs host supports `_redirects`; otherwise B for the top ~30 most-linked pages and accept 404s on the long tail (internal links are all updated regardless).

Every internal `](/old/path)` link across all 141 files + `README.md` + `book/` is rewritten as part of the move (scripted).

---

## 5. `config.ts`

```ts
nav: [
  { text: 'Agent',      link: '/agent/' },
  { text: 'Workflow',   link: '/workflow/' },
  { text: 'Agencies',   link: '/agencies/' },
  { text: 'LLM server', link: '/llm-server/' },
  { text: 'Deploy',     link: '/deploy/' },
  { text: 'Registry',   link: '/registry/' },
  { text: 'Reference',  link: '/reference/cli' },
]

sidebar: {
  '/agent/':      [ ...agentSidebar ],
  '/workflow/':   [ ...workflowSidebar ],
  '/agencies/':   [ ...agenciesSidebar ],
  '/llm-server/': [ ...llmServerSidebar ],
  '/deploy/':     [ ...deploySidebar ],
  '/registry/':   [ ...registrySidebar ],
  '/reference/':  [ ...referenceSidebar ],
  '/start/':      [ ...startSidebar ],
  '/examples/':   [ ...examplesSidebar ],
}
```
The current single ~200-line sidebar object is deleted and replaced by 9 smaller focused ones.

---

## 6. README + book

- **README.md**: the "three levels" list becomes the 6-product list; each bullet links to its `/(product)/` landing.
- **book/**: `Book.txt` chapter order re-grouped into Parts — Part I Agent, Part II Workflow, Part III Agencies, Part IV LLM server, Part V Deploy, Part VI Registry, Appendices. Chapter *content* is already close; this is mostly re-sectioning + updating cross-references. Do this in the same PR series so `book/` never drifts.

---

## 7. Phased PRs

1. **Scaffold + shared** — create `/start/`, `/reference/` moves, `config.ts` multi-sidebar skeleton, redirect infra. No product content yet. Site still builds.
2. **kdeps agent** — migrate `/agent/`, write its landing, wire its sidebar. Reference implementation for the pattern.
3. **kdeps workflow** — the big one (~55 files). Split into 3a resources, 3b concepts, 3c config+quickstart if needed.
4. **kdeps agencies**
5. **kdeps LLM server**
6. **kdeps deploy**
7. **kdeps registry** (+ new landing)
8. **examples re-tag + `examples/index.md` regroup**, README, book/ Parts, final link sweep, remove this plan file.

Each PR: `npm run build` green (no dead links), `book/` synced, redirect list extended.

---

## 8. Open risks

- **Deep-link churn**: kdeps.io, blog posts, the `kdeps` skill, and Claude/Cursor prompt caches point at `/modes/agent-loop-*` etc. Redirects + a grace period mitigate; the skill's doc-URL references must be updated in lockstep (`private/` skill repo).
- **`resources/llm/backends.md`** genuinely serves two products (workflow config + llm-server). It lives under `/workflow/` with a prominent cross-link from `/llm-server/`; do not duplicate.
- **`examples/`** not being a product means the nav has 6 products + Reference + a slightly orphaned Examples. Acceptable — Examples is a cross-cutting gallery by design.

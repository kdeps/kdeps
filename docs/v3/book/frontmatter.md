# AI Appliances: Build & Deploy Autonomous AI Agents and Agencies in YAML

**By Joel Bryan Juliano**

---

*For every developer who has shipped a "prototype" that somehow made it to production.*

---

## About This Book

This book is a practical guide to building and deploying production-ready AI agents using **kdeps** — an open-source, YAML-based framework that lets you define AI workflows as self-contained appliances and deploy them anywhere.

You will learn how to build deterministic AI pipelines, wire them into autonomous agent loops, compose multi-agent systems (agencies), and ship them as Docker images, Kubernetes workloads, standalone binaries, or edge-deployable ISOs — all without touching a proprietary cloud platform or rewriting anything when you switch LLM providers.

**What you need to follow along:**

- Familiarity with YAML
- Basic command-line comfort (bash, curl, docker)
- No prior AI/ML experience required — but knowing what an LLM API response looks like helps

**What this book is not:**

This is not a machine learning theory book. It is not a tutorial on prompt engineering as a discipline. It is a book about running AI in production as reliably as you run a database.

---

## How This Book Is Organized

**Part I — Foundation** (Chapters 1–3) explains what kdeps is, why it exists, and how all the pieces fit together conceptually.

**Part II — Building Agents** (Chapters 4–11) goes deep on workflow mode, agent mode, every resource type, and the expression language that connects them.

**Part III — Multi-Agent Systems** (Chapters 12–13) covers components (reusable resource bundles) and agencies (multi-agent orchestration).

**Part IV — Configuration & Operations** (Chapters 14–16) covers the full `workflow.yaml` reference, sessions, CORS, route restrictions, and advanced server settings.

**Part V — Deployment** (Chapters 17–20, 26) is end-to-end deployment: Docker, Kubernetes, standalone binaries, web frontends, and standalone LLM server appliances (`kdeps llm`) that any kdeps client can use over OpenAI-compatible `/v1`.

**Part VI — Going Further** (Chapters 21–25) covers the validate/debug toolchain, iteration with `items:` and `loop:`, error handling with `onError:`, real-world examples, and native bot/file input sources.

**Appendices** cover troubleshooting common failure modes, security hardening, and automated testing strategies.

---

## Code Conventions

YAML is whitespace-sensitive. Every code block in this book has been tested. File paths shown in comments (e.g., `# resources/llm.yaml`) reflect the directory structure inside a kdeps project. When you see `&#123;&#123; get('q') &#125;&#125;` inside a YAML string, the double braces are kdeps expression interpolation — not a template engine artifact.

Commands prefixed with `$` are run in your terminal. Commands without a prefix are shown inline for clarity.

# Why kdeps?

One `workflow.yaml` runs as an API, a bot, a file job, or a prompt. A `chat` with `loop:` is the step where the model picks tools. You pay for the hardware it runs on.

## The same file

One `workflow.yaml` is the pipeline you deploy. `kdeps --agent-loop` opens a `chat` with `loop:` as a REPL, on a local model, with no Docker. The registry is optional. It shares agents by name. See [one workflow, any input](#one-workflow-any-input). The [where to read next](/start/#where-to-read-next) table names the other docs.

---

## Deterministic by design

An LLM is probabilistic: the same question twice can get two answers. The workflow puts that model in a fixed shell.

The same request always goes through the same resources. `requires:` sets the order. Validations run before a model call. The output follows a fixed schema. The text that the model writes can be different each time. But the steps, their order, and the shape of the output do not change.

If the input is wrong, the workflow stops quickly with a clear error. It does not hallucinate a response. The structure of the output is reproducible and auditable, and the workflow can run safely without supervision. For these reasons, kdeps is applicable to production, not only to demos.

Inside `loop:`, the model picks tools. `requires:` still sets the order of the steps around that chat.

## Git-native

**The YAML is the behavior specification.** The `workflow.yaml` and
`resources/*.yaml` files in your repository fully define what your appliance
does. They set the model that it calls, the data that it validates, the steps
and their order, and the shape of the response. There is no database of agent
state, no proprietary project file, and no web console that holds the source of
truth. When you change the YAML and commit it, the appliance changes its
behavior. Your git history is the changelog of the agent behavior.

Only the backend binding is *outside* the repository. The backend binding is
the LLM backend, model names, API keys, and database connections in
`~/.kdeps/config.yaml` and environment variables. This is intentional. The same
repository runs with a local llamafile on your laptop and with a cloud model in
production, with no diff. The agent logic is in git. Only its connection to a
specific model is not in git.

The core loop needs nothing but git and the `kdeps` binary: edit YAML, commit,
`kdeps run` or `kdeps bundle build`. Everything below builds on that.

## Review, version, build, share

- **Review** an agent change the way you review code - a diff of resources,
  validations, and prompts in a pull request.
- **Version** with a git tag. A tag is a release. `kdeps validate` and
  `kdeps bundle build` run against any checkout or tag.
- **Build reproducibly** in CI: `kdeps validate` -> `kdeps bundle build` ->
  `docker push`, triggered on tag. The same commit yields the same appliance,
  runtime and model pinned.
- **Share (optional).** The [kdeps registry](/registry/) lets you install and
  publish agents and components by name. `kdeps install owner/repo` pulls
  directly from a GitHub repository, with no registry entry. You do not need
  the registry to build or deploy your own agent.

The [`git` component](/workflow/resources/git) also lets an agent read and write
repository state as part of its pipeline - status, diff, log, commit, branch.

## One workflow, any input

```d2
direction: right

api: "api\nHTTP request" {shape: oval}
bot: "bot\nDiscord, Slack, Telegram" {shape: oval}
file: "file\nstdin or path" {shape: oval}
prompt: "interactive\nterminal prompt" {shape: oval}
W: "workflow.yaml\nDAG, deterministic order"
L: "chat with loop:\nagent: tools, memory"

api -> W
bot -> W
file -> W
prompt -> W
W -> L: "when a step is an agent"
```

kdeps validates the inputs, runs the resources in a fixed order, and gives predictable and auditable output. A step that needs judgment is a `chat` with `loop:`: the model calls tools over many rounds inside that one step. `kdeps --agent-loop` opens the same agent as a REPL.

Same `workflow.yaml`, no rewrites, no re-config.

## Agencies

kdeps [agencies](/reference/glossary#agency) let you compose multiple specialized agents into a single system. Each agent has its own model, resources, and logic. They communicate through the `agent:` resource type. This type runs the full pipeline of another agent and returns its output. Each step is version-controlled and testable, and you can deploy it independently.

## Built to last

Model APIs deprecate and SDK interfaces change. kdeps reduces how many of those moving parts can break a running workload. The thing you archive is the built appliance - the Docker image, ISO, or self-contained binary you produce with `kdeps bundle`. It pins the kdeps runtime, the executors, and (for local models) the model itself into one frozen unit. Redeploying that image later runs the same runtime and model, with no live dependency on a vendor API.

## What stays stable

What stays stable:

- **Local LLMs do not break underneath you.** With Ollama or any self-hosted model, the interface changes only when you update it. No vendor deprecation notices, no sunset dates.
- **The backend is separate from the workflow.** Cloud model names are in `~/.kdeps/config.yaml`, not in `workflow.yaml`. When a provider removes a model, you change one line in the configuration. The workflow does not change.
- **Your logic is code you own.** SQL, LLM prompts, Python, exec, email, inter-agent calls - these change when you change them.

## What you should expect to maintain

What you must maintain:

| Thing | Why it moves | What to do |
|---|---|---|
| The `workflow.yaml` schema | kdeps does not carry legacy schema shims - a newer `kdeps` binary may reject an older file. `apiVersion: kdeps.io/v1` marks the current schema, not a frozen one. | Keep the source YAML with the image; re-validate with `kdeps validate` before you upgrade the runtime, or continue to run the built image. |
| `http_request` targets | External APIs change schema, auth, endpoints | Target a versioned path (`/v2/users`, not `/users`) |

The guarantee is not that your YAML runs forever on any future kdeps. It is that the appliance you ship is a self-contained unit you can freeze, archive, and redeploy without a live dependency on any vendor.

## Who it is for

| Role | Use case |
|------|----------|
| Developers | Ship AI features into products (APIs, bots, internal tools) without glue code |
| Operations teams | Automate repetitive work: log analysis, PR reviews, triage, ticket creation, incident messaging |
| SMEs without an AI hire | Add AI to existing business processes and APIs with near-zero code and no recurring AI bill |
| Marketing and growth | Content pipelines, SEO automation, campaign reporting |
| Regulated and public-sector teams | Meet data-residency rules. Run your own LLM and the workflow in the region you choose |
| Any team | Replace a human clicking through tabs and copy-pasting between tools |

Examples: log-file analysis, automated code review, a JIRA ticket from an alert, an incident summary posted to MS Teams. Each is a workflow calling a model and one or two APIs, deployed as one image.

## The name

kdeps is short for knowledge dependencies. It grew out of [Kartographer](https://github.com/kdeps/kartographer), a graph library that resolves dependent nodes: you can orchestrate knowledge from a model, machine, or person as a graph. A kdeps workflow is that graph of resources, run in order.

## Built on

kdeps uses [Kartographer](https://github.com/kdeps/kartographer), a graph library that resolves dependent nodes. You can use it on its own.

| Project | What it does | Where kdeps uses it |
|---|---|---|
| [kdeps/kartographer](https://github.com/kdeps/kartographer) | A graph library that resolves dependent nodes. | The link graph for `search_local` `graphBoost` and for skill-library linking. |

## See also

- [Quickstart](/workflow/quickstart) - build your first workflow API
- [Run locally](/agent/quickstart) - agent REPL in 30 seconds
- [Data sovereignty](/start/data-sovereignty) - your prompts stay on your servers
- [Load a workflow as a tool](/workflow/as-a-tool) - the same file, called by the model
- [Workflow](/workflow/) - the YAML pipeline
- [Agent REPL](/agent/) - a chat with `loop:`

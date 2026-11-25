# Agent loop reference

This page lists every field and option. The [overview](/workflow/agent-loop/) shows what it does and a first example.


The agent loop is a workflow `chat` with [`loop:`](/reference/resources/llm#agent-loop). It calls tools over many rounds until the model gives a final answer. `kdeps run` runs it like any other resource. The default model is local, so it needs no API key and runs offline.

```text
input -> chat (loop:) -> tool rounds -> answer -> next resource
```

## Write one

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: assistant
  description: An agent that works on the files of this folder
  version: "1.0.0"
  targetActionId: assistant
deploy:
  input:
    sources: [interactive]   # kdeps run opens a prompt; each line is one turn
```

```yaml
# resources/assistant.yaml
actionId: assistant
name: Assistant
chat:
  model:
    name: ministral3                   # local 3B llamafile; a 1B model does not call tools
  context:
    prompt: "{{ input('message') }}"
  loop:
    tools: [read_file, edit_file, list_files, search_local, bash_exec]   # empty offers every built-in tool
    maxRounds: 25                      # tool rounds per request
    session: "{{ get('session') }}"    # one stored conversation across requests
```

Run `kdeps new my-agent --template assistant` to get these two files. Then run it:

```bash
cd my-agent
kdeps run workflow.yaml   # or bare: kdeps
```

The same chat serves an HTTP API, a bot, or a file when you change `sources:`. See [Input sources](/workflow/input-sources).

## What the loop adds

| Topic | Page |
|-------|------|
| Built-in tool catalog and permission modes | [Built-in tools](/workflow/agent-loop/tools) |
| Tool calls that a model writes as text | [Fenced tools](/workflow/agent-loop/fenced-tools) |
| Shell execution and background jobs | [Shell execution](/reference/agent-loop/tool-reference#shell-execution) |
| The `subagent` tool | [Subagents](/workflow/agent-loop/subagent) |
| Independent review of the answer | [Judge panel](/workflow/agent-loop/judges) |
| One-time exceptions for a denied tool call | [Approval tokens](/workflow/agent-loop/approvals) |
| Persistent memory | [Persistent memory](/workflow/agent-loop/memory) |
| Skills and `KDEPS.md` instructions | [Skills and instructions](/workflow/agent-loop/skills) |
| Limits and thresholds as YAML | [Reactive LLM events](/workflow/agent-loop/events) |

Every key is under the chat in the workflow. The full list is in the [chat resource reference](/reference/resources/llm#agent-loop).

## See also

- [Workflow](/workflow/) - the YAML pipeline
- [Input sources](/workflow/input-sources) - API, bot, file, or interactive prompt
- [LLM provider reference](/reference/llm-server/providers) - backend config and model names
- [Agencies](/agencies/) - several workflows as one system

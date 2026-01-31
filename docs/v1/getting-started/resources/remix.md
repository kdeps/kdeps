---
outline: deep
---

# Reusing and Remixing AI Agents

KDeps makes it easy to integrate existing AI agents, enabling reusing and remixing of pre-built AI agents
into your own AI workflows.

## Installing a KDeps AI Agent

To begin, install the AI agent using the `kdeps install` command:

```bash
kdeps install conveyour_counting_ai-1.2.5.kdeps
```

Once installed, the agent is ready to be registered in your workflow.

## Registering AI Agents in Your Workflow

After installation, you must register the AI agent in your `workflow.pkl` file. External workflows are referenced using
`@` followed by the agent name.

For example, to include the latest version of the AI agent in your workflow:

```apl
Workflows {
  "@conveyour_counting_ai"
}
```

This will include to all the resources provided by the `conveyour_counting_ai` agent. If you prefer a specific
version of the agent, include the `:version` specifier, like this:


```apl
Workflows {
  "@conveyour_counting_ai:1.2.5"
}
```

## Utilizing an External AI Agent

Once the agent is registered in your `workflow.pkl` file, you can include it in the `requires` block of your resources:

```apl
Requires {
  "@conveyour_counting_ai/countImageLLM:1.2.5"
  "@conveyour_counting_ai/sortImageItemsLLM:1.2.5"
}
```

After specifying the required resources, you can use a `function` or retrieve output through `file`. Here’s an example:

```apl
local sortedItemsJsonPath = "@(llm.file("@conveyour_counting_ai/sortImageItemsLLM:1.2.5"))"
local sortedItemsJsonData = "@(read?("\(sortedItemsJsonPath)")?.text)"

local report = new Mapping {
  ["fruit_count"] = "@(document.JSONParser(sortedItemsJsonData)?.fruit_count_integer)"
  ["vegetable_count"] = "@(document.JSONParser(sortedItemsJsonData)?.vegetable_count_integer)"
  ["stock_analysis_report"] = "@(document.JSONParser(stockAnalysisLLMReporter)?.report_markdown)"
}
```

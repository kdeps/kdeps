---
outline: deep
---

# Graph Dependency

Kdeps utilizes its custom graph library, [Kartographer](https://github.com/kdeps/kartographer), to manage resource
dependencies. Kartographer enables the traversal and resolution of dependent nodes, allowing Kdeps to
orchestrate the execution order of resources.

This capability makes Kdeps particularly well-suited for building context-aware RAG (Retrieval-Augmented Generation) AI
agents that require chaining large language models (LLMs) and other components. It also allows reusing and remixing
other AI agents.

### Defining Dependencies

To construct a dependency graph, you must define resource dependencies in the resource's `requires`
configuration. Additionally, the target node should be specified in the workflow's `targetActionID` parameter.

Here’s an example of how to define a resource’s dependencies using `requires`:

```apl
requires {
    "resourceID1"
    "resourceID2"
    "resourceID3"
}
```

### Understanding Graph Dependencies

When your workflow's `targetActionID` is set to a target, say `JSONResponder`, and your resources have defined dependencies.

Kartographer processes the graph by executing:

1. All the dependent nodes first.
2. Finally, the specified target node, such as `JSONResponder`.

For instance, given the following dependency chain:

`LLMResourceJSON -> PythonResource -> JSONResponder`

Kartographer ensures that each node is executed in sequential order. However, by design, a resource ID cannot be reused
within the same workflow. This avoids complex dependency problems such as circular dependencies. If you need to reuse a
resource, it should be under a unique ID, as shown below:

`LLMResourceJSON -> PythonResource -> LLMResourceJSON2 -> JSONResponder`

> *Note:*
> Kdeps executes resource in a top-down queue manner. By design, Kdeps does not allow multiple resource actions to be
> executed in a single resource file. If you need to perform a new resource action, you have to create a new resource
> file with a unique ID, then define it as a dependency.

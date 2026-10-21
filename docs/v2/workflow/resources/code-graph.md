# Graphing an indexed folder

The `code_index_folder`/`code_graph_file`/`code_graph_topic`/`code_graph_all` components are different from the LSP/rg operations of [Code intelligence](/workflow/resources/code-navigation): instead of searching live files, they build and query a small persistent graph database of the current working directory - tracking which files link to which, and which files share a topic. Run kdeps from inside a `docs/` directory (or a source tree, with `extensions` opted in - see below) and ask "what references this file?" or "show me everything tagged `auth`."

They work in both modes: as DAG steps in workflow mode, and as built-in capabilities in the agent loop.

References are derived per file type:

| File type | How references are found |
|-----------|---------------------------|
| `.md` / `.markdown` / `.txt` | Markdown links (`[text](path)`) and wikilinks (`[[path]]`) |
| `.html` / `.htm` | `href`/`src` attribute values |
| Go, Python, Rust, TypeScript/JavaScript, C/C++, Ruby, Java | Import/include statements - relative (`from . import x`, `#include "local.h"`) and non-relative/module-style (`import "github.com/x/pkg"`), the latter matched against files actually present in the indexed folder. Heuristic regex extraction, not a full parser - ambiguous or external/stdlib imports are dropped, not guessed. |
| `.json` / `.yaml` / `.yml` (bare, no frontmatter) | No reference extraction |

All of the above (plus any file with a leading `---` YAML frontmatter block) get topic extraction from a `topics:`/`tags:` list - for a bare `.json`/`.yaml`/`.yml` file, a top-level `topics:`/`tags:` key works with no frontmatter markers needed.

The default `extensions` (`.md`/`.markdown`/`.txt`/`.yaml`/`.yml`) only cover docs - indexing source code, HTML, or JSON requires opting in explicitly, so pointing the index at a project root never silently walks an entire source tree by surprise:

```yaml
component:
  name: code_index_folder
  with:
    extensions: [".go", ".md"]  # opt in to Go import graphing alongside docs
```

`code_index_folder` must run once before `code_graph_file`/`code_graph_topic`/`code_graph_all` can query the same graph database (same `graphDBPath`, which defaults to `~/.kdeps/projects/<folder>-<hash>/graph.db` for the working folder; nothing is written into the folder itself).

```text
code_index_folder -> walks the CWD, extracts links + frontmatter topics
                  -> writes them into ~/.kdeps/projects/<folder>-<hash>/graph.db
                       -> code_graph_file(path)   returns that file's references + topic-related files
                       -> code_graph_topic(topic) returns every file tagged `topic` + the full reference graph
                       -> code_graph_all()        returns the full reference graph + every root file
```

**Index the current working directory:**

```yaml
component:
  name: code_index_folder
  with:
    extensions: [".md", ".yaml"]  # optional; defaults to .md/.markdown/.txt/.yaml/.yml
```

`path` is not accepted here - see the note above. Run `kdeps run` (or the `code_index_folder` agent tool) from inside the directory you want indexed.

**Output:**
```json
{
  "success": true,
  "path": "/current/working/directory",
  "filesIndexed": 42
}
```

**Graph a single file** - its references, plus every other indexed file sharing a topic with it:

```yaml
component:
  name: code_graph_file
  with:
    path: /path/to/docs/intro.md
```

**Output:**
```json
{
  "success": true,
  "path": "/path/to/docs/intro.md",
  "references": {
    "/path/to/docs/intro.md": ["/path/to/docs/setup.md"],
    "/path/to/docs/setup.md": []
  },
  "relatedByTopic": ["/path/to/docs/getting-started.md"]
}
```

**Graph everything tagged with a topic:**

```yaml
component:
  name: code_graph_topic
  with:
    topic: getting-started
    # path: /path/to/docs  # optional: locate the graph db elsewhere; defaults to ~/.kdeps/projects/<folder>-<hash>/graph.db
```

**Graph the whole index** - the full reference graph plus every root file (files nothing else references):

```yaml
component:
  name: code_graph_all
```

Each component returns the tool's JSON body as text. Read fields with `fromJSON`:

```yaml
# resources/respond.yaml
actionId: respond
name: Respond
requires: [graphAll]
apiResponse:
  success: true
  response:
    roots: "{{ fromJSON(output('graphAll')).roots }}"
    references: "{{ fromJSON(output('graphAll')).references }}"
```

`references` is always a full adjacency map (`file -> files it links to`); the caller walks it to render a tree or diagram. This mirrors [kartographer](https://github.com/kdeps/kartographer)'s own `DependencyGraph` traversal machinery, reused here through `IndexedGraph`.

## See also

- [Code intelligence](/workflow/resources/code-navigation) - the other LSP/rg-backed operations

# Graphing an Indexed Folder

The `indexFolder`/`graphFile`/`graphTopic`/`graphAll` operations of the [Code Intelligence resource](/resources/codeintelligence) are different from its LSP/rg operations: instead of searching live files, they build and query a small persistent graph database of the current working directory — tracking which files link to which, and which files share a topic. Run kdeps from inside a `docs/` directory (or a source tree, with `extensions` opted in — see below) and ask "what references this file?" or "show me everything tagged `auth`."

References are derived per file type:

| File type | How references are found |
|-----------|---------------------------|
| `.md` / `.markdown` / `.txt` | Markdown links (`[text](path)`) and wikilinks (`[[path]]`) |
| `.html` / `.htm` | `href`/`src` attribute values |
| Go, Python, Rust, TypeScript/JavaScript, C/C++, Ruby, Java | Import/include statements — relative (`from . import x`, `#include "local.h"`) and non-relative/module-style (`import "github.com/x/pkg"`), the latter matched against files actually present in the indexed folder. Heuristic regex extraction, not a full parser — ambiguous or external/stdlib imports are dropped, not guessed. |
| `.json` / `.yaml` / `.yml` (bare, no frontmatter) | No reference extraction |

All of the above (plus any file with a leading `---` YAML frontmatter block) get **topic** extraction from a `topics:`/`tags:` list — for a bare `.json`/`.yaml`/`.yml` file, a top-level `topics:`/`tags:` key works with no frontmatter markers needed.

The default `extensions` (`.md`/`.markdown`/`.txt`/`.yaml`/`.yml`) only cover docs — indexing source code, HTML, or JSON requires opting in explicitly, so pointing `indexFolder` at a project root never silently walks an entire source tree by surprise:

```yaml
codeIntelligence:
  operation: indexFolder
  extensions: [".go", ".md"]  # opt in to Go import graphing alongside docs
```

`indexFolder` must run once before `graphFile`/`graphTopic`/`graphAll` can query the same graph database (same `graphDBPath`, which defaults to `<CWD>/.kdeps/graph.db`).

```text
indexFolder() -> walks the CWD, extracts links + frontmatter topics
              -> writes them into <CWD>/.kdeps/graph.db
                   -> graphFile(path)   returns that file's references + topic-related files
                   -> graphTopic(topic) returns every file tagged `topic` + the full reference graph
                   -> graphAll()        returns the full reference graph + every root file
```

**Index the current working directory:**

```yaml
codeIntelligence:
  operation: indexFolder
  extensions: [".md", ".yaml"]  # optional; defaults to .md/.markdown/.txt/.yaml/.yml
```

`path` is not accepted here — see the note above. Run `kdeps run` (or the `code_index_folder` agent tool) from inside the directory you want indexed.

**Output:**
```json
{
  "success": true,
  "path": "/current/working/directory",
  "filesIndexed": 42
}
```

**Graph a single file** — its references, plus every other indexed file sharing a topic with it:

```yaml
codeIntelligence:
  operation: graphFile
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
codeIntelligence:
  operation: graphTopic
  topic: getting-started
  # path: /path/to/docs  # optional: locate the graph db elsewhere; defaults to <CWD>/.kdeps/graph.db
```

**Graph the whole index** — the full reference graph plus every root file (files nothing else references):

```yaml
codeIntelligence:
  operation: graphAll
```

`references` is always a full adjacency map (`file -> files it links to`); the caller walks it to render a tree or diagram. This mirrors [kartographer](https://github.com/kdeps/kartographer)'s own `DependencyGraph` traversal machinery, reused here through `IndexedGraph`.

## See also

- [Code Intelligence Resource](/resources/codeintelligence) -- the other LSP/rg-backed operations

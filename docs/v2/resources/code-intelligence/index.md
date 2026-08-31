# Code intelligence

Navigate and map a codebase. The `codeIntelligence:` resource has two families
of operations, each with its own reference page.

*Applies to both workflow mode and agent mode.*

| Operations | What they do | Reference |
| :--- | :--- | :--- |
| `symbolSearch`, `definition`, `references`, `hover`, `diagnostics` | Live code navigation via a language server, falling back to ripgrep | [Code navigation](/resources/code-intelligence/navigation) |
| `indexFolder`, `graphFile`, `graphTopic`, `graphAll` | Build and query a persistent graph of which files link to which and share a topic | [Folder graph](/resources/code-intelligence/graph) |

## See also

- [Search resources](/resources/search/) - filename and content search over files
- [File resources](/resources/files/) - read and edit source files
- [Resources overview](/resources/overview) - all resource types

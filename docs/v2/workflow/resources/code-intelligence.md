# Code intelligence

Navigate and map a codebase. The `codeIntelligence:` resource has two families
of operations, each with its own reference page.

*Applies to both workflow mode and agent mode.*

| Operations | What they do | Reference |
| :--- | :--- | :--- |
| `symbolSearch`, `definition`, `references`, `hover`, `diagnostics` | Live code navigation via a language server, falling back to ripgrep | [Code navigation](/workflow/resources/code-navigation) |
| `indexFolder`, `graphFile`, `graphTopic`, `graphAll` | Build and query a persistent graph of which files link to which and share a topic | [Folder graph](/workflow/resources/code-graph) |

## See also

- [Search resources](/workflow/resources/search) - filename and content search over files
- [File resources](/workflow/resources/files) - read and edit source files
- [Resources overview](/workflow/resources) - all resource types

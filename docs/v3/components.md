# Components

Reusable packs from the registry. Install once, call from a resource.

## Install

```bash
kdeps registry search scraper
kdeps registry install <name>
```

Browse the public catalog at [kdeps.io](https://kdeps.io).

## Use in a resource

```yaml
actionId: reply
component:
  name: botreply
  with:
    platform: telegram
    message: "Hello!"
```

`with:` maps to the component's declared inputs. Outputs are available to later steps via `get('reply')` (your `actionId`).

## vs built-in resources

| Built-in (`chat`, `httpClient`, …) | Component |
|------------------------------------|-----------|
| Shipped in the kdeps binary | Installed from registry |
| Stable core actions | Optional, versioned packs |
| Always available | Must `registry install` on the host / image |

In **agent mode**, installed components can also surface as tools depending on how the project is loaded — prefer wrapping non-trivial packs in a small workflow if you need a hard contract.

## Agency note

Multi-agent setups use `agent:` resources and agency manifests so one workflow can call another. Treat each agent like a function with a clear `apiResponse`. Keep shared secrets in machine config, not in the agency YAML.

# Example components (E2E fixtures)

These are example components used **only by the end-to-end tests** — they are
not shipped as built-ins with kdeps. `tests/e2e/common.sh` points
`KDEPS_COMPONENT_DIR` here so tests can call `component: { name: <x> }` without a
global install.

To use a component in a real project, install it from the registry, e.g.:

```bash
kdeps registry install kdeps-scraper-example
```

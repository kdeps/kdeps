# Build a reusable component

Build a custom component - a bundle of resources with a
typed input interface - and call it from a workflow. Components are how you
package reusable logic and share it across projects.

The workflow calls the component with inputs:

<div v-pre>

```yaml
# resources/greet.yaml
actionId: greet
name: Greet
component:
  name: formatter          # the component's metadata.name
  with:
    name: "{{ get('who', 'World') }}"
    shout: "{{ get('loud', 'false') }}"
```

</div>

## Read next

- [Full walkthrough](/reference/examples/custom-component) - every step, file and command
- [Components](/agencies/components) - registry components, `tools.components`
- [Two-agent agency tutorial](/examples/agency) - composing whole agents instead

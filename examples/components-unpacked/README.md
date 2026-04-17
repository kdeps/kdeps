# Components Unpacked Example

This example demonstrates using **unpacked component directories** in the `components/` folder, alongside `.komponent` archives.

## Structure

```
components-unpacked/
├── workflow.yaml
├── resources/
│   └── response.yaml
└── components/
    └── greeter/           # Unpacked component directory
        └── component.yaml
```

## How It Works

1. **Component Directory**: The `components/greeter/` directory contains a standard `component.yaml` file (and optionally a `resources/` subdirectory).

2. **Auto-Loading**: When parsing the workflow, `loadComponents()` scans the `components/` directory and:
   - Processes each subdirectory that contains a `component.yaml` (or .j2/.yml variants)
   - Loads the component and merges its resources into the workflow
   - Deduplicates by `actionId` (later entries win)

3. **Component Resource**: The `greeter` component defines a resource `sayHello` that sets a `greeting` variable.

4. **Workflow Flow**:
   - `sayHello` (from unpacked component) → sets `greeting = "Hello from unpacked component!"`
   - `finalResponse` → returns `greeting` as JSON

## Running the Example

```bash
kdeps run examples/components-unpacked/workflow.yaml
```

Expected output:
```json
{
  "success": true,
  "message": "Hello from unpacked component!"
}
```

## Unpacked Components vs .komponent Archives

| Feature | Unpacked Directory | .komponent Archive |
|---------|-------------------|-------------------|
| Development | Easy to edit, version control friendly | Single file, portable |
| Distribution | Multiple files | Single archive file |
| Use Case | Local development, debugging | Sharing, versioning, CI/CD |
| Loading | Direct from filesystem | Extracted to temp dir automatically |

Both formats are fully supported and can coexist in the same `components/` directory. Use unpacked directories during development for easy iteration, then package as `.komponent` for distribution.

## Creating an Unpacked Component

Simply create a subdirectory under `components/` with a `component.yaml`:

```bash
mkdir components/my-component
cat > components/my-component/component.yaml << 'EOF'
apiVersion: kdeps.io/v1
kind: Component
metadata:
  name: my-component
resources:
  - apiVersion: kdeps.io/v1
    kind: Resource
    metadata:
      actionId: myAction
    run:
      expr:
        - set('result', 'done')
EOF
```

The component's resources will be automatically available to your workflow.

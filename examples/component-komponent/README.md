# Component with .komponent Auto-Loading

This example demonstrates how `.komponent` archives can be automatically loaded from the `components/` directory.

## Structure

```
component-komponent/
├── workflow.yaml
├── resources/
│   └── response.yaml  # Final API response
└── components/
    └── greeter.komponent  # Pre-packaged component
```

## How It Works

1. **Component Package**: The `greeter-1.0.0.komponent` file is a gzipped tar archive containing a `component.yaml` and its resources. It is **not** unpacked; it stays as an archive in the `components/` directory.

2. **Auto-Loading**: When the workflow is parsed, the `loadComponents()` function:
   - Scans the `components/` directory
   - Detects `.komponent` archives
   - Extracts them to temporary directories
   - Processes the component as if it were an unpacked directory
   - Merges the component's resources into the workflow

3. **Component Resources**: The `greeter` component provides a `sayHello` resource that:
   - Creates a greeting and sets the `greeting` variable

4. **Workflow Flow**:
   - `sayHello` (from component) → sets `greeting = "Hello from .komponent!"`
   - `finalResponse` → returns `greeting` as JSON

## Running the Example

```bash
# Parse and execute the workflow (no external dependencies)
kdeps run examples/component-komponent/workflow.yaml
```

Expected output:
```json
{
  "success": true,
  "message": "Hello from .komponent!"
}
```

## Creating Your Own .komponent

1. Create a component directory with `component.yaml` and optionally `resources/`:
   ```bash
   mkdir my-component
   cat > my-component/component.yaml << 'EOF'
   apiVersion: kdeps.io/v1
   kind: Component
   metadata:
     name: my-component
   ...
   EOF
   ```

2. Package it:
   ```bash
   kdeps package my-component --output .
   # Creates my-component-1.0.0.komponent
   ```

3. Copy the `.komponent` file to your workflow's `components/` directory.

4. Reference the component's `actionId`s in your workflow's resources as needed.

## Benefits of .komponent

- **Distribution**: Single file contains the complete component
- **Versioning**: Version number in filename makes updates explicit
- **Encapsulation**: Component internals are packaged, not scattered
- **Reusability**: Drop the same `.komponent` into any workflow's `components/` folder

# Mixed Components Example (Packed + Unpacked, Multiple Resources)

This advanced example demonstrates:
- **Unpacked component directories** (`formatter/`)
- **Packed `.komponent` archives** (`data-processor-2.0.0.komponent`)
- **Components with multiple resources**
- **Cross-component dependencies**

## Structure

```
components-advanced/
├── workflow.yaml
├── resources/
│   ├── init.yaml       # Sets initial 'name' parameter
│   └── output.yaml     # Final API response
└── components/
    ├── formatter/                  # Unpacked component (2 resources)
    │   └── component.yaml
    └── data-processor-2.0.0.komponent  # Packed component (2 resources)
```

## Components

### 1. `formatter` (unpacked directory)
Provides two independent resources:
- `formatName`: Creates `formattedName = "Hello, <name>! Welcome!"`
- `addTimestamp`: Sets `timestamp = now()`

### 2. `data-processor` (packed `.komponent`)
Provides two resources with dependencies:
- `uppercaseText`: Requires `formatName`, sets `upper = uppercase(formattedName)`
- `logResult`: Requires `uppercaseText`, logs the processed result

## Workflow Flow

1. `setName` → initializes `name = "Claude"`
2. `formatName` → creates `formattedName`
3. `addTimestamp` → creates `timestamp` (runs in parallel with formatName)
4. `uppercaseText` → waits for `formatName`, creates `upper`
5. `logResult` → waits for `uppercaseText`, executes shell command
6. `finalResponse` → waits for `logResult`, returns all collected data

## Execution Order

The execution graph resolves dependencies automatically:
```
setName
  ├── formatName ──┐
  │                ↓
  └── addTimestamp  uppercaseText ── logResult ── finalResponse
```

**Note:** `addTimestamp` and `formatName` can run in parallel since they have no dependency between them.

## Running the Example

```bash
kdeps run examples/components-advanced/workflow.yaml
```

Expected output:
```json
{
  "success": true,
  "message": "Hello, Claude! Welcome!",
  "upper": "HELLO, CLAUDE! WELCOME!",
  "timestamp": "2025-..."
}
```

## Key Takeaways

- ✅ Unpacked directories and `.komponent` archives can coexist in `components/`
- ✅ Each component can define multiple resources
- ✅ Resources from different components can depend on each other
- ✅ Auto-loading merges all resources into a single execution graph
- ✅ Duplicate `actionId`s are automatically deduplicated (last wins)

This pattern enables:
- Reusable component libraries (packed as `.komponent`)
- Local development with unpacked components (easy to iterate)
- Mix-and-match composition of complex workflows from many components

# Control Flow Examples

This example demonstrates control flow in kdeps using expr-lang features:
- **if-else** via ternary operator (`? :`)
- **and/or** via logical operators (`&&`, `||`, `!`)
- **loops** via functional list operations (`filter`, `map`, `all`, `any`)

## Why No Traditional Loops?

expr-lang intentionally excludes `for`, `while`, and statement-based `if/else` to:
- **Prevent infinite loops** - All expressions must terminate
- **Ensure safety** - No Turing-complete code in templates
- **Promote functional style** - Cleaner, more declarative code

## Features Demonstrated

### 1. If-Else (Ternary Operator)

```yaml
# Simple condition
status: {{age >= 18 ? "adult" : "child"}}

# Nested conditions  
category: {{age < 13 ? "child" : (age < 20 ? "teen" : "adult")}}

# With calculations
discount: {{premium ? price * 0.8 : price}}
```

### 2. Logical Operators

```yaml
# AND
eligible: {{age >= 18 && verified}}

# OR
hasAccess: {{premium || trial}}

# NOT
enabled: {{!disabled}}

# Complex
canPurchase: {{(age >= 18 && verified) || admin}}
```

### 3. List Operations (Loop Alternatives)

```yaml
# Filter (like: for item in items if condition)
adults: {{filter(users, .age >= 18)}}

# Map (like: [item.name for item in items])
names: {{map(users, .name)}}

# All (like: all(item.valid for item in items))
allValid: {{all(items, .valid)}}

# Any (like: any(item.active for item in items))
hasActive: {{any(items, .active)}}
```

## Running the Example

```bash
kdeps run examples/control-flow/workflow.yaml
```

## Learn More

- [Control Flow Documentation](../../docs/CONTROL_FLOW.md) - Complete guide
- [expr-lang Documentation](https://expr-lang.org/docs/language-definition) - Full language reference

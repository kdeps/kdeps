# Replacing expr-lang with Mustache - Implementation Guide

## Goal

Replace expr-lang dependency with mustache while maintaining kdeps custom functions (get, info, env, set, etc.).

## Current State

- **expr-lang** handles ALL expressions:
  - Function calls: `{{ get('q') }}`
  - Arithmetic: `{{ 2 + 2 }}`
  - Comparisons: `{{ x > y }}`
  - Conditionals: `{{ x ? y : z }}`
  
- **mustache** only used for simple variables in unified evaluation

## Target State

- **mustache** as primary template engine
- kdeps functions as mustache lambdas
- Minimal or no expr-lang dependency

## Implementation Approaches

### Approach 1: Pure Mustache (Breaking Changes)

**Convert all syntax to mustache:**

```yaml
# Before:
prompt: "{{ get('q') }}"
result: "{{ count + 10 }}"
status: "{{ score > 80 ? 'Pass' : 'Fail' }}"

# After:
prompt: "{{#get}}q{{/get}}"
result: "{{#add}}count|10{{/add}}"
status: "{{#if}}{{#gt}}score|80{{/gt}}{{/if}}Pass{{else}}Fail{{/if}}"
```

**Pros:**
- Clean separation, no expr-lang
- Logic-less templates
- Smaller binary

**Cons:**
- BREAKING: All existing templates need updates
- More verbose for complex logic
- Users need to learn new syntax

### Approach 2: Hybrid with Preprocessing (Recommended)

**Preprocess templates, use mustache for functions, keep expr for logic:**

```yaml
# User writes (backward compatible):
prompt: "{{ get('q') }}"
result: "{{ count + 10 }}"

# Preprocessor converts:
prompt: "{{#get}}q{{/get}}"  # Mustache lambda
result: "{{ count + 10 }}"   # Keep as expr for arithmetic
```

**Implementation:**
1. Preprocess: Convert `{{ func('arg') }}` → `{{#func}}arg{{/func}}`
2. Mustache lambdas for kdeps functions
3. Keep expr ONLY for arithmetic/comparisons
4. Gradual migration path

**Pros:**
- Backward compatible
- Reduces expr-lang usage significantly
- User-friendly
- Gradual migration

**Cons:**
- Still has expr-lang dependency (but minimal)
- Two systems (simpler than before though)

### Approach 3: Custom Expression Parser

**Parse expr syntax ourselves, evaluate with mustache:**

**Implementation:**
1. Parse: `{{ get('q') }}` → AST
2. Evaluate: Use mustache for variables, custom logic for operations
3. No expr-lang dependency

**Pros:**
- Full control
- Backward compatible
- No expr-lang

**Cons:**
- Complex to implement
- Need to maintain parser
- Risk of bugs

## Recommendation

**Start with Approach 2 (Hybrid with Preprocessing)**

### Phase 1: Implement Preprocessing + Mustache Lambdas
- [x] Create preprocessor (done)
- [ ] Implement mustache lambdas for all kdeps functions
- [ ] Update evaluator to preprocess templates
- [ ] Test backward compatibility

### Phase 2: Minimize expr-lang Usage
- [ ] Identify arithmetic/comparison patterns
- [ ] Create mustache helpers for common operations
- [ ] Provide migration guide

### Phase 3: Optional - Remove expr-lang Entirely
- [ ] If needed, implement custom parser
- [ ] Or accept breaking changes for full mustache

## kdeps Functions as Mustache Lambdas

### Implementation Example

```go
// get() function as mustache lambda
data["get"] = mustache.LambdaFunc(func(text string, render mustache.RenderFunc) (string, error) {
    // text = "q" (from {{#get}}q{{/get}})
    varName, _ := render(text)
    
    // Call kdeps API
    value, err := api.Get(varName)
    if err != nil {
        return "", nil  // Mustache behavior
    }
    
    // Return as string
    return fmt.Sprintf("%v", value), nil
})
```

### All kdeps Functions

- `get(name, typeHint)` → `{{#get}}name{{/get}}` or `{{#getTyped}}name|type{{/getTyped}}`
- `info(field)` → `{{#info}}field{{/info}}`
- `env(name)` → `{{#env}}name{{/env}}`
- `set(key, value)` → `{{#set}}key:value{{/set}}`
- `file(pattern)` → `{{#file}}pattern{{/file}}`
- `safe(value, field)` → `{{#safe}}value:field{{/safe}}`

## Migration Path

### For Users

**Current templates continue working:**
```yaml
prompt: "{{ get('q') }}"  # Still works (preprocessed)
```

**Can adopt new syntax gradually:**
```yaml
prompt: "{{#get}}q{{/get}}"  # New syntax (optional)
```

**Mixed usage supported:**
```yaml
message: "Hello {{#get}}name{{/get}}, result is {{ count * 2 }}"
```

## Decision Points

1. **Full mustache or hybrid?**
   - Recommendation: Hybrid (keep expr for arithmetic)
   
2. **Break backward compatibility?**
   - Recommendation: No (use preprocessing)
   
3. **Remove expr-lang entirely?**
   - Recommendation: Phase 3 (after hybrid working)

## Next Steps

1. Implement mustache lambdas for all kdeps functions
2. Update evaluator to use preprocessing
3. Test with existing examples
4. Measure binary size reduction
5. Document new patterns

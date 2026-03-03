# Control Flow in kdeps

Complete guide to control flow, conditionals, logical operators, and iteration in kdeps expressions.

## Table of Contents

1. [Overview](#overview)
2. [If-Else (Ternary Operator)](#if-else-ternary-operator)
3. [Logical Operators](#logical-operators)
4. [While Loops](#while-loops)
5. [List Operations (Foreach)](#list-operations-foreach)
6. [Common Patterns](#common-patterns)
7. [Best Practices](#best-practices)

## Overview

kdeps uses **expr-lang** for expressions, which provides:
- ✅ **Conditionals** via ternary operator (`? :`)
- ✅ **Logical operators** (`&&`, `||`, `!`)
- ✅ **While loops** via the `loop.while` resource field
- ✅ **List operations** (`filter`, `map`, `all`, `any`, `one`, `none`)

Workflow resources are **Turing complete**: the `loop` block enables conditional iteration with
mutable state. A default safety cap of 1000 iterations applies per resource execution; set
`maxIterations` to any positive integer for tighter or looser control. Turing completeness is
preserved because the cap is configurable, not fixed.

---

## If-Else (Ternary Operator)

### Syntax

```
condition ? valueIfTrue : valueIfFalse
```

### Examples

#### Simple Condition
```yaml
status: {{age >= 18 ? "adult" : "child"}}
```

#### Nested Conditions
```yaml
category: {{age < 13 ? "child" : (age < 20 ? "teen" : "adult")}}
```

#### With Calculations
```yaml
price: {{premium ? basePrice * 0.8 : basePrice}}
total: {{quantity > 10 ? price * 0.9 : price}}
```

#### With Get()
```yaml
discount: {{get('userType') == 'premium' ? 0.2 : 0}}
result: {{get('score') > 80 ? get('score') * 2 : get('score')}}
```

### Common Use Cases

**1. Access Control**
```yaml
canAccess: {{user.age >= 18 && user.verified ? "granted" : "denied"}}
```

**2. Pricing Logic**
```yaml
finalPrice: {{
  user.premium 
    ? price * 0.8 
    : (user.trial ? price * 0.9 : price)
}}
```

**3. Status Messages**
```yaml
message: {{
  score >= 90 ? "Excellent!" :
  score >= 70 ? "Good" :
  score >= 50 ? "Pass" :
  "Fail"
}}
```

---

## Logical Operators

### AND (`&&`)

Both conditions must be true.

```yaml
# Simple AND
eligible: {{age >= 18 && verified}}

# Multiple conditions
canPurchase: {{age >= 18 && verified && balance > 0}}

# With ternary
access: {{admin && active ? "full" : "limited"}}
```

### OR (`||`)

At least one condition must be true.

```yaml
# Simple OR
hasAccess: {{premium || trial}}

# Multiple conditions
canView: {{premium || trial || free}}

# With ternary
level: {{premium || partner ? "pro" : "standard"}}
```

### NOT (`!`)

Negates a boolean value.

```yaml
# Simple NOT
enabled: {{!disabled}}
visible: {{!hidden}}

# Combined
shouldShow: {{!disabled && active}}
```

### Complex Combinations

```yaml
# Parentheses for grouping
result: {{(age >= 18 && verified) || admin}}

# Multiple operators
access: {{(premium || trial) && !suspended && active}}

# With comparisons
valid: {{(score > 50 || bonus > 10) && !disqualified}}
```

---

## While Loops

The `loop` block on a resource enables **while-loop** iteration — the resource body (primary
execution type and `expr` blocks) is repeated as long as the `while` condition is truthy.

### Syntax

```yaml
run:
  loop:
    while: "<expression>"   # required: loop continues while this is truthy
    maxIterations: 1000     # optional: safety cap (default: 1000)
  expr:
    - "{{ <body expressions> }}"
```

`loop` can be combined with any primary execution type (`exec`, `python`, `sql`, `httpClient`, etc.)
or used with only `expr`/`exprBefore`/`exprAfter` blocks.

Every resource block runs on **each iteration** of the loop — including primary execution types
(`httpClient`, `chat`, `exec`, `python`, `sql`, `tts`, `botReply`, `scraper`, `embedding`) and
`apiResponse`. When `apiResponse` is present, each iteration produces one response map; collecting
multiple per-iteration responses constitutes a **streaming response**.

### Loop Context Variables

Inside the loop body, three context accessors are available (callable methods, consistent with
`item.index()`, `item.count()` in the `items` feature):

| Method          | Description                                             |
|-----------------|---------------------------------------------------------|
| `loop.index()`  | Zero-based iteration counter (0, 1, 2, …)              |
| `loop.count()`  | One-based iteration counter (1, 2, 3, …)               |
| `loop.results()`| Results accumulated from all *previous* iterations     |

Loop context is also accessible via `get('key', 'loop')` / `set('key', value, 'loop')`, parallel
to the `'item'` storage type used by the `items` feature.

### Examples

#### Counter Loop

```yaml
metadata:
  actionId: count-to-five
  name: Count to Five
run:
  loop:
    while: "loop.index() < 5"
  expr:
    - "{{ set('result', loop.count()) }}"
  apiResponse:
    success: true
    response:
      count: "{{ get('result') }}"
```

#### Loop with State Mutation

```yaml
metadata:
  actionId: fibonacci
  name: Fibonacci Loop
run:
  loop:
    while: "loop.index() < 10"
    maxIterations: 20
  exprBefore:
    - "{{ set('a', get('a') == nil ? 0 : get('a')) }}"
    - "{{ set('b', get('b') == nil ? 1 : get('b')) }}"
  expr:
    - "{{ set('tmp', get('b')) }}"
    - "{{ set('b', get('a') + get('b')) }}"
    - "{{ set('a', get('tmp')) }}"
  apiResponse:
    success: true
    response:
      fib: "{{ get('a') }}"
```

#### Accumulate Results (loop.results())

```yaml
metadata:
  actionId: collect-three
  name: Collect Three Results
run:
  loop:
    # Stop once we have 3 accumulated results (parallel to item.values() in items)
    while: "len(loop.results()) < 3"
  expr:
    - "{{ set('val', loop.count()) }}"
  apiResponse:
    success: true
    response:
      collected: "{{ loop.results() }}"
```

#### Loop-scoped Variables (set/get with 'loop' type)

```yaml
metadata:
  actionId: loop-scoped-counter
  name: Loop-scoped Counter
run:
  loop:
    # Read loop-scoped variable (parallel to get('k', 'item') in items)
    while: "default(get('step', 'loop'), 0) < 5"
  expr:
    # Write to loop scope (parallel to set('k', v, 'item') in items)
    - "{{ set('step', loop.count(), 'loop') }}"
  apiResponse:
    success: true
```

#### Loop with Primary Execution Type

```yaml
metadata:
  actionId: retry-until-success
  name: Retry Until Success
run:
  loop:
    while: "get('status') != 'ok' && loop.index() < 5"
  httpClient:
    method: GET
    url: "https://api.example.com/status"
  expr:
    - "{{ set('status', http.responseBody('retry-until-success')) }}"
```

### Safety Cap

`maxIterations` prevents runaway loops. When the cap is reached the loop stops silently (it does
not return an error). The default cap is **1000** iterations. Set it explicitly for tighter
control:

```yaml
loop:
  while: "true"
  maxIterations: 5   # run exactly 5 times
```

---

## List Operations (Foreach)

expr-lang doesn't have traditional `for` or `while` loops. Instead, use functional operations.

### filter()

Select items matching a condition.

**Syntax:**
```
filter(array, condition)
```

**Examples:**
```yaml
# Filter adults
adults: {{filter(users, .age >= 18)}}

# Filter by property
active: {{filter(items, .active)}}

# Complex condition
eligible: {{filter(users, .age >= 18 && .verified)}}

# With nested objects
premiumUsers: {{filter(users, .subscription.type == "premium")}}
```

**Equivalent to:**
```python
# Python
[user for user in users if user.age >= 18]
```

### map()

Transform each item in an array.

**Syntax:**
```
map(array, expression)
```

**Examples:**
```yaml
# Extract property
names: {{map(users, .name)}}
emails: {{map(users, .email)}}

# Calculate
doubled: {{map(numbers, . * 2)}}

# With ternary
labels: {{map(users, .age >= 18 ? .name + " (adult)" : .name)}}

# Nested properties
cities: {{map(users, .address.city)}}
```

**Equivalent to:**
```python
# Python
[user.name for user in users]
```

### all()

Check if ALL items match a condition.

**Syntax:**
```
all(array, condition)
```

**Examples:**
```yaml
# All adults?
allAdults: {{all(users, .age >= 18)}}

# All active?
allActive: {{all(items, .active)}}

# Complex condition
allEligible: {{all(users, .age >= 18 && .verified)}}
```

**Equivalent to:**
```python
# Python
all(user.age >= 18 for user in users)
```

### any()

Check if ANY item matches a condition.

**Syntax:**
```
any(array, condition)
```

**Examples:**
```yaml
# Any admin?
hasAdmin: {{any(users, .admin)}}

# Any active?
hasActive: {{any(items, .active)}}

# Complex
hasEligible: {{any(users, .age >= 18 && .verified)}}
```

**Equivalent to:**
```python
# Python
any(user.admin for user in users)
```

### one()

Check if EXACTLY ONE item matches.

```yaml
hasOneAdmin: {{one(users, .admin)}}
```

### none()

Check if NO items match.

```yaml
noSuspended: {{none(users, .suspended)}}
```

---

## Common Patterns

### Pattern 1: Conditional Filtering

```yaml
# Filter with dynamic condition
results: {{filter(items, .price >= get('minPrice'))}}

# Filter and transform
discountedPrices: {{map(
  filter(items, .price > 100),
  .price * 0.9
)}}
```

### Pattern 2: Validation

```yaml
# Check if all required fields present
valid: {{all([name, email, age], . != "")}}

# Check if any error exists
hasErrors: {{any(validations, .error != null)}}
```

### Pattern 3: Access Control

```yaml
# Multi-level access check
access: {{
  admin ? "full" :
  (verified && age >= 18) ? "standard" :
  "limited"
}}
```

### Pattern 4: Data Transformation

```yaml
# Transform and filter
output: {{map(
  filter(users, .active),
  {
    name: .name,
    email: .email,
    premium: .premium
  }
)}}
```

### Pattern 5: Aggregation

```yaml
# Count matching items
activeCount: {{len(filter(items, .active))}}

# Check completion
allComplete: {{all(tasks, .status == "done")}}
```

---

## Best Practices

### 1. Use Parentheses for Clarity

```yaml
# Good
result: {{(age >= 18 && verified) || admin}}

# Less clear
result: {{age >= 18 && verified || admin}}
```

### 2. Break Complex Expressions

```yaml
# Instead of one huge expression
adults: {{filter(users, .age >= 18)}}
verifiedAdults: {{filter(adults, .verified)}}
names: {{map(verifiedAdults, .name)}}
```

### 3. Use Meaningful Variable Names

```yaml
# Good
eligibleUsers: {{filter(users, .age >= 18 && .verified)}}

# Less clear
result: {{filter(users, .age >= 18 && .verified)}}
```

### 4. Prefer Functional Over Nested Ternary

```yaml
# Good - use filter/map
activeUsers: {{filter(users, .active)}}

# Avoid - deeply nested ternary
result: {{x ? (y ? (z ? a : b) : c) : d}}
```

### 5. Document Complex Logic

```yaml
# Calculate final price with multi-tier discounts
# Premium: 20% off, Trial: 10% off, Regular: full price
finalPrice: {{
  user.premium ? price * 0.8 :
  user.trial ? price * 0.9 :
  price
}}
```

---

## Examples

See working examples in:
- `examples/control-flow/` - Complete demonstrations
- `pkg/parser/expression/evaluator_controlflow_test.go` - 22 test cases

---

## Learn More

- [expr-lang Language Definition](https://expr-lang.org/docs/language-definition)
- [Hybrid Expressions](./HYBRID_EXPRESSIONS.md)
- [Mustache Expressions](./MUSTACHE_EXPRESSIONS_IMPLEMENTATION.md)
- [Expression Documentation Hub](./README_EXPRESSIONS.md)

---

## Summary

✅ **If-Else**: Ternary operator (`condition ? true : false`)  
✅ **AND/OR**: Logical operators (`&&`, `||`, `!`)  
✅ **While Loops**: `loop.while` resource field with `loop.index()` / `loop.count()` / `loop.results()` context  
✅ **Foreach**: `items` resource field iterates a list  
✅ **Functional**: List operations (`filter`, `map`, `all`, `any`)  
✅ **Turing Complete**: Unbounded conditional iteration + mutable state = full computational power  

**Control flow in kdeps is functional, safe, and expressive!**

# Control Flow in kdeps

Complete guide to control flow, conditionals, logical operators, and iteration in kdeps expressions.

## Table of Contents

1. [Overview](#overview)
2. [If-Else (Ternary Operator)](#if-else-ternary-operator)
3. [Logical Operators](#logical-operators)
4. [List Operations (Loop Alternatives)](#list-operations-loop-alternatives)
5. [Why No Traditional Loops?](#why-no-traditional-loops)
6. [Common Patterns](#common-patterns)
7. [Best Practices](#best-practices)

## Overview

kdeps uses **expr-lang** for expressions, which provides:
- ✅ **Conditionals** via ternary operator (`? :`)
- ✅ **Logical operators** (`&&`, `||`, `!`)
- ✅ **List operations** (`filter`, `map`, `all`, `any`, `one`, `none`)
- ❌ **No traditional loops** (`for`, `while`) - by design for safety

All expressions are **guaranteed to terminate** - no infinite loops possible.

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

## List Operations (Loop Alternatives)

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

## Why No Traditional Loops?

### Design Philosophy

expr-lang **intentionally excludes** traditional loops (`for`, `while`, `do-while`).

**Reasons:**

1. **Safety** - Prevents infinite loops
2. **Termination** - All expressions guaranteed to finish
3. **Predictability** - No Turing-complete code
4. **Simplicity** - Functional approach is cleaner

### What About Complex Iteration?

Use **functional composition**:

```yaml
# Get names of adult users
adultNames: {{map(filter(users, .age >= 18), .name)}}

# Count verified adults
verifiedAdultCount: {{len(filter(users, .age >= 18 && .verified))}}

# Get emails of premium users
premiumEmails: {{map(filter(users, .premium), .email)}}
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
✅ **Loops**: Functional operations (`filter`, `map`, `all`, `any`)  
✅ **Safe**: No infinite loops - all expressions terminate  
✅ **Powerful**: Compose operations for complex logic  

**Control flow in kdeps is functional, safe, and expressive!**

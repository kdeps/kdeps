# Contributing to kdeps docs

Add or change a page under `docs/v2/`. Read the
[style guide](./style-guide) first.

## 1. Pick one content type

A page is exactly one type. If your page answers two questions, split it.

| You want to... | Use type | Skeleton below |
| :--- | :--- | :--- |
| Explain an idea or how a subsystem works | Concept | [Concept](#concept-skeleton) |
| Build one working example end to end | Tutorial | [Tutorial](#tutorial-skeleton) |
| Give exact steps for one task the reader already understands | How-to | [How-to](#how-to-skeleton) |
| List fields, flags, functions, or values | Reference | [Reference](#reference-skeleton) |
| Get a new user to a working result fast | Quickstart | [Quickstart](#quickstart-skeleton) |
| Help a user fix a specific failure | Troubleshooting | [Troubleshooting](#troubleshooting-skeleton) |

## 2. Use the matching skeleton

### Concept skeleton

```markdown
# <Noun phrase>

<One sentence: "X does Y". State the mode only on shared pages.>

<Definition paragraph. What it is, what problem it solves.>

<D2 or ASCII diagram if the concept has 4+ parts or branching.>

## Background   <!-- optional -->
## Use cases
## Comparison of <alternatives>   <!-- optional -->
## See also
```

### Tutorial skeleton

```markdown
# <Verb phrase: "Build a ...">

<One or two sentences: what you build. Facts the reader needs, if any.>

**Needs:** <only real dependencies: a tool, an account, a model. Omit if none.>

## <Task 1>
1. <Verb-first step.>

## Next steps
```

### How-to skeleton

```markdown
# <Verb phrase: "Deploy kdeps to ...">

## Overview
<One or two sentences: what this achieves and when to do it.>

## Before you start   <!-- optional -->
## <Task>
1. <Verb-first step.>

## See also
```

### Reference skeleton

```markdown
# <Thing> reference

<Scope: what this covers and how it relates to other pages. State the mode(s).>

## <Group>
| Field | Description | Example |
| :--- | :--- | :--- |

## Commands   <!-- optional -->
| Command | Description | Argument | Example |
| :--- | :--- | :--- | :--- |

## Examples
## See also
```

Resource pages (`resources/*`) are reference pages. Use:
`Overview` -> `Fields` table -> `Examples` -> `See also`.

### Quickstart skeleton

```markdown
# <Product> quickstart

<One sentence: what you will have running at the end.>

## Install
## Part 1: <task>
### Step 1: <step>

## Next steps
```

### Troubleshooting skeleton

```markdown
# Troubleshooting <area>

<Scope in one or two sentences.>

## Symptom: <what the user sees>
### Cause
### Solution
### For more information
```

## 3. Validate

```bash
cd docs/v2
npm run build     # VitePress build - catches dead links and broken D2 blocks
```

Update `README.md` and `book/` in the same change when a page changes behavior,
schema, or a CLI flag.

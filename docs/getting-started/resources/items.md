---
outline: deep
---

# Items Iteration

Items iteration enables a resource to process a sequence of items in a loop, facilitating efficient handling of multiple inputs or tasks. By defining an `items` block, you specify a set of values to iterate over, which can be accessed using `item.current()`, `item.prev()`, and `item.next()` within the resource's `run` block. This feature is versatile and applicable to various resource types, such as API processing, file operations, or LLM chat sessions.

## Defining an `items` Block

The `items` block is declared within a resource, listing the values to be processed sequentially. Each item is handled individually by the `run` block, allowing the resource to execute its logic for each value.

### Example: Iterating Over Song Lyrics

In this example, a resource iterates over lines from the song "American Pie" by Don McLean, processing each line. This could represent analyzing lyrics, storing them, or passing them to another system.

```apl
amends "package://schema.kdeps.com/core@0.2.30#/Resource.pkl"

actionID = "processLyrics"
name = "Process Lyrics Resource"
description = "This resource processes song lyrics line by line."
category = ""

items {
    "A long, long time ago"
    "I can still remember"
    "How that music used to make me smile"
    "And I knew if I had my chance"
}

run {
    restrictToHTTPMethods {
        "GET"
    }
    restrictToRoutes {
        "/api/v1/lyrics"
    }
    // Process the current lyric line (e.g., store, analyze, or pass to another system)
    local result = "@(item.current())""
}
```

Here, the `run` block assigns the current lyric line to a local variable `result` for processing. The resource is restricted to `GET` requests on the `/api/v1/lyrics` route, showing how iteration integrates with other resource constraints. The actual processing of `result` depends on the implementation (e.g., storing in a database, analyzing sentiment, or sending to an API).

## Accessing Iteration Context

The following methods provide access to the current, previous, and next items during iteration:

| **Method**       | **Description**                                                                 |
|------------------|---------------------------------------------------------------------------------|
| `item.current()` | Returns the current item in the iteration (e.g., "A long, long time ago" in the first iteration). |
| `item.prev()`    | Returns the previous item, or an empty string (`""`) if there is no previous item (e.g., `""` for the first lyric). |
| `item.next()`    | Returns the next item, or an empty string (`""`) if there is no next item (e.g., `""` for the last lyric). |

### Example: Contextual Processing with `item.prev()` and `item.next()`

You can use the iteration context to build complex processing logic, such as combining lyric lines or maintaining sequence information.

```apl
items {
    "A long, long time ago"
    "I can still remember"
    "How that music used to make me smile"
    "And I knew if I had my chance"
}

run {
    local message = """
    Current lyric: @(item.current())
    Previous lyric: @(item.prev() ?: "none")
    Next lyric: @(item.next() ?: "none")
    """
    // Handle the message (e.g., store it, send it to an API, or process further)
}
```

For the item "I can still remember", the `message` variable would contain:

```
Current lyric: I can still remember
Previous lyric: A long, long time ago
Next lyric: How that music used to make me smile
```

This example constructs a string using `item.current()`, `item.prev()`, and `item.next()`. The `?:` operator provides a fallback value ("none") when `item.prev()` or `item.next()` returns an empty string (`""`) at the start or end of the iteration. The `message` can then be processed according to the resource's requirements.

## Combining Items Iteration with Resource Features

Items iteration can be paired with other resource configurations, such as `skipCondition`, `restrictToHTTPMethods`, `restrictToRoutes`, or `preflightCheck`, to create tailored workflows.

### Example: Skipping Specific Items

You can use a `skipCondition` to bypass certain items during iteration.

```apl
items {
    "A long, long time ago"
    "I can still remember"
    "How that music used to make me smile"
    "And I knew if I had my chance"
}

run {
    skipCondition {
        "@(item.current())" == "How that music used to make me smile" // Skip this lyric
    }
    // Process the current lyric (e.g., pass to a system or store)
    local result = "@(item.current())"
}
```

In this case, the resource processes all lyrics except "How that music used to make me smile", demonstrating how `skipCondition` refines iteration behavior.

### Example: File Processing with Iteration

Items iteration is also useful for processing a list of files or resources.

```apl
items {
    "/tmp/verse1.txt"
    "/tmp/verse2.txt"
    "/tmp/verse3.txt"
}

run {
    local content = "@(read?(item.current())?.text)"
    // Process the file content (e.g., validate, transform, or store)
}
```

Here, the resource reads the content of each file specified in the `items` block and processes it, illustrating a non-API use case.

## Using Items Iteration in Specialized Resources

Items iteration can be applied to specialized resources, such as an LLM chat resource, where each item might represent a prompt or input for generating creative content.

### Example: LLM Chat Resource for MTV Video Scenarios

The following example uses an LLM chat resource to iterate over lyrics from "American Pie," asking the AI to generate a suitable scenario for an MTV music video based on each lyric line.

```apl
amends "package://schema.kdeps.com/core@0.2.30#/Resource.pkl"

actionID = "llmResource"
name = "LLM Chat Resource"
description = "This resource generates MTV video scenarios based on song lyrics."
category = ""

items {
    "A long, long time ago"
    "I can still remember"
    "How that music used to make me smile"
    "And I knew if I had my chance"
}

run {
    restrictToHTTPMethods {
        "GET"
    }
    restrictToRoutes {
        "/api/v1/mtv-scenarios"
    }
    skipCondition {
        "@(item.current())" == "And I knew if I had my chance" // Skip this lyric
    }
    chat {
        model = "llama3.2:1b"
        role = "user"
        prompt = """
        Based on the lyric "@(item.current())" from the song "American Pie," generate a suitable scenario for an MTV music video. The scenario should include a vivid setting, key visual elements, and a mood that matches the lyric's tone.
        """
        JSONResponse = true
        JSONResponseKeys {
            "setting"
            "visual_elements"
            "mood"
        }
        timeoutDuration = 60.s
    }
}
```

In this LLM chat resource, the `chat` block uses `item.current()` within the `prompt` to ask the AI to generate an MTV music video scenario for each lyric. The resource processes "A long, long time ago", "I can still remember", and "How that music used to make me smile" (skipping "And I knew if I had my chance" due to the `skipCondition`). Each iteration sends the prompt to the LLM and receives a structured JSON response with `setting`, `visual_elements`, and `mood` keys. For example, for the lyric "A long, long time ago", the LLM might return a scenario with a nostalgic 1950s diner setting, vintage cars, and a wistful mood.

## Best Practices

- **Use Meaningful Items**: Select item values that align with the resource’s purpose, such as lyric lines, file paths, or prompts.
- **Handle Empty Strings**: Use fallback operators (e.g., `?:`) to manage empty string (`""`) results from `item.prev()` or `item.next()` at the iteration boundaries.
- **Integrate Constraints**: Combine iteration with `skipCondition`, `restrictToHTTPMethods`, or `restrictToRoutes` to control execution flow.
- **Test Incrementally**: Validate the `items` block with a small dataset to ensure correct behavior before scaling.

Items iteration provides a robust mechanism to process multiple inputs sequentially, making it a valuable tool for diverse resource types and workflows, from lyric processing to API-driven or LLM-based applications like generating MTV video scenarios.

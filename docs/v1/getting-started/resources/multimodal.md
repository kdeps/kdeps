---
outline: deep
---

# Multi-Modal LLM Models

KDeps enables seamless interaction with multi-modal LLM models such as `llava` and `llama3.2-vision`. For a
comprehensive list of supported multi-modal LLM models, refer to the [Ollama Vision](https://ollama.com/search?c=vision)
models page.

### Adding an LLM Model

Before using an LLM model, you need to include it in the `workflow.pkl` file. For example:

```apl
Models {
    "llama3.3"
    "llama3.2-vision"
}
```

### Interacting with an LLM Model

To interact with a model, you'll need to provide a prompt and a file. Below is an example of an LLM Resource that
utilizes the `llama3.2-vision` model with a file uploaded via the API:

```apl
ActionID = "llamaVision"
Chat {
  Model = "llama3.2-vision"
  Prompt = "Describe this image"
  JSONResponse = true
  JSONResponseKeys = {
    "description_text"
    "style_text"
    "category_text"
  }
  Files {
    "@(request.files()[0])" // Uses the first uploaded file
  }
}
```

### Using Processed Image Data

Once the image is processed, you can leverage its output in your resources. For example:

```apl
local jsonPath = "@(llm.file("llamaVision"))"
local JSONData = "@(read?("\(jsonPath)")?.text)"

local imageDescription = "@(document.JSONParser(JSONData)?.description_text)"
local imageStyle = "@(document.JSONParser(JSONData)?.style_text)"
local imageCategory = "@(document.JSONParser(JSONData)?.category_text)"
```

This example demonstrates how to extract key details like description, style, and category from the processed image data
for further use in your workflow.

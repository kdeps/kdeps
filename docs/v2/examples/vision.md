# Analyze an uploaded image

Build an API that accepts an image upload and a question. The API sends both to
a multimodal LLM and returns a structured description of the image.

The chat gets the uploaded image as a file:

<div v-pre>

```yaml
# resources/vision-llm.yaml (part)
chat:
  model:
    name: llama3.2-vision
  context:
    prompt: "{{ get('q') }}"
    files:
      - "{{ get('file', 'filepath') }}"   # the uploaded file
```

</div>

## Read next

- [Full walkthrough](/reference/examples/vision) - every step, file and command
- [LLM resource](/workflow/resources/llm) - vision, files, streaming, tools
- [LLM backends](/workflow/resources/llm-backends) - Ollama configuration

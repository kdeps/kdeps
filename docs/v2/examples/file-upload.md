# Handle a file upload

Build an endpoint that accepts one or more file
uploads and returns their metadata: count, names, MIME types, and the path of
the first file on disk.

The response reads the upload metadata:

<div v-pre>

```yaml
# resources/file-processor.yaml (part)
apiResponse:
  success: true
  response:
    file_count: "{{ info('filecount') }}"
    files: "{{ info('files') }}"
    first_file:
      path: "{{ get('file', 'filepath') }}"    # 'file' = the form field name
```

</div>

## Read next

- [Full walkthrough](/reference/examples/file-upload) - every step, file and command
- [Image analysis tutorial](/examples/vision) - attach the upload to an LLM
- [Document summarizer tutorial](/examples/file-processor) - the `file` input source

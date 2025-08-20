# File Uploads

KDeps supports uploading files via its [API](../configuration/workflow#api-server-settings) in addition to sourcing files from the [Data Folder](../resources/data).

To enable file uploads through the KDeps API, you must configure the HTTP `POST` method in the `methods` section of your configuration. For further details, refer to the [API Routes](../configuration/workflow#api-routes) documentation.

## Uploading Files as `FORM` Data

Files are uploaded via a `POST` request containing a `FORM` with the field name `file[]`. This format supports uploading multiple files simultaneously.

Here’s an example of using `curl` to send a form with multiple files. The `@` prefix indicates that the files are binary:

```bash
curl 'http://localhost:3000/api/v1/files' -X POST \
      -F "file[]=@file1.jpg" -F "file[]=@file2.png"
```

## Accessing Uploaded Files in Resources

You can access uploaded files in your resources using the `request` function.

The request functions that we are going to discuss for handling uploaded files are:

- **`request.files()`**: Retrieves a list of uploaded file paths.
- **`request.filetypes()`**: Returns the MIME types of uploaded files.
- **`request.filesByType("mimetype")`**: Filters and retrieves files of a specific MIME type.

There are other available request file operation functions. For the full list of available request
functions, refer to the [API Request Functions](../resources/functions#api-request-functions) documentation.

### Example Usage

Using the earlier `curl` example:

- To retrieve the path of the first uploaded file: `"@(request.files()[0])"`.
- To determine the file type of the first uploaded file: `"@(request.filetypes()[0])"`.
- To access the second file’s path: `"@(request.files()[1])"`, which corresponds to `file2.png`.

The `request.filesByType("mimetype")` function is particularly useful for filtering files based on their MIME type.

---
outline: deep
---

# Global Functions

Global functions are independent utilities designed to support and enhance resource functionality. They provide tools
that resources can leverage for efficient operations and interaction. Written in [Apple PKL](https://pkl-lang.org),
these functions are built with flexibility in mind, allowing you to modify and extend them as needed to suit your
specific use cases.

Below is a list of the global functions available for each resource:

## API Request Functions

> *Note:* Request functions does not work on Lambda mode. Doing so will cause an exception on the AI Agents.

| **Function**                    | **Description**                                                                |
|:--------------------------------|:-------------------------------------------------------------------------------|
| request.ID()                    | Retrieves the unique request UUID.                                             |
| request.IP()                    | Retrieves the client IP address.                                               |
| request.data()                  | Retrieves the request body data.                                               |
| request.params("id")            | Fetches the value of a specific HTTP parameter from the request.               |
| request.header("id")            | Fetches the value of a specific HTTP header from the request.                  |
| request.file("name")            | Accesses details of an uploaded file, including its `filepath` and `filetype`. |
| request.filetype("name")        | Retrieves the MIME type of an uploaded file by its name.                       |
| request.filepath("name")        | Retrieves the file path of an uploaded file by its name.                       |
| request.filecount()             | Returns the total number of uploaded files in the request.                     |
| request.files()                 | Retrieves a list of file paths for all uploaded files.                         |
| request.filetypes()             | Retrieves a list of MIME types for all uploaded files.                         |
| request.filesByType("mimetype") | Retrieves file paths of all uploaded files matching the specified MIME type.   |
| request.path()                  | Retrieves the URI path of the API request.                                     |
| request.method()                | Retrieves the HTTP method (e.g., GET, POST) of the API request.                |

## Item Loop Interaction

| **Function**   | **Description**                       |
|:---------------|:--------------------------------------|
| item.current() | Fetches the current item in the loop  |
| item.prev()    | Fetches the previous item in the loop |
| item.next()    | Fetches the next item in the loop     |

## Manual LLM Tool Execution

| **Function**                               | **Description**                                  |
|:-------------------------------------------|:-------------------------------------------------|
| tool.getOutput("key")                      | Fetches the results of the tool execution of key |
| tool.runScript("key", "path", "arg1,arg2") | Execute the script with arguments                |
| tool.history("key")                        | Fetches the output history execution of the key  |

## Memory Operation Functions

| **Function**                      | **Description**                                   |
|:----------------------------------|:--------------------------------------------------|
| memory.getRecord("key")           | Fetches the value of key from persistent memory   |
| memory.deleteRecord("key")        | Delete the memory item from the persistent memory |
| memory.setRecord("key", "value")  | Stores the value of key to the persistent memory  |
| memory.clear()                    | Clears all persistent memory (CAUTION!)           |
| session.getRecord("key")          | Fetches the value of key from session memory      |
| session.deleteRecord("key")       | Delete the memory item from the session memory    |
| session.setRecord("key", "value") | Stores the value of key to the session memory     |
| session.clear()                   | Clears all session memory (CAUTION!)              |

> *Note:* The `setRecord`, `deleteRecord` and `clear` are side-effecting functions—it performs an action but doesn't return a
> meaningful value. That is why it is recommended to placed them inside an `expr` block: to ensure the expression is
> evaluated for its effect.

## Data Folder Functions

| **Function**                                    | **Description**                                                                                                                    |
|:------------------------------------------------|:-----------------------------------------------------------------------------------------------------------------------------------|
| data.filepath("agent_name/version", "filename") | Returns the file path of a stored file within the `data/` folder. Requires specifying the `agent_name/version` and the `filename`. |

## Skip Condition Functions

| **Function**                  | **Description**                                       |
|:------------------------------|:------------------------------------------------------|
| skip.ifFileExists("string")   | Returns true if file exists; false otherwise          |
| skip.ifFolderExists("string") | Returns true if folder exists; false otherwise        |
| skip.ifFileIsEmpty("string")  | Returns true if file is empty exists; false otherwise |

## Document JSON Parsers

| **Function**                         | **Description**                                             |
|:-------------------------------------|:------------------------------------------------------------|
| document.JSONParser("string")        | Parse a JSON `String` and returns a native `Dynamic` object |
| document.JSONParserMapping("string") | Parse a JSON `String` and returns a native `Mapping` object |

## Document JSON, YAML and XML Generators

| **Function**                     | **Description**                                      |
|:---------------------------------|:-----------------------------------------------------|
| document.JSONRenderDocument(Any) | Parse `Any` object and returns a JSON `String`       |
| document.JSONRenderValue(Any)    | Parse `Any` object and returns a JSON `String` Value |
| document.yamlRenderDocument(Any) | Parse `Any` object and returns a Yaml `String`       |
| document.yamlRenderValue(Any)    | Parse `Any` object and returns a Yaml `String` Value |
| document.xmlRenderDocument(Any)  | Parse `Any` object and returns a XML `String`        |
| document.xmlRenderValue(Any)     | Parse `Any` object and returns a XML `String` Value  |

## PKL Modules

These PKL modules come pre-included in KDeps via `import "pkl:<module>"`. They are readily available for use via
`<functon>.<property>` or `<functon>.<methods>`, allowing you to extend and enhance your resource functions further.


For detailed information on individual methods and properties, refer to the [Apple PKL
Documentation](https://pkl-lang.org/package-docs/pkl/current/index.html).

| **Function** | **Description**                                                   |
|:-------------|:------------------------------------------------------------------|
| test         | A template for writing tests                                      |
| math         | Basic math constants and functions                                |
| platform     | Information about the runtime platform                            |
| semver       | Parsing, comparison, and manipulation of semantic version numbers |
| shell        | Utilities for generating shell scripts.                           |

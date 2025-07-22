# Data & Memory

Comprehensive data management, storage, and file handling capabilities for Kdeps AI agents. These resources provide persistent storage, file operations, data processing, and state management across requests and sessions.

## Available Data & Memory Resources

### **Memory Operations (`memory.md`)**
**Persistent Data Storage and State Management**

The Memory system provides persistent data storage that survives across requests and sessions. Features include:
- Key-value storage with JSON serialization
- Cross-request data persistence
- Session-scoped and global-scoped storage
- Automatic data type handling and serialization
- Memory cleanup and expiration policies

**Key Operations**:
- **Set Operations**: `memory.setRecord(key, value)`, `memory.setBatch(data)`
- **Get Operations**: `memory.get(key)`, `memory.getAll()`, `memory.exists(key)`
- **Delete Operations**: `memory.delete(key)`, `memory.clear()`, `memory.clearPattern(pattern)`
- **Query Operations**: `memory.search(pattern)`, `memory.keys()`, `memory.size()`

**Data Types Supported**:
- Primitive types (string, number, boolean)
- Complex objects and arrays
- Nested data structures
- Binary data (handled as direct strings)

**Use Cases**: User sessions, application state, caching, configuration storage, temporary data

**Example Applications**:
- User preference storage across sessions
- Caching expensive API responses
- Workflow state management
- Application configuration and settings

### **Data Folder (`data.md`)**
**File Management and Project Data Organization**

The Data Folder system manages files and project-specific data within the agent environment. Capabilities include:
- File upload and download handling
- Directory structure management
- File metadata and versioning
- Secure file access and permissions
- File type validation and processing

**File Operations**:
- **Upload/Download**: File transfer to/from agent environment
- **Directory Management**: Create, list, and manage folder structures
- **File Processing**: Text processing, image handling, document parsing
- **Metadata Management**: File information, timestamps, checksums
- **Access Control**: File permissions and security policies

**Supported File Types**:
- Text files (TXT, CSV, JSON, XML, YAML)
- Documents (PDF, DOC, DOCX, RTF)
- Images (PNG, JPG, GIF, SVG, WebP)
- Data files (Excel, databases, archives)
- Binary files and custom formats

**Use Cases**: Document processing, file uploads, data imports, asset management, backup storage

### **Working with JSON (`json.md`)**
**JSON Processing Utilities and Data Manipulation**

The JSON utilities provide comprehensive JSON processing capabilities. Features include:
- JSON parsing and validation
- Schema validation and enforcement
- Data transformation and manipulation
- JSON Path queries and updates
- Performance-optimized processing

**JSON Operations**:
- **Parsing**: `document.JSONParser(data)`, `json.parse(string)`
- **Validation**: Schema validation, structure checking
- **Transformation**: Data mapping, field extraction, format conversion
- **Querying**: JSONPath queries, conditional selection
- **Generation**: Dynamic JSON creation, template-based generation

**Advanced Features**:
- **Schema Enforcement**: Validate JSON against predefined schemas
- **Data Transformation**: Convert between different JSON structures
- **Batch Processing**: Handle multiple JSON documents efficiently
- **Error Handling**: Graceful handling of malformed JSON
- **Performance Optimization**: Streaming processing for large datasets

**Use Cases**: API data processing, configuration management, data transformation, validation

### **File Uploads (`files.md`)**
**File Handling and Upload Processing**

The File Upload system handles file uploads and processing within AI workflows. Capabilities include:
- Multi-file upload support
- File type detection and validation
- Automatic file processing pipelines
- Temporary and permanent file storage
- Integration with data processing resources

**Upload Features**:
- **Multiple Formats**: Support for various file types and formats
- **Size Limits**: Configurable file size and upload limits
- **Validation**: File type, size, and content validation
- **Processing**: Automatic processing based on file type
- **Storage**: Temporary and persistent storage options

**Processing Pipelines**:
- **Text Files**: Automatic parsing and content extraction
- **Images**: Metadata extraction, thumbnail generation, format conversion
- **Documents**: Text extraction, OCR processing, structure analysis
- **Data Files**: CSV/Excel parsing, data validation, format conversion
- **Archives**: Extraction and individual file processing

**Security Features**:
- File type whitelisting and blacklisting
- Content scanning and malware detection
- Access control and permission management
- Secure temporary storage with automatic cleanup

**Use Cases**: Document upload processing, image analysis, data import, content management

## Data Management Patterns

### **Session-Based Storage**
```apl
// Store user session data across requests
Expr {
    "@(memory.setRecord('user_session', request.data().sessionId))"
    "@(memory.setRecord('last_activity', utils.timestamp()))"
}
```

### **File Processing Pipeline**
```apl
// Process uploaded files through multiple stages
ActionID = "fileProcessingResource"
Requires { "uploadValidation"; "contentExtraction"; "dataAnalysis" }
Run {
    FileUpload {
        AllowedTypes { "pdf"; "docx"; "txt" }
        MaxSize = "10MB"
        ProcessingPipeline = "documentAnalysis"
    }
}
```

### **JSON Data Transformation**
```apl
// Transform and validate JSON data
Expr {
    "local processedData = @(document.JSONParser(request.data().json))"
    "@(memory.setRecord('processed_data', processedData))"
}
```

### **Persistent Cache Implementation**
```apl
// Implement caching with expiration
Expr {
    "if @(memory.exists('cache_key')) == false {
        @(memory.setRecord('cache_key', @(client.responseBody('apiResource'))))
        @(memory.setRecord('cache_expiry', utils.timestamp() + 3600))
    }"
}
```

## Advanced Data Patterns

### **Data Aggregation Workflow**
```apl
// Collect and aggregate data from multiple sources
ActionID = "dataAggregatorResource"
Requires { "sourceA"; "sourceB"; "sourceC" }
Run {
    Expr {
        "local aggregatedData = {
            'sourceA': @(memory.get('sourceA_data')),
            'sourceB': @(memory.get('sourceB_data')),
            'sourceC': @(memory.get('sourceC_data'))
        }"
        "@(memory.setRecord('aggregated_result', aggregatedData))"
    }
}
```

### **File-Based Data Processing**
```apl
// Process uploaded files and store results
ActionID = "fileDataProcessor"
Run {
    FileUpload {
        ProcessingRules {
            "csv": "parseCSVData"
            "json": "validateJSONSchema"
            "pdf": "extractTextContent"
        }
        StorageLocation = "data/processed/"
    }
    
    Expr {
        "@(memory.setRecord('processing_status', 'completed'))"
        "@(memory.setRecord('file_count', data.fileCount()))"
    }
}
```

### **Multi-Format Data Export**
```apl
// Export data in multiple formats
ActionID = "dataExportResource"
Run {
    DataExport {
        Source = "@(memory.get('report_data'))"
        Formats {
            "json": "data/exports/report.json"
            "csv": "data/exports/report.csv"
            "pdf": "data/exports/report.pdf"
        }
    }
}
```

## Performance Optimization

### **Memory Management**
- Use appropriate data types for storage efficiency
- Implement memory cleanup for temporary data
- Consider memory limits for large datasets
- Use batch operations for multiple records

### **File Handling**
- Implement streaming for large file processing
- Use appropriate file storage locations
- Clean up temporary files automatically
- Optimize file processing pipelines

### **JSON Processing**
- Use streaming parsers for large JSON files
- Implement schema validation early
- Cache parsed results when appropriate
- Optimize JSONPath queries

## Security Considerations

### **Data Protection**
- Encrypt sensitive data in memory storage
- Implement proper access controls
- Use secure file storage locations
- Regular cleanup of temporary data

### **File Security**
- Validate all uploaded files
- Implement file type restrictions
- Scan for malicious content
- Use secure file processing pipelines

### **Access Control**
- Implement proper authentication for data access
- Use session-based security for user data
- Audit data access and modifications
- Implement data retention policies

## Best Practices

### **Memory Usage**
- Use clear, descriptive keys for stored data
- Implement data expiration policies
- Regular cleanup of unused data
- Monitor memory usage and performance

### **File Management**
- Organize files in logical directory structures
- Use consistent naming conventions
- Implement backup and recovery procedures
- Monitor file storage usage

### **Data Validation**
- Validate all input data early
- Use schema validation for structured data
- Implement data integrity checks
- Provide clear error messages for validation failures

### **Performance**
- Cache frequently accessed data
- Use batch operations for multiple records
- Implement efficient data structures
- Monitor and optimize data access patterns

## Integration Examples

### **Complete Data Processing Workflow**
```apl
// Full data processing pipeline with validation and storage
ActionID = "completeDataProcessor"
Requires { "uploadValidator"; "dataParser"; "dataAnalyzer" }
Run {
    // File upload with validation
    FileUpload {
        AllowedTypes { "csv"; "json"; "xlsx" }
        MaxSize = "50MB"
        Validation = "dataSchemaValidation"
    }
    
    // Process and store data
    Expr {
        "local processedData = @(document.JSONParser(data.content()))"
        "@(memory.setRecord('raw_data', processedData))"
        "@(memory.setRecord('processing_timestamp', utils.timestamp()))"
    }
    
    // Generate report and export
    DataExport {
        Source = "@(memory.get('analysis_results'))"
        Format = "json"
        Location = "data/reports/analysis_result.json"
    }
}
```

## Next Steps

- **[Core Resources](../core-resources/README.md)**: Learn about resources that work with data
- **[Functions & Utilities](../functions-utilities/README.md)**: Explore data manipulation functions
- **[Workflow Control](../workflow-control/README.md)**: Add validation and flow control for data processing

See each documentation file for detailed configuration options, examples, and implementation patterns. 
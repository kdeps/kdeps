// Package messages centralizes all log and API-response message literals so they can
// be reused across the code-base and kept consistent.  Constants are grouped by
// functional area (Docker, Resolver, Archiver, Downloader, Utils, etc.).
package messages

// Log and API response message constants.
const (
	// Docker – server utilities
	MsgServerCheckingReady = "checking if ollama server is ready"
	MsgServerWaitingReady  = "waiting for ollama server to be ready..."
	MsgServerReady         = "ollama server is ready"
	MsgServerNotReady      = "ollama server not ready"
	MsgServerTimeout       = "timeout waiting for ollama server to be ready"
	MsgServerRetrying      = "server not yet ready. Retrying..."

	MsgStartOllamaBackground   = "starting ollama server in the background..."
	MsgStartOllamaFailed       = "failed to start ollama server"
	MsgOllamaStartedBackground = "ollama server started in the background."

	// Docker – web server
	MsgLogDirFoundFile = "found file"
	MsgProxyingRequest = "proxying request"

	// Web server error / response messages
	ErrUnsupportedServerType  = "unsupported server type"
	RespUnsupportedServerType = "500: Unsupported server type"

	ErrProxyHostPortMissing  = "proxy host or port not configured"
	RespProxyHostPortMissing = "500: Proxy host or port not configured"

	ErrInvalidProxyURL  = "invalid proxy URL"
	RespInvalidProxyURL = "500: Invalid proxy URL"

	ErrFailedProxyRequest = "failed to proxy request"
	RespFailedReachApp    = "502: Failed to reach app server"

	// API server generic messages
	MsgAwaitingResponse = "awaiting response..."

	// API server error response texts (kept identical to previous literals)
	ErrProcessRequestFile     = "Failed to process request file"
	ErrEmptyResponse          = "Empty response received, possibly due to configuration issues. Please verify: 1. Allowed route paths and HTTP methods match the incoming request. 2. Skip validations that are skipping the required resource to produce the requests. 3. Timeout settings are sufficient for long-running processes (e.g., LLM operations)."
	ErrReadResponseFile       = "Failed to read response file"
	ErrDecodeResponseContent  = "Failed to decode response content"
	ErrMarshalResponseContent = "Failed to marshal response content"

	// decodeResponseContent internal
	ErrUnmarshalRespContent = "failed to unmarshal response content"
	ErrDecodeBase64String   = "failed to decode Base64 string"

	// Resolver messages
	MsgProcessingResources   = "processing resources..."
	MsgAllResourcesProcessed = "all resources finished processing"
	MsgItemsDBEmptyRetry     = "Items database list is empty, retrying"

	// Archiver – file operations
	MsgMovingExistingToBackup = "moving existing file to backup"
	MsgFileCopiedSuccessfully = "file copied successfully"
	MsgNoDataFoundSkipping    = "no data found, skipping"

	// Archiver – package handler & others
	MsgStartingExtractionPkg  = "starting extraction of package"
	MsgExtractionCompleted    = "extraction and population completed successfully"
	MsgProjectPackaged        = "project packaged successfully"
	MsgFoundFileInFolder      = "found file %s in folder %s"
	MsgReturningFoundFilePath = "returning found file path: %s"

	// Resource compiler
	MsgResourcesCompiled = "resources compiled successfully"
	MsgProcessingPkl     = "processing .pkl"
	MsgProcessedPklFile  = "processed .pkl file"

	// Version utils
	MsgComparingVersions       = "comparing versions"
	MsgVersionComparisonResult = "version comparison result"
	MsgLatestVersionDetermined = "latest version determined"
	MsgFoundVersionDirectory   = "found version directory"

	// Workflow handler
	MsgExtractionRuntimeDone = "extraction in runtime folder completed!"
	MsgRemovedAgentDirectory = "removed existing agent directory"

	// Downloader messages
	MsgRemovedExistingLatestFile  = "removed existing file for latest version"
	MsgCheckingFileExistsDownload = "checking if file exists"
	MsgFileAlreadyExistsSkipping  = "file already exists and is non-empty, skipping download"
	MsgStartingFileDownload       = "starting file download"
	MsgDownloadComplete           = "download complete"

	// Utils files messages
	MsgWaitingForFileReady = "waiting for file to be ready..."
	MsgFileIsReady         = "file is ready!"
)

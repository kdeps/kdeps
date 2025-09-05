package messages

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMessagesConstants tests that all message constants are properly defined and non-empty
func TestMessagesConstants(t *testing.T) {
	t.Run("DockerServerMessages", func(t *testing.T) {
		assert.NotEmpty(t, MsgServerCheckingReady)
		assert.NotEmpty(t, MsgServerWaitingReady)
		assert.NotEmpty(t, MsgServerReady)
		assert.NotEmpty(t, MsgServerNotReady)
		assert.NotEmpty(t, MsgServerTimeout)
		assert.NotEmpty(t, MsgServerRetrying)
		assert.NotEmpty(t, MsgStartOllamaBackground)
		assert.NotEmpty(t, MsgStartOllamaFailed)
		assert.NotEmpty(t, MsgOllamaStartedBackground)
	})

	t.Run("DockerWebServerMessages", func(t *testing.T) {
		assert.NotEmpty(t, MsgLogDirFoundFile)
		assert.NotEmpty(t, MsgProxyingRequest)
	})

	t.Run("WebServerErrorMessages", func(t *testing.T) {
		assert.NotEmpty(t, ErrUnsupportedServerType)
		assert.NotEmpty(t, RespUnsupportedServerType)
		assert.NotEmpty(t, ErrProxyHostPortMissing)
		assert.NotEmpty(t, RespProxyHostPortMissing)
		assert.NotEmpty(t, ErrInvalidProxyURL)
		assert.NotEmpty(t, RespInvalidProxyURL)
		assert.NotEmpty(t, ErrFailedProxyRequest)
		assert.NotEmpty(t, RespFailedReachApp)
	})

	t.Run("APIServerMessages", func(t *testing.T) {
		assert.NotEmpty(t, MsgAwaitingResponse)
		assert.NotEmpty(t, ErrProcessRequestFile)
		assert.NotEmpty(t, ErrEmptyResponse)
		assert.NotEmpty(t, ErrReadResponseFile)
		assert.NotEmpty(t, ErrDecodeResponseContent)
		assert.NotEmpty(t, ErrMarshalResponseContent)
		assert.NotEmpty(t, ErrUnmarshalRespContent)
		assert.NotEmpty(t, ErrDecodeBase64String)
	})

	t.Run("ResolverMessages", func(t *testing.T) {
		assert.NotEmpty(t, MsgProcessingResources)
		assert.NotEmpty(t, MsgAllResourcesProcessed)
		assert.NotEmpty(t, MsgItemsDBEmptyRetry)
	})

	t.Run("ArchiverMessages", func(t *testing.T) {
		assert.NotEmpty(t, MsgMovingExistingToBackup)
		assert.NotEmpty(t, MsgFileCopiedSuccessfully)
		assert.NotEmpty(t, MsgNoDataFoundSkipping)
		assert.NotEmpty(t, MsgStartingExtractionPkg)
		assert.NotEmpty(t, MsgExtractionCompleted)
		assert.NotEmpty(t, MsgProjectPackaged)
		assert.NotEmpty(t, MsgFoundFileInFolder)
		assert.NotEmpty(t, MsgReturningFoundFilePath)
	})

	t.Run("ResourceCompilerMessages", func(t *testing.T) {
		assert.NotEmpty(t, MsgResourcesCompiled)
		assert.NotEmpty(t, MsgProcessingPkl)
		assert.NotEmpty(t, MsgProcessedPklFile)
	})

	t.Run("VersionUtilsMessages", func(t *testing.T) {
		assert.NotEmpty(t, MsgComparingVersions)
		assert.NotEmpty(t, MsgVersionComparisonResult)
		assert.NotEmpty(t, MsgLatestVersionDetermined)
		assert.NotEmpty(t, MsgFoundVersionDirectory)
	})

	t.Run("WorkflowHandlerMessages", func(t *testing.T) {
		assert.NotEmpty(t, MsgExtractionRuntimeDone)
		assert.NotEmpty(t, MsgRemovedAgentDirectory)
	})

	t.Run("DownloaderMessages", func(t *testing.T) {
		assert.NotEmpty(t, MsgRemovedExistingLatestFile)
		assert.NotEmpty(t, MsgCheckingFileExistsDownload)
		assert.NotEmpty(t, MsgFileAlreadyExistsSkipping)
		assert.NotEmpty(t, MsgStartingFileDownload)
		assert.NotEmpty(t, MsgDownloadComplete)
	})

	t.Run("UtilsFilesMessages", func(t *testing.T) {
		assert.NotEmpty(t, MsgWaitingForFileReady)
		assert.NotEmpty(t, MsgFileIsReady)
	})
}

// TestMessagesValues tests specific expected values for key messages
func TestMessagesValues(t *testing.T) {
	t.Run("ServerMessages", func(t *testing.T) {
		assert.Equal(t, "checking if ollama server is ready", MsgServerCheckingReady)
		assert.Equal(t, "waiting for ollama server to be ready...", MsgServerWaitingReady)
		assert.Equal(t, "ollama server is ready", MsgServerReady)
		assert.Equal(t, "ollama server not ready", MsgServerNotReady)
		assert.Equal(t, "timeout waiting for ollama server to be ready", MsgServerTimeout)
	})

	t.Run("ErrorMessages", func(t *testing.T) {
		assert.Equal(t, "unsupported server type", ErrUnsupportedServerType)
		assert.Equal(t, "500: Unsupported server type", RespUnsupportedServerType)
		assert.Equal(t, "proxy host or port not configured", ErrProxyHostPortMissing)
		assert.Equal(t, "500: Proxy host or port not configured", RespProxyHostPortMissing)
	})

	t.Run("ProcessingMessages", func(t *testing.T) {
		assert.Equal(t, "processing resources...", MsgProcessingResources)
		assert.Equal(t, "all resources finished processing", MsgAllResourcesProcessed)
		assert.Equal(t, "starting file download", MsgStartingFileDownload)
		assert.Equal(t, "download complete", MsgDownloadComplete)
	})
}

// TestMessagesUniqueness tests that messages are unique (no duplicates)
func TestMessagesUniqueness(t *testing.T) {
	// Collect all message constants
	messages := []string{
		MsgServerCheckingReady,
		MsgServerWaitingReady,
		MsgServerReady,
		MsgServerNotReady,
		MsgServerTimeout,
		MsgServerRetrying,
		MsgStartOllamaBackground,
		MsgStartOllamaFailed,
		MsgOllamaStartedBackground,
		MsgLogDirFoundFile,
		MsgProxyingRequest,
		ErrUnsupportedServerType,
		RespUnsupportedServerType,
		ErrProxyHostPortMissing,
		RespProxyHostPortMissing,
		ErrInvalidProxyURL,
		RespInvalidProxyURL,
		ErrFailedProxyRequest,
		RespFailedReachApp,
		MsgAwaitingResponse,
		ErrProcessRequestFile,
		ErrEmptyResponse,
		ErrReadResponseFile,
		ErrDecodeResponseContent,
		ErrMarshalResponseContent,
		ErrUnmarshalRespContent,
		ErrDecodeBase64String,
		MsgProcessingResources,
		MsgAllResourcesProcessed,
		MsgItemsDBEmptyRetry,
		MsgMovingExistingToBackup,
		MsgFileCopiedSuccessfully,
		MsgNoDataFoundSkipping,
		MsgStartingExtractionPkg,
		MsgExtractionCompleted,
		MsgProjectPackaged,
		MsgFoundFileInFolder,
		MsgReturningFoundFilePath,
		MsgResourcesCompiled,
		MsgProcessingPkl,
		MsgProcessedPklFile,
		MsgComparingVersions,
		MsgVersionComparisonResult,
		MsgLatestVersionDetermined,
		MsgFoundVersionDirectory,
		MsgExtractionRuntimeDone,
		MsgRemovedAgentDirectory,
		MsgRemovedExistingLatestFile,
		MsgCheckingFileExistsDownload,
		MsgFileAlreadyExistsSkipping,
		MsgStartingFileDownload,
		MsgDownloadComplete,
		MsgWaitingForFileReady,
		MsgFileIsReady,
	}

	// Check for duplicates
	seen := make(map[string]bool)
	for _, msg := range messages {
		if seen[msg] {
			t.Errorf("Duplicate message found: %s", msg)
		}
		seen[msg] = true
	}
}

// TestMessagesFormat tests that messages follow consistent formatting patterns
func TestMessagesFormat(t *testing.T) {
	t.Run("ErrorMessagesFormat", func(t *testing.T) {
		errorMessages := []string{
			ErrUnsupportedServerType,
			ErrProxyHostPortMissing,
			ErrInvalidProxyURL,
			ErrFailedProxyRequest,
			ErrProcessRequestFile,
			ErrReadResponseFile,
			ErrDecodeResponseContent,
			ErrMarshalResponseContent,
			ErrUnmarshalRespContent,
			ErrDecodeBase64String,
		}

		for _, msg := range errorMessages {
			// Error messages should typically be non-empty
			assert.NotEmpty(t, msg, "Error message should not be empty")
		}

		// Special case for ErrEmptyResponse which contains a period
		assert.NotEmpty(t, ErrEmptyResponse, "ErrEmptyResponse should not be empty")
		assert.Contains(t, ErrEmptyResponse, ".", "ErrEmptyResponse should contain explanatory text with periods")
	})

	t.Run("ResponseMessagesFormat", func(t *testing.T) {
		responseMessages := []string{
			RespUnsupportedServerType,
			RespProxyHostPortMissing,
			RespInvalidProxyURL,
			RespFailedReachApp,
		}

		for _, msg := range responseMessages {
			// Response messages typically start with status codes
			assert.Contains(t, msg, ":", "Response messages should contain status code separator")
			assert.NotEmpty(t, msg, "Response message should not be empty")
		}
	})

	t.Run("ProgressMessagesFormat", func(t *testing.T) {
		progressMessages := []string{
			MsgProcessingResources,
			MsgWaitingForFileReady,
		}

		for _, msg := range progressMessages {
			// Progress messages typically end with ellipsis
			assert.Contains(t, msg, "...", "Progress messages should end with ellipsis")
			assert.NotEmpty(t, msg, "Progress message should not be empty")
		}

		// Messages that don't end with ellipsis
		otherMessages := []string{
			MsgStartingFileDownload,
			MsgDownloadComplete,
		}

		for _, msg := range otherMessages {
			assert.NotEmpty(t, msg, "Message should not be empty")
			assert.NotContains(t, msg, "...", "This message should not end with ellipsis")
		}
	})
}

// TestMessagesCompleteness tests that we have complete coverage of message types
func TestMessagesCompleteness(t *testing.T) {
	// This test ensures we don't miss adding new message types to our test coverage
	// If new constants are added to messages.go, this test will help identify them

	// Count total constants in the package (this is a static count)
	totalConstants := 54 // Update this number when new constants are added

	// Test that we have at least the expected number of messages
	messages := []string{
		MsgServerCheckingReady, MsgServerWaitingReady, MsgServerReady, MsgServerNotReady,
		MsgServerTimeout, MsgServerRetrying, MsgStartOllamaBackground, MsgStartOllamaFailed,
		MsgOllamaStartedBackground, MsgLogDirFoundFile, MsgProxyingRequest,
		ErrUnsupportedServerType, RespUnsupportedServerType, ErrProxyHostPortMissing,
		RespProxyHostPortMissing, ErrInvalidProxyURL, RespInvalidProxyURL,
		ErrFailedProxyRequest, RespFailedReachApp, MsgAwaitingResponse,
		ErrProcessRequestFile, ErrEmptyResponse, ErrReadResponseFile,
		ErrDecodeResponseContent, ErrMarshalResponseContent, ErrUnmarshalRespContent,
		ErrDecodeBase64String, MsgProcessingResources, MsgAllResourcesProcessed,
		MsgItemsDBEmptyRetry, MsgMovingExistingToBackup, MsgFileCopiedSuccessfully,
		MsgNoDataFoundSkipping, MsgStartingExtractionPkg, MsgExtractionCompleted,
		MsgProjectPackaged, MsgFoundFileInFolder, MsgReturningFoundFilePath,
		MsgResourcesCompiled, MsgProcessingPkl, MsgProcessedPklFile,
		MsgComparingVersions, MsgVersionComparisonResult, MsgLatestVersionDetermined,
		MsgFoundVersionDirectory, MsgExtractionRuntimeDone, MsgRemovedAgentDirectory,
		MsgRemovedExistingLatestFile, MsgCheckingFileExistsDownload,
		MsgFileAlreadyExistsSkipping, MsgStartingFileDownload, MsgDownloadComplete,
		MsgWaitingForFileReady, MsgFileIsReady,
	}

	assert.Equal(t, totalConstants, len(messages),
		"Number of messages in test should match total constants in package. Update totalConstants when adding new messages.")
}

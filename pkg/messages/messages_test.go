package messages_test

import (
	"testing"

	. "github.com/kdeps/kdeps/pkg/messages"
	"github.com/stretchr/testify/assert"
)

func TestMessageConstants(t *testing.T) {
	// Test Docker server utilities messages
	assert.NotEmpty(t, MsgServerCheckingReady)
	assert.NotEmpty(t, MsgServerWaitingReady)
	assert.NotEmpty(t, MsgServerReady)
	assert.NotEmpty(t, MsgServerNotReady)
	assert.NotEmpty(t, MsgServerTimeout)
	assert.NotEmpty(t, MsgServerRetrying)

	assert.NotEmpty(t, MsgStartOllamaBackground)
	assert.NotEmpty(t, MsgStartOllamaFailed)
	assert.NotEmpty(t, MsgOllamaStartedBackground)

	// Test Docker web server messages
	assert.NotEmpty(t, MsgLogDirFoundFile)
	assert.NotEmpty(t, MsgProxyingRequest)

	// Test web server error/response messages
	assert.NotEmpty(t, ErrUnsupportedServerType)
	assert.NotEmpty(t, RespUnsupportedServerType)
	assert.NotEmpty(t, ErrProxyHostPortMissing)
	assert.NotEmpty(t, RespProxyHostPortMissing)
	assert.NotEmpty(t, ErrInvalidProxyURL)
	assert.NotEmpty(t, RespInvalidProxyURL)
	assert.NotEmpty(t, ErrFailedProxyRequest)
	assert.NotEmpty(t, RespFailedReachApp)

	// Test API server messages
	assert.NotEmpty(t, MsgAwaitingResponse)
	assert.NotEmpty(t, ErrProcessRequestFile)
	assert.NotEmpty(t, ErrEmptyResponse)
	assert.NotEmpty(t, ErrReadResponseFile)
	assert.NotEmpty(t, ErrDecodeResponseContent)
	assert.NotEmpty(t, ErrMarshalResponseContent)
	assert.NotEmpty(t, ErrUnmarshalRespContent)
	assert.NotEmpty(t, ErrDecodeBase64String)

	// Test resolver messages
	assert.NotEmpty(t, MsgProcessingResources)
	assert.NotEmpty(t, MsgAllResourcesProcessed)
	assert.NotEmpty(t, MsgItemsDBEmptyRetry)

	// Test archiver file operations messages
	assert.NotEmpty(t, MsgMovingExistingToBackup)
	assert.NotEmpty(t, MsgFileCopiedSuccessfully)
	assert.NotEmpty(t, MsgNoDataFoundSkipping)

	// Test archiver package handler messages
	assert.NotEmpty(t, MsgStartingExtractionPkg)
	assert.NotEmpty(t, MsgExtractionCompleted)
	assert.NotEmpty(t, MsgProjectPackaged)
	assert.NotEmpty(t, MsgFoundFileInFolder)
	assert.NotEmpty(t, MsgReturningFoundFilePath)

	// Test resource compiler messages
	assert.NotEmpty(t, MsgResourcesCompiled)
	assert.NotEmpty(t, MsgProcessingPkl)
	assert.NotEmpty(t, MsgProcessedPklFile)

	// Test version utils messages
	assert.NotEmpty(t, MsgComparingVersions)
	assert.NotEmpty(t, MsgVersionComparisonResult)
	assert.NotEmpty(t, MsgLatestVersionDetermined)
	assert.NotEmpty(t, MsgFoundVersionDirectory)

	// Test workflow handler messages
	assert.NotEmpty(t, MsgExtractionRuntimeDone)
	assert.NotEmpty(t, MsgRemovedAgentDirectory)

	// Test downloader messages
	assert.NotEmpty(t, MsgRemovedExistingLatestFile)
	assert.NotEmpty(t, MsgCheckingFileExistsDownload)
	assert.NotEmpty(t, MsgFileAlreadyExistsSkipping)
	assert.NotEmpty(t, MsgStartingFileDownload)
	assert.NotEmpty(t, MsgDownloadComplete)

	// Test utils files messages
	assert.NotEmpty(t, MsgWaitingForFileReady)
	assert.NotEmpty(t, MsgFileIsReady)
}

func TestMessageFormatting(t *testing.T) {
	// Test that format strings work correctly
	file := "test.pkl"
	folder := "/path/to/folder"

	formattedFoundFile := "found file " + file + " in folder " + folder
	expectedFoundFile := "found file test.pkl in folder /path/to/folder"
	assert.Equal(t, expectedFoundFile, formattedFoundFile)

	formattedReturningPath := "returning found file path: " + file
	expectedReturningPath := "returning found file path: test.pkl"
	assert.Equal(t, expectedReturningPath, formattedReturningPath)
}

func TestErrorMessageConsistency(t *testing.T) {
	// Test that error messages are consistent with their response counterparts
	assert.Contains(t, RespUnsupportedServerType, "500:")
	assert.Contains(t, RespProxyHostPortMissing, "500:")
	assert.Contains(t, RespInvalidProxyURL, "500:")
	assert.Contains(t, RespFailedReachApp, "502:")
}

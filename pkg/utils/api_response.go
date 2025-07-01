package utils

import (
	"fmt"
	"sync"

	apiserverresponse "github.com/kdeps/schema/gen/api_server_response"
)

// ErrorWithActionID represents an error with actionID captured when created
type ErrorWithActionID struct {
	Code     int
	Message  string
	ActionID string
}

// Map to hold error blocks per request ID with thread-safe access
var (
	requestErrors        = make(map[string][]*apiserverresponse.APIServerErrorsBlock)
	requestErrorsWithIDs = make(map[string][]*ErrorWithActionID)
	errorsMutex          sync.RWMutex
)

func NewAPIServerResponse(success bool, data []any, errorCode int, errorMessage string, requestID string) *apiserverresponse.APIServerResponseImpl {
	responseBlock := &apiserverresponse.APIServerResponseBlock{Data: data}

	errorsMutex.Lock()
	defer errorsMutex.Unlock()

	// If there is an error, append it to the request-specific errors slice
	if errorCode != 0 || errorMessage != "" {
		newError := &apiserverresponse.APIServerErrorsBlock{
			Code:    errorCode,
			Message: errorMessage,
		}
		requestErrors[requestID] = append(requestErrors[requestID], newError)
	}

	// Get the current errors for this request ID
	currentErrors := requestErrors[requestID]

	// Use the concrete implementation APIServerResponseImpl to return the response
	return &apiserverresponse.APIServerResponseImpl{
		Success:  success,
		Response: responseBlock,
		Errors:   &currentErrors, // Pass the pointer to the request-specific errors slice
	}
}

// NewAPIServerResponseWithActionID creates an API server response and stores error with actionID
func NewAPIServerResponseWithActionID(success bool, data []any, errorCode int, errorMessage, requestID, actionID string) *apiserverresponse.APIServerResponseImpl {
	responseBlock := &apiserverresponse.APIServerResponseBlock{Data: data}

	errorsMutex.Lock()
	defer errorsMutex.Unlock()

	// If there is an error, append it to both error collections
	if errorCode != 0 || errorMessage != "" {
		// Store in the new collection with actionID
		newErrorWithID := &ErrorWithActionID{
			Code:     errorCode,
			Message:  errorMessage,
			ActionID: actionID,
		}
		requestErrorsWithIDs[requestID] = append(requestErrorsWithIDs[requestID], newErrorWithID)

		// Also store in the old collection for backward compatibility
		newError := &apiserverresponse.APIServerErrorsBlock{
			Code:    errorCode,
			Message: errorMessage,
		}
		requestErrors[requestID] = append(requestErrors[requestID], newError)
	}

	// Get the current errors for this request ID
	currentErrors := requestErrors[requestID]

	// Use the concrete implementation APIServerResponseImpl to return the response
	return &apiserverresponse.APIServerResponseImpl{
		Success:  success,
		Response: responseBlock,
		Errors:   &currentErrors,
	}
}

// ClearRequestErrors clears the errors for a specific request ID to prevent memory leaks
func ClearRequestErrors(requestID string) {
	errorsMutex.Lock()
	defer errorsMutex.Unlock()
	delete(requestErrors, requestID)
	delete(requestErrorsWithIDs, requestID)
}

// GetRequestErrors returns a copy of the errors for a specific request ID
func GetRequestErrors(requestID string) []*apiserverresponse.APIServerErrorsBlock {
	errorsMutex.RLock()
	defer errorsMutex.RUnlock()
	errors := requestErrors[requestID]
	// Return a copy to avoid race conditions
	result := make([]*apiserverresponse.APIServerErrorsBlock, len(errors))
	copy(result, errors)
	return result
}

// GetRequestErrorsWithActionID returns a copy of the errors with actionID for a specific request ID
func GetRequestErrorsWithActionID(requestID string) []*ErrorWithActionID {
	errorsMutex.RLock()
	defer errorsMutex.RUnlock()
	errors := requestErrorsWithIDs[requestID]
	// Return a copy to avoid race conditions
	result := make([]*ErrorWithActionID, len(errors))
	copy(result, errors)
	return result
}

// MergeAllErrors ensures all accumulated errors are included in the response
// This function merges existing workflow errors with any new response errors
func MergeAllErrors(requestID string, newErrors []*apiserverresponse.APIServerErrorsBlock) []*apiserverresponse.APIServerErrorsBlock {
	errorsMutex.Lock()
	defer errorsMutex.Unlock()

	// Get existing accumulated errors
	existingErrors := requestErrors[requestID]

	// Create a map to track unique errors (by code + message combination)
	uniqueErrors := make(map[string]*apiserverresponse.APIServerErrorsBlock)

	// Add existing errors first
	for _, err := range existingErrors {
		if err != nil {
			key := fmt.Sprintf("%d:%s", err.Code, err.Message)
			uniqueErrors[key] = err
		}
	}

	// Add new errors, avoiding duplicates
	for _, err := range newErrors {
		if err != nil {
			key := fmt.Sprintf("%d:%s", err.Code, err.Message)
			uniqueErrors[key] = err
		}
	}

	// Convert back to slice
	var allErrors []*apiserverresponse.APIServerErrorsBlock
	for _, err := range uniqueErrors {
		allErrors = append(allErrors, err)
	}

	// Update the stored errors with the merged result
	requestErrors[requestID] = allErrors

	return allErrors
}

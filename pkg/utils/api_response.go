package utils

import (
	"sync"

	apiserverresponse "github.com/kdeps/schema/gen/api_server_response"
)

// Map to hold error blocks per request ID with thread-safe access
var (
	requestErrors = make(map[string][]*apiserverresponse.APIServerErrorsBlock)
	errorsMutex   sync.RWMutex
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

// ClearRequestErrors clears the errors for a specific request ID to prevent memory leaks
func ClearRequestErrors(requestID string) {
	errorsMutex.Lock()
	defer errorsMutex.Unlock()
	delete(requestErrors, requestID)
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

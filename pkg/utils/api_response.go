package utils

import (
	apiserverresponse "github.com/kdeps/schema/gen/api_server_response"
)

// PersistentErrors stores error blocks across multiple API responses.
var PersistentErrors []*apiserverresponse.APIServerErrorsBlock

func NewAPIServerResponse(success bool, data []any, errorCode int, errorMessage string) *apiserverresponse.APIServerResponseImpl {
	responseBlock := &apiserverresponse.APIServerResponseBlock{Data: data}

	// If there is an error, append it to the persistent errors slice
	if errorCode != 0 || errorMessage != "" {
		newError := &apiserverresponse.APIServerErrorsBlock{
			Code:    errorCode,
			Message: errorMessage,
		}
		PersistentErrors = append(PersistentErrors, newError)
	}

	// Use the concrete implementation APIServerResponseImpl to return the response
	return &apiserverresponse.APIServerResponseImpl{
		Success:  success,
		Response: responseBlock,
		Errors:   &PersistentErrors, // Pass the pointer to the persistent errors slice
	}
}

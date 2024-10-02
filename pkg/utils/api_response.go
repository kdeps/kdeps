package utils

import (
	apiserverresponse "github.com/kdeps/schema/gen/api_server_response"
)

func NewAPIServerResponse(success bool, data []any, errorCode int, errorMessage string) apiserverresponse.APIServerResponse {
	responseBlock := &apiserverresponse.APIServerResponseBlock{Data: data}
	var errorsBlock *apiserverresponse.APIServerErrorsBlock

	// If there is an error, create the errors block
	if errorCode != 0 || errorMessage != "" {
		errorsBlock = &apiserverresponse.APIServerErrorsBlock{
			Code:    errorCode,
			Message: errorMessage,
		}
	}

	return apiserverresponse.APIServerResponse{
		Success:  success,
		Response: responseBlock,
		Errors:   errorsBlock,
	}
}

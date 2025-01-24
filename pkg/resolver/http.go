package resolver

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/kdeps/kdeps/pkg/evaluator"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/utils"
	pklHTTP "github.com/kdeps/schema/gen/http"
	"github.com/spf13/afero"
)

func (dr *DependencyResolver) HandleHTTPClient(actionID string, httpBlock *pklHTTP.ResourceHTTPClient) error {
	errChan := make(chan error, 1) // Channel to capture the error

	go func() {
		// Decode Base64 encoded fields before processing
		if err := dr.decodeHTTPBlock(httpBlock); err != nil {
			errChan <- err // Send the error to the channel
			dr.Logger.Error("failed to decode HTTP block", "actionID", actionID, "error", err)
			return
		}

		// Proceed with processing the decoded block
		if err := dr.processHTTPBlock(actionID, httpBlock); err != nil {
			errChan <- err // Send the error to the channel
			dr.Logger.Error("failed to process HTTP block", "actionID", actionID, "error", err)
			return
		}

		errChan <- nil // Send a nil if no error occurred
	}()

	// Wait for the result from the goroutine
	err := <-errChan
	if err != nil {
		return err // Return the error to the caller
	}

	return nil
}

func (dr *DependencyResolver) processHTTPBlock(actionID string, httpBlock *pklHTTP.ResourceHTTPClient) error {
	if err := dr.DoRequest(httpBlock); err != nil {
		return err
	}

	if err := dr.AppendHTTPEntry(actionID, httpBlock); err != nil {
		return err
	}

	return nil
}

func (dr *DependencyResolver) decodeHTTPBlock(httpBlock *pklHTTP.ResourceHTTPClient) error {
	// Check if the URL is Base64 encoded before decoding
	if utils.IsBase64Encoded(httpBlock.Url) {
		decodedURL, err := utils.DecodeBase64String(httpBlock.Url)
		if err != nil {
			return fmt.Errorf("failed to decode URL: %w", err)
		}
		httpBlock.Url = decodedURL
	}

	// Decode the headers if they exist
	if httpBlock.Headers != nil {
		decodedHeaders := make(map[string]string)
		for key, value := range *httpBlock.Headers {
			// Check if the header value is Base64 encoded
			if utils.IsBase64Encoded(value) {
				decodedValue, err := utils.DecodeBase64String(value)
				if err != nil {
					return fmt.Errorf("failed to decode header %s: %w", key, err)
				}
				decodedHeaders[key] = decodedValue
			} else {
				// If not Base64 encoded, leave the value as it is
				decodedHeaders[key] = value
				dr.Logger.Debug("header value is not Base64 encoded, skipping decoding", "header", key)
			}
		}
		httpBlock.Headers = &decodedHeaders
	}

	// Decode the params if they exist
	if httpBlock.Params != nil {
		decodedParams := make(map[string]string)
		for key, value := range *httpBlock.Params {
			// Check if the param value is Base64 encoded
			if utils.IsBase64Encoded(value) {
				decodedValue, err := utils.DecodeBase64String(value)
				if err != nil {
					return fmt.Errorf("failed to decode params %s: %w", key, err)
				}
				decodedParams[key] = decodedValue
			} else {
				// If not Base64 encoded, leave the value as it is
				decodedParams[key] = value
				dr.Logger.Debug("param value is not Base64 encoded, skipping decoding", "params", key)
			}
		}
		httpBlock.Params = &decodedParams
	}

	// Decode the data field if it exists
	if httpBlock.Data != nil {
		decodedData := make([]string, len(*httpBlock.Data))
		for i, v := range *httpBlock.Data {
			// Check if the data value is Base64 encoded
			if utils.IsBase64Encoded(v) {
				decodedValue, err := utils.DecodeBase64String(v)
				if err != nil {
					return fmt.Errorf("failed to decode data at index %d: %w", i, err)
				}
				decodedData[i] = decodedValue
			} else {
				// If not Base64 encoded, leave the value as it is
				decodedData[i] = v
				dr.Logger.Debug("data value is not Base64 encoded, skipping decoding", "index", i)
			}
		}
		httpBlock.Data = &decodedData
	}

	return nil
}

func (dr *DependencyResolver) WriteResponseBodyToFile(resourceID string, responseBodyEncoded *string) (string, error) {
	// Convert resourceID to be filename friendly
	resourceIDFile := utils.ConvertToFilenameFriendly(resourceID)
	// Define the file path using the FilesDir and resource ID
	outputFilePath := filepath.Join(dr.FilesDir, resourceIDFile)

	// Ensure the ResponseBody is not nil
	if responseBodyEncoded != nil {
		// Prepare the content to write
		var content string
		if utils.IsBase64Encoded(*responseBodyEncoded) {
			// Decode the Base64-encoded ResponseBody string
			decodedResponseBody, err := utils.DecodeBase64String(*responseBodyEncoded)
			if err != nil {
				return "", fmt.Errorf("failed to decode Base64 string for resource ID: %s: %w", resourceID, err)
			}
			content = decodedResponseBody
		} else {
			// Use the ResponseBody content as-is if not Base64-encoded
			content = *responseBodyEncoded
		}

		// Write the content to the file
		err := afero.WriteFile(dr.Fs, outputFilePath, []byte(content), 0o644)
		if err != nil {
			return "", fmt.Errorf("failed to write ResponseBody to file for resource ID: %s: %w", resourceID, err)
		}
	} else {
		return "", nil
	}

	return outputFilePath, nil
}

func (dr *DependencyResolver) AppendHTTPEntry(resourceID string, newHTTPClient *pklHTTP.ResourceHTTPClient) error {
	// Define the path to the PKL file
	pklPath := filepath.Join(dr.ActionDir, "client/"+dr.RequestID+"__client_output.pkl")

	// Get the current timestamp
	newTimestamp := uint32(time.Now().UnixNano())

	// Load existing PKL data
	pklRes, err := pklHTTP.LoadFromPath(dr.Context, pklPath)
	if err != nil {
		return fmt.Errorf("failed to load PKL file: %w", err)
	}

	existingResources := *pklRes.GetResources() // Dereference the pointer to get the map

	// Check if the URL is already Base64 encoded
	var filePath, encodedURL string
	if utils.IsBase64Encoded(newHTTPClient.Url) {
		encodedURL = newHTTPClient.Url // Use the URL as it is if already Base64 encoded
	} else {
		encodedURL = utils.EncodeBase64String(newHTTPClient.Url) // Otherwise, encode it
	}

	existingResources[resourceID] = &pklHTTP.ResourceHTTPClient{
		Method:    newHTTPClient.Method,
		Url:       encodedURL, // Use either encoded or already Base64 URL
		Data:      newHTTPClient.Data,
		Headers:   newHTTPClient.Headers,
		Response:  newHTTPClient.Response,
		File:      &filePath,
		Timestamp: &newTimestamp,
	}

	// Build the new content for the PKL file in the specified format
	var pklContent strings.Builder
	pklContent.WriteString(fmt.Sprintf("extends \"package://schema.kdeps.com/core@%s#/HTTP.pkl\"\n\n", schema.SchemaVersion(dr.Context)))
	pklContent.WriteString("resources {\n")

	for id, resource := range existingResources {
		pklContent.WriteString(fmt.Sprintf("  [\"%s\"] {\n", id))
		pklContent.WriteString(fmt.Sprintf("    method = \"%s\"\n", resource.Method))
		pklContent.WriteString(fmt.Sprintf("    url = \"%s\"\n", resource.Url)) // Encoded or unchanged URL
		pklContent.WriteString(fmt.Sprintf("    timeoutDuration = %d\n", resource.TimeoutDuration))
		pklContent.WriteString(fmt.Sprintf("    timestamp = %d\n", *resource.Timestamp))

		// Base64 encode the data block
		if resource.Data != nil {
			pklContent.WriteString("    data {\n")
			for _, value := range *resource.Data {
				var encodedData string
				if utils.IsBase64Encoded(value) {
					encodedData = value // Use as it is if already Base64 encoded
				} else {
					encodedData = utils.EncodeBase64String(value) // Otherwise, encode it
				}
				pklContent.WriteString(fmt.Sprintf("      \"%s\"\n", encodedData))
			}
			pklContent.WriteString("    }\n")
		} else {
			pklContent.WriteString("    data {\"\"}\n")
		}

		// Base64 encode the headers block
		if resource.Headers != nil {
			pklContent.WriteString("    headers {\n")
			for key, value := range *resource.Headers {
				var encodedHeaderValue string
				if utils.IsBase64Encoded(value) {
					encodedHeaderValue = value // Use as it is if already Base64 encoded
				} else {
					encodedHeaderValue = utils.EncodeBase64String(value) // Otherwise, encode it
				}
				pklContent.WriteString(fmt.Sprintf("      [\"%s\"] = \"%s\"\n", key, encodedHeaderValue))
			}
			pklContent.WriteString("    }\n")
		} else {
			pklContent.WriteString("    headers {[\"\"] = \"\"\n}\n")
		}

		// Base64 encode the params block
		if resource.Params != nil {
			pklContent.WriteString("    params {\n")
			for key, value := range *resource.Params {
				var encodedParamValue string
				if utils.IsBase64Encoded(value) {
					encodedParamValue = value // Use as it is if already Base64 encoded
				} else {
					encodedParamValue = utils.EncodeBase64String(value) // Otherwise, encode it
				}
				pklContent.WriteString(fmt.Sprintf("      [\"%s\"] = \"%s\"\n", key, encodedParamValue))
			}
			pklContent.WriteString("    }\n")
		} else {
			pklContent.WriteString("    params {[\"\"] = \"\"\n}\n")
		}

		// Base64 encode the response body
		if resource.Response != nil {
			pklContent.WriteString("    response {\n")
			if resource.Response.Headers != nil {
				pklContent.WriteString("    headers {\n")
				for key, value := range *resource.Response.Headers {
					var encodedHeaderValue string
					if utils.IsBase64Encoded(value) {
						encodedHeaderValue = value // Use as it is if already Base64 encoded
					} else {
						encodedHeaderValue = utils.EncodeBase64String(value) // Otherwise, encode it
					}
					pklContent.WriteString(fmt.Sprintf("      [\"%s\"] = #\"\"\"\n%s\n\"\"\"#\n", key, encodedHeaderValue))
				}
				pklContent.WriteString("    }\n")
			}

			if resource.Response.Body != nil {
				filePath, err = dr.WriteResponseBodyToFile(resourceID, resource.Response.Body)
				if err != nil {
					return fmt.Errorf("failed to write Response Body to file: %w", err)
				}

				resource.File = &filePath

				var encodedBody string
				if utils.IsBase64Encoded(*resource.Response.Body) {
					encodedBody = *resource.Response.Body // Use as it is if already Base64 encoded
				} else {
					encodedBody = utils.EncodeBase64String(*resource.Response.Body) // Otherwise, encode it
				}
				pklContent.WriteString(fmt.Sprintf("    body = #\"\"\"\n%s\n\"\"\"#\n", encodedBody))
			}

			pklContent.WriteString("    }\n")
		} else {
			pklContent.WriteString("    response {\nheaders{[\"\"] = \"\"\n}\nbody=\"\"}\n")
		}

		pklContent.WriteString(fmt.Sprintf("    file = \"%s\"\n", filePath))

		pklContent.WriteString("  }\n")
	}

	pklContent.WriteString("}\n")

	// Write the new PKL content to the file using afero
	err = afero.WriteFile(dr.Fs, pklPath, []byte(pklContent.String()), 0o644)
	if err != nil {
		return fmt.Errorf("failed to write to PKL file: %w", err)
	}

	// Evaluate the PKL file using EvalPkl
	evaluatedContent, err := evaluator.EvalPkl(dr.Fs, dr.Context, pklPath, fmt.Sprintf("extends \"package://schema.kdeps.com/core@%s#/HTTP.pkl\"\n\n", schema.SchemaVersion(dr.Context)), dr.Logger)
	if err != nil {
		return fmt.Errorf("failed to evaluate PKL file: %w", err)
	}

	// Rebuild the PKL content with the "extends" header and evaluated content
	var finalContent strings.Builder
	finalContent.WriteString(evaluatedContent)

	// Write the final evaluated content back to the PKL file
	err = afero.WriteFile(dr.Fs, pklPath, []byte(finalContent.String()), 0o644)
	if err != nil {
		return fmt.Errorf("failed to write evaluated content to PKL file: %w", err)
	}

	return nil
}

func (dr *DependencyResolver) DoRequest(client *pklHTTP.ResourceHTTPClient) error {
	// Create the HTTP client
	timeoutDuration := 30 // default timeout
	if client.TimeoutDuration != nil {
		timeoutDuration = *client.TimeoutDuration
	}
	HTTPClient := &http.Client{
		Timeout: time.Duration(timeoutDuration) * time.Second,
	}

	// Map of methods that can have a body (POST, PUT, PATCH)
	methodsWithBody := map[string]bool{
		"POST":  true,
		"PUT":   true,
		"PATCH": true,
	}

	// Validate method
	if client.Method == "" {
		return errors.New("an HTTP method is required")
	}

	// Append query parameters to the URL if present
	if client.Params != nil {
		parsedURL, err := url.Parse(client.Url)
		if err != nil {
			return fmt.Errorf("failed to parse URL: %w", err)
		}
		query := parsedURL.Query()
		for key, value := range *client.Params {
			query.Add(key, value)
		}
		parsedURL.RawQuery = query.Encode()
		client.Url = parsedURL.String()
	}

	// Initialize request
	var req *http.Request
	var err error

	// If the method supports a body, ensure data is provided, otherwise create a request without a body
	if methodsWithBody[client.Method] {
		if client.Data == nil {
			return fmt.Errorf("%s method requires data, but none provided", client.Method)
		}
		req, err = http.NewRequestWithContext(dr.Context, client.Method, client.Url, bytes.NewBufferString(fmt.Sprintf("%s", *client.Data)))
	} else {
		req, err = http.NewRequestWithContext(dr.Context, client.Method, client.Url, nil)
	}

	// Handle error in request creation
	if err != nil {
		return fmt.Errorf("failed to create %s request: %w", client.Method, err)
	}

	// Set headers if available
	if client.Headers != nil {
		for key, value := range *client.Headers {
			req.Header.Set(key, value)
		}
	}

	// Execute the request
	resp, err := HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Initialize response fields if necessary
	if client.Response == nil {
		client.Response = &pklHTTP.ResponseBlock{}
	}
	if client.Response.Body == nil {
		client.Response.Body = new(string)
	}
	*client.Response.Body = string(body)

	// Store response headers
	if client.Response.Headers == nil {
		client.Response.Headers = new(map[string]string)
	}
	headersMap := make(map[string]string)
	for key, values := range resp.Header {
		headersMap[key] = values[0]
	}
	*client.Response.Headers = headersMap

	// Store timestamp (seconds since Unix epoch)
	timestamp := uint32(time.Now().Unix())
	client.Timestamp = &timestamp

	return nil
}

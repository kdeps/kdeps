package resolver

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"kdeps/pkg/evaluator"
	"kdeps/pkg/schema"
	"kdeps/pkg/utils"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	pklHttp "github.com/kdeps/schema/gen/http"
	"github.com/spf13/afero"
)

func (dr *DependencyResolver) HandleHttpClient(actionId string, httpBlock *pklHttp.ResourceHTTPClient) error {
	go func() error {
		// Decode Base64 encoded fields before processing
		if err := dr.decodeHttpBlock(httpBlock); err != nil {
			dr.Logger.Error("Failed to decode HTTP block", "actionId", actionId, "error", err)
			return err
		}

		// Proceed with processing the decoded block
		if err := dr.processHttpBlock(actionId, httpBlock); err != nil {
			dr.Logger.Error("Failed to process HTTP block", "actionId", actionId, "error", err)
			return err
		}

		return nil
	}()

	return nil
}

func (dr *DependencyResolver) processHttpBlock(actionId string, httpBlock *pklHttp.ResourceHTTPClient) error {
	if err := dr.DoRequest(httpBlock); err != nil {
		return err
	}

	if err := dr.AppendHttpEntry(actionId, httpBlock); err != nil {
		return err
	}

	return nil
}

func (dr *DependencyResolver) decodeHttpBlock(httpBlock *pklHttp.ResourceHTTPClient) error {
	// Check if the URL is Base64 encoded before decoding
	if utils.IsBase64Encoded(httpBlock.Url) {
		decodedUrl, err := utils.DecodeBase64String(httpBlock.Url)
		if err != nil {
			return fmt.Errorf("failed to decode URL: %w", err)
		}
		httpBlock.Url = decodedUrl
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
				dr.Logger.Info("Header value is not Base64 encoded, skipping decoding", "header", key)
			}
		}
		httpBlock.Headers = &decodedHeaders
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
				dr.Logger.Info("Data value is not Base64 encoded, skipping decoding", "index", i)
			}
		}
		httpBlock.Data = &decodedData
	}

	return nil
}

func (dr *DependencyResolver) AppendHttpEntry(resourceId string, newHttpClient *pklHttp.ResourceHTTPClient) error {
	// Define the path to the PKL file
	pklPath := filepath.Join(dr.ActionDir, "client/"+dr.RequestId+"__client_output.pkl")

	// Get the current timestamp
	newTimestamp := uint32(time.Now().UnixNano())

	// Load existing PKL data
	pklRes, err := pklHttp.LoadFromPath(*dr.Context, pklPath)
	if err != nil {
		return fmt.Errorf("failed to load PKL file: %w", err)
	}

	existingResources := *pklRes.GetResources() // Dereference the pointer to get the map

	// Check if the URL is already Base64 encoded
	var encodedUrl string
	if utils.IsBase64Encoded(newHttpClient.Url) {
		encodedUrl = newHttpClient.Url // Use the URL as it is if already Base64 encoded
	} else {
		encodedUrl = utils.EncodeBase64String(newHttpClient.Url) // Otherwise, encode it
	}

	existingResources[resourceId] = &pklHttp.ResourceHTTPClient{
		Method:    newHttpClient.Method,
		Url:       encodedUrl, // Use either encoded or already Base64 URL
		Data:      newHttpClient.Data,
		Headers:   newHttpClient.Headers,
		Response:  newHttpClient.Response,
		Timestamp: &newTimestamp,
	}

	// Build the new content for the PKL file in the specified format
	var pklContent strings.Builder
	pklContent.WriteString(fmt.Sprintf("extends \"package://schema.kdeps.com/core@%s#/Http.pkl\"\n\n", schema.SchemaVersion))
	pklContent.WriteString("resources {\n")

	for id, resource := range existingResources {
		pklContent.WriteString(fmt.Sprintf("  [\"%s\"] {\n", id))
		pklContent.WriteString(fmt.Sprintf("    method = \"%s\"\n", resource.Method))
		pklContent.WriteString(fmt.Sprintf("    url = \"%s\"\n", resource.Url)) // Encoded or unchanged URL
		pklContent.WriteString(fmt.Sprintf("    timeoutSeconds = %d\n", resource.TimeoutSeconds))
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

		pklContent.WriteString("  }\n")
	}

	pklContent.WriteString("}\n")

	// Write the new PKL content to the file using afero
	err = afero.WriteFile(dr.Fs, pklPath, []byte(pklContent.String()), 0644)
	if err != nil {
		return fmt.Errorf("failed to write to PKL file: %w", err)
	}

	// Evaluate the PKL file using EvalPkl
	evaluatedContent, err := evaluator.EvalPkl(dr.Fs, pklPath, dr.Logger)
	if err != nil {
		return fmt.Errorf("failed to evaluate PKL file: %w", err)
	}

	// Rebuild the PKL content with the "extends" header and evaluated content
	var finalContent strings.Builder
	finalContent.WriteString(fmt.Sprintf("extends \"package://schema.kdeps.com/core@%s#/Http.pkl\"\n\n", schema.SchemaVersion))
	finalContent.WriteString(evaluatedContent)

	// Write the final evaluated content back to the PKL file
	err = afero.WriteFile(dr.Fs, pklPath, []byte(finalContent.String()), 0644)
	if err != nil {
		return fmt.Errorf("failed to write evaluated content to PKL file: %w", err)
	}

	return nil
}

func (dr *DependencyResolver) DoRequest(client *pklHttp.ResourceHTTPClient) error {
	// Create the HTTP client
	timeoutSeconds := 30 // default timeout
	if client.TimeoutSeconds != nil {
		timeoutSeconds = *client.TimeoutSeconds
	}
	httpClient := &http.Client{
		Timeout: time.Duration(timeoutSeconds) * time.Second,
	}

	// Map of methods that can have a body (POST, PUT, PATCH)
	methodsWithBody := map[string]bool{
		"POST":  true,
		"PUT":   true,
		"PATCH": true,
	}

	// Validate method
	if client.Method == "" {
		return fmt.Errorf("HTTP method is required")
	}

	// Initialize request
	var req *http.Request
	var err error

	// If the method supports a body, ensure data is provided, otherwise create a request without a body
	if methodsWithBody[client.Method] {
		if client.Data == nil {
			return fmt.Errorf("%s method requires data, but none provided", client.Method)
		}
		req, err = http.NewRequest(client.Method, client.Url, bytes.NewBuffer([]byte(fmt.Sprintf("%s", *client.Data))))
	} else {
		req, err = http.NewRequest(client.Method, client.Url, nil)
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
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Initialize response fields if necessary
	if client.Response == nil {
		client.Response = &pklHttp.ResponseBlock{}
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

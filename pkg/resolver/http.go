package resolver

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	pklHttp "github.com/kdeps/schema/gen/http"
	"github.com/spf13/afero"
)

func (dr *DependencyResolver) HandleHttpClient(actionId string, httpBlock *pklHttp.ResourceHTTPClient) error {
	go func() error {
		err := dr.processHttpBlock(actionId, httpBlock)
		if err != nil {
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

func (dr *DependencyResolver) AppendHttpEntry(resourceId string, newHttpClient *pklHttp.ResourceHTTPClient) error {
	// Define the path to the PKL file
	pklPath := filepath.Join(dr.ActionDir, "client/client_output.pkl")

	// Get the current timestamp
	newTimestamp := uint32(time.Now().UnixNano())

	// Load existing PKL data
	pklRes, err := pklHttp.LoadFromPath(*dr.Context, pklPath)
	if err != nil {
		return fmt.Errorf("failed to load PKL file: %w", err)
	}

	existingResources := *pklRes.Resource // Dereference the pointer to get the map

	existingResources[resourceId] = &pklHttp.ResourceHTTPClient{
		Method:          newHttpClient.Method,
		Url:             newHttpClient.Url,
		Data:            newHttpClient.Data,
		Headers:         newHttpClient.Headers,
		ResponseData:    newHttpClient.ResponseData,
		ResponseHeaders: newHttpClient.ResponseHeaders,
		Timestamp:       &newTimestamp,
	}

	// Build the new content for the PKL file in the specified format
	var pklContent strings.Builder
	pklContent.WriteString("amends \"package://schema.kdeps.com/core@0.1.0#/HttpClient.pkl\"\n\n")
	pklContent.WriteString("resource {\n")

	for id, resource := range existingResources {
		pklContent.WriteString(fmt.Sprintf("  [\"%s\"] {\n", id))
		pklContent.WriteString(fmt.Sprintf("    method = \"%s\"\n", resource.Method))
		pklContent.WriteString(fmt.Sprintf("    url = \"%s\"\n", resource.Url))
		pklContent.WriteString(fmt.Sprintf("    timeoutSeconds = %d\n", resource.TimeoutSeconds))
		pklContent.WriteString(fmt.Sprintf("    timestamp = %d\n", *resource.Timestamp))

		if resource.Data != nil {
			pklContent.WriteString("    data {\n")
			for _, value := range *resource.Data {
				pklContent.WriteString(fmt.Sprintf("      \"\"\"\n%s\n\"\"\"\n", value))
			}
			pklContent.WriteString("    }\n")
		} else {
			pklContent.WriteString("    data {}\n") // Handle nil case for Env
		}

		if resource.Headers != nil {
			pklContent.WriteString("    headers {\n")
			for key, value := range *resource.Headers {
				pklContent.WriteString(fmt.Sprintf("      [\"%s\"] = \"%s\"\n", key, value))
			}
			pklContent.WriteString("    }\n")
		} else {
			pklContent.WriteString("    headers {}\n") // Handle nil case for Env
		}

		if resource.ResponseData != nil {
			pklContent.WriteString("    responseData {\n")
			for _, value := range *resource.ResponseData {
				pklContent.WriteString(fmt.Sprintf("      \"\"\"\n%s\n\"\"\"\n", value))
			}
			pklContent.WriteString("    }\n")
		} else {
			pklContent.WriteString("    responseData {}\n") // Handle nil case for Env
		}

		if resource.ResponseHeaders != nil {
			pklContent.WriteString("    responseHeaders {\n")
			for key, value := range *resource.ResponseHeaders {
				pklContent.WriteString(fmt.Sprintf("      [\"%s\"] = \"%s\"\n", key, value))
			}
			pklContent.WriteString("    }\n")
		} else {
			pklContent.WriteString("    env {}\n") // Handle nil case for Env
		}

		pklContent.WriteString("  }\n")
	}

	pklContent.WriteString("}\n")

	// Write the new PKL content to the file using afero
	err = afero.WriteFile(dr.Fs, pklPath, []byte(pklContent.String()), 0644)
	if err != nil {
		return fmt.Errorf("failed to write to PKL file: %w", err)
	}

	return nil
}

func (dr *DependencyResolver) DoRequest(client *pklHttp.ResourceHTTPClient) error {
	// Create the HTTP client
	httpClient := &http.Client{
		Timeout: time.Duration(*client.TimeoutSeconds) * time.Second,
	}

	// Initialize a new request variable
	var req *http.Request
	var err error

	// Handle based on the HTTP method (GET or POST)
	switch client.Method {
	case "GET":
		req, err = http.NewRequest("GET", client.Url, nil)
		if err != nil {
			return fmt.Errorf("failed to create GET request: %w", err)
		}
	case "POST":
		if client.Data == nil {
			return fmt.Errorf("POST method requires data, but none provided")
		}
		// Combine data into a string
		postData := []byte(fmt.Sprintf("%s", *client.Data))
		req, err = http.NewRequest("POST", client.Url, bytes.NewBuffer(postData))
		if err != nil {
			return fmt.Errorf("failed to create POST request: %w", err)
		}
	default:
		return fmt.Errorf("unsupported HTTP method: %s", client.Method)
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

	// Store response data
	responseDataStr := string(body)
	client.ResponseData = &[]string{responseDataStr}

	// Store response headers
	headersMap := make(map[string]string)
	for key, values := range resp.Header {
		headersMap[key] = values[0]
	}
	client.ResponseHeaders = &headersMap

	// Store timestamp (seconds since Unix epoch)
	timestamp := uint32(time.Now().Unix())
	client.Timestamp = &timestamp

	return nil
}

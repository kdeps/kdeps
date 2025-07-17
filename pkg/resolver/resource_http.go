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

	"github.com/apple/pkl-go/pkl"

	"github.com/kdeps/kdeps/pkg/utils"
	pklHTTP "github.com/kdeps/schema/gen/http"
	"github.com/spf13/afero"
)

func (dr *DependencyResolver) HandleHTTPClient(actionID string, httpBlock *pklHTTP.ResourceHTTPClient) error {
	// Canonicalize the actionID if it's a short ActionID
	canonicalActionID := actionID
	if dr.PklresHelper != nil {
		canonicalActionID = dr.PklresHelper.resolveActionID(actionID)
		if canonicalActionID != actionID {
			dr.Logger.Debug("canonicalized actionID", "original", actionID, "canonical", canonicalActionID)
		}
	}

	// Synchronously decode the HTTP block.
	if err := dr.decodeHTTPBlock(httpBlock); err != nil {
		dr.Logger.Error("failed to decode HTTP block", "actionID", canonicalActionID, "error", err)
		return err
	}

	// Process the HTTP block synchronously to ensure timestamp is updated before returning
	if err := dr.processHTTPBlock(canonicalActionID, httpBlock); err != nil {
		dr.Logger.Error("failed to process HTTP block", "actionID", canonicalActionID, "error", err)
		return err
	}

	return nil
}

func (dr *DependencyResolver) processHTTPBlock(actionID string, httpBlock *pklHTTP.ResourceHTTPClient) error {
	// Always ensure we have a timestamp, even if the request fails
	if httpBlock.Timestamp == nil {
		httpBlock.Timestamp = &pkl.Duration{
			Value: float64(time.Now().UnixNano()),
			Unit:  pkl.Nanosecond,
		}
	}

	// Attempt the HTTP request
	if err := dr.DoRequestFn(httpBlock); err != nil {
		dr.Logger.Error("HTTP request failed", "actionID", actionID, "error", err)
		// Even if the request fails, we need to update the timestamp to prevent timeout
		httpBlock.Timestamp = &pkl.Duration{
			Value: float64(time.Now().UnixNano()),
			Unit:  pkl.Nanosecond,
		}
	}

	// Write the HTTP response to output file for pklres access
	if httpBlock.Response != nil && httpBlock.Response.Body != nil {
		dr.Logger.Debug("processHTTPBlock: writing response body to file", "actionID", actionID)
		filePath, err := dr.WriteResponseBodyToFile(actionID, httpBlock.Response.Body)
		if err != nil {
			dr.Logger.Error("processHTTPBlock: failed to write response body to file", "actionID", actionID, "error", err)
			return fmt.Errorf("failed to write response body to file: %w", err)
		}
		httpBlock.File = &filePath
		dr.Logger.Debug("processHTTPBlock: wrote response body to file", "actionID", actionID, "filePath", filePath)
	}

	dr.Logger.Info("processHTTPBlock: skipping AppendHTTPEntry - using real-time pklres", "actionID", actionID)
	// Note: AppendHTTPEntry is no longer needed as we use real-time pklres access
	// The HTTP output files are directly accessible through pklres.getResourceOutput()

	// Store the complete http resource record in the PKL mapping after processing is complete
	if dr.PklresHelper != nil {
		// Create a ResourceHTTPClient object for storage
		resourceHTTP := &pklHTTP.ResourceHTTPClient{
			Method:          httpBlock.Method,
			Url:             httpBlock.Url,
			Headers:         httpBlock.Headers,
			Params:          httpBlock.Params,
			Data:            httpBlock.Data,
			Response:        httpBlock.Response,
			File:            httpBlock.File,
			ItemValues:      httpBlock.ItemValues,
			Timestamp:       httpBlock.Timestamp,
			TimeoutDuration: httpBlock.TimeoutDuration,
		}

		// Store the resource record using the new method
		if err := dr.PklresHelper.StoreResourceRecord("client", actionID, actionID, fmt.Sprintf("%+v", resourceHTTP)); err != nil {
			dr.Logger.Error("processHTTPBlock: failed to store http resource in pklres", "actionID", actionID, "error", err)
		} else {
			dr.Logger.Info("processHTTPBlock: stored http resource in pklres", "actionID", actionID)
		}
	}

	return nil
}

func (dr *DependencyResolver) decodeHTTPBlock(httpBlock *pklHTTP.ResourceHTTPClient) error {
	if utils.IsBase64Encoded(httpBlock.Url) {
		decodedURL, err := utils.DecodeBase64String(httpBlock.Url)
		if err != nil {
			return fmt.Errorf("failed to decode URL: %w", err)
		}
		httpBlock.Url = decodedURL
	}

	var err error
	httpBlock.Headers, err = utils.DecodeStringMap(httpBlock.Headers, "header")
	if err != nil {
		return err
	}

	httpBlock.Params, err = utils.DecodeStringMap(httpBlock.Params, "param")
	if err != nil {
		return err
	}

	httpBlock.Data, err = utils.DecodeStringSlice(httpBlock.Data, "data")
	return err
}

func (dr *DependencyResolver) WriteResponseBodyToFile(resourceID string, responseBodyEncoded *string) (string, error) {
	if responseBodyEncoded == nil {
		return "", nil
	}

	resourceIDFile := utils.GenerateResourceIDFilename(resourceID, dr.RequestID)
	outputFilePath := filepath.Join(dr.FilesDir, resourceIDFile)

	content, err := utils.DecodeBase64IfNeeded(*responseBodyEncoded)
	if err != nil {
		return "", fmt.Errorf("failed to decode response body: %w", err)
	}

	if err := afero.WriteFile(dr.Fs, outputFilePath, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}
	return outputFilePath, nil
}

// AppendHTTPEntry has been removed as it's no longer needed.
// We now use real-time pklres access through getResourceOutput() instead of storing PKL content.

func encodeResponseHeaders(response *pklHTTP.ResponseBlock) string {
	if response == nil || response.Headers == nil {
		return "    Headers {[\"\"] = \"\"}\n"
	}
	var builder strings.Builder
	builder.WriteString("    Headers {\n")
	for k, v := range *response.Headers {
		builder.WriteString(fmt.Sprintf("      [\"%s\"] = #\"\"\"\n%s\n\"\"\"#\n", k, utils.EncodeValue(v)))
	}
	builder.WriteString("    }\n")
	return builder.String()
}

func encodeResponseBody(response *pklHTTP.ResponseBlock, dr *DependencyResolver, resourceID string) string {
	if response == nil || response.Body == nil {
		return "    Body=\"\"\n"
	}
	if _, err := dr.WriteResponseBodyToFile(resourceID, response.Body); err != nil {
		dr.Logger.Fatalf("unable to write HTTP response body to file for resource %s", resourceID)
	}
	return fmt.Sprintf("    Body = #\"\"\"\n%s\n\"\"\"#\n", utils.EncodeValue(*response.Body))
}

func (dr *DependencyResolver) DoRequest(client *pklHTTP.ResourceHTTPClient) error {
	// Validate required parameters
	if client == nil {
		return errors.New("nil HTTP client configuration")
	}
	if client.Method == "" {
		return errors.New("HTTP method required")
	}
	if client.Url == "" {
		return errors.New("URL cannot be empty")
	}

	// Configure timeout with proper duration handling
	httpClient := &http.Client{
		Timeout: func() time.Duration {
			switch {
			case dr.DefaultTimeoutSec > 0:
				return time.Duration(dr.DefaultTimeoutSec) * time.Second
			case dr.DefaultTimeoutSec == 0:
				return 0 // unlimited
			case client.TimeoutDuration != nil:
				return client.TimeoutDuration.GoDuration()
			default:
				return 30 * time.Second
			}
		}(),
		Transport: &http.Transport{
			DisableCompression: false,
			DisableKeepAlives:  false,
			MaxIdleConns:       10,
			IdleConnTimeout:    90 * time.Second,
		},
	}

	// Parse and clean the URL first to handle any URL encoding issues
	parsedURL, err := url.Parse(client.Url)
	if err != nil {
		return fmt.Errorf("invalid URL %q: %w", client.Url, err)
	}

	// URL-encode the path component to handle spaces and special characters
	// This fixes issues where PKL interpolation creates URLs with unencoded characters
	if parsedURL.Path != "" && strings.Contains(parsedURL.Path, " ") {
		// Split path to preserve the base path and encode only the last segment
		pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
		if len(pathParts) > 0 {
			// URL-encode the last path segment (which typically contains the search term)
			lastSegment := pathParts[len(pathParts)-1]
			pathParts[len(pathParts)-1] = url.PathEscape(lastSegment)
			parsedURL.Path = "/" + strings.Join(pathParts, "/")
		}
	}

	// Process query parameters
	if client.Params != nil {
		query := parsedURL.Query()
		for k, v := range *client.Params {
			query.Add(k, v)
		}
		parsedURL.RawQuery = query.Encode()
	}

	client.Url = parsedURL.String()

	// Debug: Log the final URL being requested
	dr.Logger.Info("HTTP request URL", "url", client.Url, "method", client.Method)

	// Handle request body
	var reqBody io.Reader
	if isMethodWithBody(client.Method) {
		if client.Data == nil {
			return fmt.Errorf("HTTP %s requires request body", client.Method)
		}
		reqBody = bytes.NewBufferString(strings.Join(*client.Data, ""))
	}

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(dr.Context, client.Method, client.Url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create %s request to %q: %w",
			client.Method, client.Url, err)
	}

	// Set headers
	if client.Headers != nil {
		for k, v := range *client.Headers {
			req.Header.Set(k, v)
		}
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "pkl-http-client/1.0")
	}

	// Initialize response object if needed
	if client.Response == nil {
		client.Response = &pklHTTP.ResponseBlock{}
	}

	// Execute request
	resp, err := httpClient.Do(req)
	if err != nil {
		// Even if the request fails, we need to set a response to prevent timeout
		errorBody := fmt.Sprintf(`{"error": "HTTP request failed", "message": "%s", "url": "%s"}`, err.Error(), client.Url)
		client.Response.Body = &errorBody

		// Set error headers
		errorHeaders := map[string]string{
			"Content-Type": "application/json",
			"X-Error":      "true",
		}
		client.Response.Headers = &errorHeaders

		// Set timestamp
		timestamp := pkl.Duration{
			Value: float64(time.Now().UnixNano()),
			Unit:  pkl.Nanosecond,
		}
		client.Timestamp = &timestamp

		return fmt.Errorf("request to %q failed: %w", client.Url, err)
	}
	defer resp.Body.Close()

	// Read response body with size limit
	maxBodySize := int64(10 * 1024 * 1024) // 10MB
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		// Even if reading body fails, we still have a response
		errorBody := fmt.Sprintf(`{"error": "Failed to read response body", "message": "%s"}`, err.Error())
		client.Response.Body = &errorBody
	} else {
		// Store response data
		bodyStr := string(body)
		client.Response.Body = &bodyStr
	}

	// Store response headers (only first values)
	headers := make(map[string]string)
	for k, v := range resp.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}
	client.Response.Headers = &headers

	// Set timestamp using proper type
	timestamp := pkl.Duration{
		Value: float64(time.Now().UnixNano()),
		Unit:  pkl.Nanosecond,
	}
	client.Timestamp = &timestamp


	return nil
}

// Helper function to check if HTTP method supports body.
func isMethodWithBody(method string) bool {
	switch strings.ToUpper(method) {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

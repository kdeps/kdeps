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

	"github.com/kdeps/kdeps/pkg/schema"
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

	// Always append the entry, regardless of success/failure
	return dr.AppendHTTPEntry(actionID, httpBlock)
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

func (dr *DependencyResolver) AppendHTTPEntry(resourceID string, client *pklHTTP.ResourceHTTPClient) error {
	// Canonicalize the resourceID for storage
	canonicalResourceID := resourceID
	if dr.PklresHelper != nil {
		canonicalResourceID = dr.PklresHelper.resolveActionID(resourceID)
	}
	dr.Logger.Debug("AppendHTTPEntry: storing resource", "resourceID", resourceID, "canonicalResourceID", canonicalResourceID)

	// Retrieve existing http resources from pklres
	existingContent, err := dr.PklresHelper.retrievePklContent("client", "")
	if err != nil {
		// If no existing content, start with empty resources
		existingContent = ""
		dr.Logger.Debug("AppendHTTPEntry: no existing content found", "resourceID", resourceID, "canonicalResourceID", canonicalResourceID)
	}

	// Parse existing resources or create new map
	existingResources := make(map[string]*pklHTTP.ResourceHTTPClient)
	if existingContent != "" {
		// For now, we'll create a simple empty structure since we're storing individual resources
		// In a more sophisticated implementation, we'd parse the existing content
		existingResources = make(map[string]*pklHTTP.ResourceHTTPClient)
		dr.Logger.Debug("AppendHTTPEntry: found existing content", "resourceID", resourceID, "canonicalResourceID", canonicalResourceID, "contentLength", len(existingContent))
	}

	// Prepare file path and write response body to file
	var filePath string
	if client.Response != nil && client.Response.Body != nil {
		filePath, err = dr.WriteResponseBodyToFile(resourceID, client.Response.Body)
		if err != nil {
			return fmt.Errorf("failed to write response body to file: %w", err)
		}
		client.File = &filePath
		dr.Logger.Debug("AppendHTTPEntry: wrote response body to file", "resourceID", resourceID, "canonicalResourceID", canonicalResourceID, "filePath", filePath)
	}

	encodedURL := utils.EncodeValue(client.Url)

	timestamp := client.Timestamp
	if timestamp == nil {
		timestamp = &pkl.Duration{
			Value: float64(time.Now().UnixNano()),
			Unit:  pkl.Nanosecond,
		}
		dr.Logger.Debug("AppendHTTPEntry: created new timestamp", "resourceID", resourceID, "canonicalResourceID", canonicalResourceID, "timestamp", timestamp.Value)
	} else {
		dr.Logger.Debug("AppendHTTPEntry: using existing timestamp", "resourceID", resourceID, "canonicalResourceID", canonicalResourceID, "timestamp", timestamp.Value)
	}

	existingResources[resourceID] = &pklHTTP.ResourceHTTPClient{
		Method:          client.Method,
		Url:             encodedURL,
		Data:            client.Data,
		Headers:         client.Headers,
		Response:        client.Response,
		File:            &filePath,
		Timestamp:       timestamp,
		TimeoutDuration: client.TimeoutDuration,
		ItemValues:      client.ItemValues,
	}

	// Clear all client resources before storing new one (for debug)
	if dr.PklresHelper != nil {
		dr.PklresHelper.clearResourceType("client")
		dr.Logger.Debug("AppendHTTPEntry: cleared existing client resources", "resourceID", resourceID, "canonicalResourceID", canonicalResourceID)
	}

	// Store the PKL content using pklres (no JSON, no custom serialization)
	var pklContent strings.Builder
	pklContent.WriteString(fmt.Sprintf("extends \"package://schema.kdeps.com/core@%s#/HTTP.pkl\"\n\n", schema.SchemaVersion(dr.Context)))
	pklContent.WriteString("Resources {\n")

	timeoutDuration := dr.DefaultTimeoutSec

	for _, res := range existingResources {
		blockKey := canonicalResourceID
		dr.Logger.Debug("AppendHTTPEntry: PKL block key", "blockKey", blockKey, "resourceID", resourceID, "canonicalResourceID", canonicalResourceID)
		pklContent.WriteString(fmt.Sprintf("  [\"%s\"] {\n", blockKey))
		pklContent.WriteString(fmt.Sprintf("    Method = \"%s\"\n", res.Method))
		pklContent.WriteString(fmt.Sprintf("    Url = \"%s\"\n", res.Url))
		pklContent.WriteString(fmt.Sprintf("    TimeoutDuration = %d.s\n", timeoutDuration))
		if res.Timestamp != nil {
			pklContent.WriteString(fmt.Sprintf("    Timestamp = %g.%s\n", res.Timestamp.Value, res.Timestamp.Unit.String()))
			dr.Logger.Debug("AppendHTTPEntry: writing timestamp to PKL", "resourceID", resourceID, "canonicalResourceID", canonicalResourceID, "timestamp", res.Timestamp.Value, "unit", res.Timestamp.Unit.String())
		}
		pklContent.WriteString("    Data ")
		pklContent.WriteString(utils.EncodePklSlice(res.Data))
		pklContent.WriteString("    Headers ")
		pklContent.WriteString(utils.EncodePklMap(res.Headers))
		pklContent.WriteString("    Params ")
		pklContent.WriteString(utils.EncodePklMap(res.Params))
		pklContent.WriteString("    Response {\n")
		pklContent.WriteString(encodeResponseHeaders(res.Response))
		pklContent.WriteString(encodeResponseBody(res.Response, dr, blockKey))
		pklContent.WriteString("    }\n")
		pklContent.WriteString(fmt.Sprintf("    File = \"%s\"\n", *res.File))
		pklContent.WriteString("    ItemValues ")
		if res.ItemValues != nil && len(*res.ItemValues) > 0 {
			pklContent.WriteString(utils.EncodePklSlice(res.ItemValues))
		} else {
			pklContent.WriteString("{}\n")
		}
		pklContent.WriteString("  }\n")
	}
	pklContent.WriteString("}\n")

	// Log the PKL content for debugging
	dr.Logger.Info("AppendHTTPEntry: PKL content to be stored", "resourceID", resourceID, "canonicalResourceID", canonicalResourceID, "pklContent", pklContent.String())

	// Store the PKL content using pklres instead of writing to file
	dr.Logger.Debug("AppendHTTPEntry: storing PKL content", "resourceID", resourceID, "canonicalResourceID", canonicalResourceID, "contentLength", pklContent.Len())
	if err := dr.PklresHelper.storePklContent("client", canonicalResourceID, pklContent.String()); err != nil {
		dr.Logger.Error("AppendHTTPEntry: failed to store PKL content", "resourceID", resourceID, "canonicalResourceID", canonicalResourceID, "error", err)
		return fmt.Errorf("failed to store PKL content in pklres: %w", err)
	}

	// List all keys after storing
	if dr.PklresHelper != nil {
		resources, _ := dr.PklresHelper.retrieveAllResourcesForType("client")
		dr.Logger.Debug("AppendHTTPEntry: keys after store", "keys", resources, "resourceID", resourceID, "canonicalResourceID", canonicalResourceID)
	}

	dr.Logger.Debug("AppendHTTPEntry: successfully stored resource", "resourceID", resourceID, "canonicalResourceID", canonicalResourceID)
	return nil
}

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

	// Process query parameters
	if client.Params != nil {
		parsedURL, err := url.Parse(client.Url)
		if err != nil {
			return fmt.Errorf("invalid URL %q: %w", client.Url, err)
		}
		query := parsedURL.Query()
		for k, v := range *client.Params {
			query.Add(k, v)
		}
		parsedURL.RawQuery = query.Encode()
		client.Url = parsedURL.String()
	}

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

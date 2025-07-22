package resolver

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/apple/pkl-go/pkl"

	pklHTTP "github.com/kdeps/schema/gen/http"
	pklResource "github.com/kdeps/schema/gen/resource"
)

func (dr *DependencyResolver) HandleHTTPClient(actionID string, httpBlock *pklHTTP.ResourceHTTPClient) error {
	dr.Logger.Info("HandleHTTPClient: ENTRY", "actionID", actionID, "httpBlock_nil", httpBlock == nil)
	if httpBlock != nil {
		dr.Logger.Info("HandleHTTPClient: httpBlock fields", "actionID", actionID, "url", httpBlock.Url, "method", httpBlock.Method)
	}
	dr.Logger.Debug("HandleHTTPClient: called", "actionID", actionID, "PklresHelper_nil", dr.PklresHelper == nil)

	// Canonicalize the actionID if it's a short ActionID
	canonicalActionID := actionID
	if dr.PklresHelper != nil {
		canonicalActionID = dr.PklresHelper.resolveActionID(actionID)
		if canonicalActionID != actionID {
			dr.Logger.Debug("canonicalized actionID", "original", actionID, "canonical", canonicalActionID)
		}
	}

	// Reload the HTTP resource to ensure PKL templates are evaluated after dependencies are processed
	// This ensures that PKL template expressions like \(client.responseBody("clientResource")) have access to dependency data
	if err := dr.reloadHTTPResourceWithDependencies(canonicalActionID, httpBlock); err != nil {
		dr.Logger.Warn("failed to reload HTTP resource, continuing with original", "actionID", canonicalActionID, "error", err)
	}

	// Process the HTTP block synchronously to ensure timestamp is updated before returning
	if err := dr.processHTTPBlock(canonicalActionID, httpBlock); err != nil {
		dr.Logger.Error("failed to process HTTP block", "actionID", canonicalActionID, "error", err)
		return err
	}

	return nil
}

// reloadHTTPResourceWithDependencies reloads the HTTP resource to ensure PKL templates are evaluated after dependencies
func (dr *DependencyResolver) reloadHTTPResourceWithDependencies(actionID string, httpBlock *pklHTTP.ResourceHTTPClient) error {
	dr.Logger.Debug("reloadHTTPResourceWithDependencies: reloading HTTP resource for fresh template evaluation", "actionID", actionID)

	// Find the resource file path for this actionID
	resourceFile := ""
	for _, resInterface := range dr.Resources {
		if res, ok := resInterface.(ResourceNodeEntry); ok {
			if res.ActionID == actionID {
				resourceFile = res.File
				break
			}
		}
	}

	if resourceFile == "" {
		return fmt.Errorf("could not find resource file for actionID: %s", actionID)
	}

	dr.Logger.Debug("reloadHTTPResourceWithDependencies: found resource file", "actionID", actionID, "file", resourceFile)

	// Reload the HTTP resource with fresh PKL template evaluation
	// Load as generic Resource since the HTTP resource extends Resource.pkl, not HTTP.pkl
	var reloadedResource interface{}
	var err error
	if dr.APIServerMode {
		reloadedResource, err = dr.LoadResourceWithRequestContextFn(dr.Context, resourceFile, Resource)
	} else {
		reloadedResource, err = dr.LoadResourceFn(dr.Context, resourceFile, Resource)
	}

	if err != nil {
		return fmt.Errorf("failed to reload HTTP resource: %w", err)
	}

	// Cast to generic Resource first
	reloadedGenericResource, ok := reloadedResource.(pklResource.Resource)
	if !ok {
		return fmt.Errorf("failed to cast reloaded resource to generic Resource")
	}

	// Extract the HTTP block from the reloaded resource
	if reloadedRun := reloadedGenericResource.GetRun(); reloadedRun != nil && reloadedRun.HTTPClient != nil {
		reloadedHTTP := reloadedRun.HTTPClient

		// Update the httpBlock with the reloaded values that contain fresh template evaluation
		if reloadedHTTP.Url != "" {
			httpBlock.Url = reloadedHTTP.Url
			dr.Logger.Debug("reloadHTTPResourceWithDependencies: updated URL from reloaded resource", "actionID", actionID)
		}

		if reloadedHTTP.Method != "" {
			httpBlock.Method = reloadedHTTP.Method
			dr.Logger.Debug("reloadHTTPResourceWithDependencies: updated method from reloaded resource", "actionID", actionID)
		}

		if reloadedHTTP.Headers != nil {
			httpBlock.Headers = reloadedHTTP.Headers
			dr.Logger.Debug("reloadHTTPResourceWithDependencies: updated headers from reloaded resource", "actionID", actionID)
		}

		if reloadedHTTP.Params != nil {
			httpBlock.Params = reloadedHTTP.Params
			dr.Logger.Debug("reloadHTTPResourceWithDependencies: updated params from reloaded resource", "actionID", actionID)
		}

		if reloadedHTTP.Data != nil {
			httpBlock.Data = reloadedHTTP.Data
			dr.Logger.Debug("reloadHTTPResourceWithDependencies: updated data from reloaded resource", "actionID", actionID)
		}
	}

	dr.Logger.Info("reloadHTTPResourceWithDependencies: successfully reloaded HTTP resource with fresh template evaluation", "actionID", actionID)
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

	// Store comprehensive resource data in pklres using batch operations for better performance
	if dr.PklresHelper != nil {
		attributes := make(map[string]string)

		// Collect all attributes for batch operation
		attributes["url"] = httpBlock.Url

		if httpBlock.Method != "" {
			attributes["method"] = httpBlock.Method
		}

		if httpBlock.Data != nil && len(*httpBlock.Data) > 0 {
			if dataJSON, err := json.Marshal(*httpBlock.Data); err == nil {
				attributes["data"] = string(dataJSON)
			} else {
				dr.Logger.Error("failed to marshal data", "actionID", actionID, "error", err)
			}
		}

		if httpBlock.Headers != nil && len(*httpBlock.Headers) > 0 {
			if headersJSON, err := json.Marshal(*httpBlock.Headers); err == nil {
				attributes["headers"] = string(headersJSON)
			} else {
				dr.Logger.Error("failed to marshal headers", "actionID", actionID, "error", err)
			}
		}

		if httpBlock.Params != nil && len(*httpBlock.Params) > 0 {
			if paramsJSON, err := json.Marshal(*httpBlock.Params); err == nil {
				attributes["params"] = string(paramsJSON)
			} else {
				dr.Logger.Error("failed to marshal params", "actionID", actionID, "error", err)
			}
		}

		if httpBlock.Response != nil {
			if httpBlock.Response.Body != nil {
				attributes["response"] = *httpBlock.Response.Body
			}

			if httpBlock.Response.Headers != nil && len(*httpBlock.Response.Headers) > 0 {
				if responseHeadersJSON, err := json.Marshal(*httpBlock.Response.Headers); err == nil {
					attributes["responseHeaders"] = string(responseHeadersJSON)
				} else {
					dr.Logger.Error("failed to marshal responseHeaders", "actionID", actionID, "error", err)
				}
			}
		}

		if httpBlock.File != nil && *httpBlock.File != "" {
			attributes["file"] = *httpBlock.File
		}

		if httpBlock.ItemValues != nil && len(*httpBlock.ItemValues) > 0 {
			if itemValuesJSON, err := json.Marshal(*httpBlock.ItemValues); err == nil {
				attributes["itemValues"] = string(itemValuesJSON)
			} else {
				dr.Logger.Error("failed to marshal itemValues", "actionID", actionID, "error", err)
			}
		}

		if httpBlock.TimeoutDuration != nil {
			attributes["timeoutDuration"] = fmt.Sprintf("%g", httpBlock.TimeoutDuration.Value)
		}

		if httpBlock.Timestamp != nil {
			attributes["timestamp"] = fmt.Sprintf("%g", httpBlock.Timestamp.Value)
		}

		// Perform batch set operation
		if err := dr.PklresHelper.SetResourceAttributes(actionID, attributes); err != nil {
			dr.Logger.Error("failed to store HTTP resource attributes in pklres", "actionID", actionID, "error", err)
		} else {
			dr.Logger.Info("stored comprehensive HTTP resource attributes in pklres", "actionID", actionID, "attributeCount", len(attributes))
		}
	}

	return nil
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

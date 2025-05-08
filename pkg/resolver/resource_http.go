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
	"github.com/kdeps/kdeps/pkg/evaluator"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/utils"
	pklHTTP "github.com/kdeps/schema/gen/http"
	"github.com/spf13/afero"
)

func (dr *DependencyResolver) HandleHTTPClient(actionID string, httpBlock *pklHTTP.ResourceHTTPClient) error {
	// Synchronously decode the HTTP block.
	if err := dr.decodeHTTPBlock(httpBlock); err != nil {
		dr.Logger.Error("failed to decode HTTP block", "actionID", actionID, "error", err)
		return err
	}

	// Process the HTTP block asynchronously in a goroutine.
	go func(aID string, block *pklHTTP.ResourceHTTPClient) {
		if err := dr.processHTTPBlock(aID, block); err != nil {
			// Log the error; you can adjust error handling as needed.
			dr.Logger.Error("failed to process HTTP block", "actionID", aID, "error", err)
		}
	}(actionID, httpBlock)

	// Return immediately; the HTTP block is processed in the background.
	return nil
}

func (dr *DependencyResolver) processHTTPBlock(actionID string, httpBlock *pklHTTP.ResourceHTTPClient) error {
	if err := dr.DoRequest(httpBlock); err != nil {
		return err
	}
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
	pklPath := filepath.Join(dr.ActionDir, "client/"+dr.RequestID+"__client_output.pkl")

	res, err := dr.LoadResource(dr.Context, pklPath, HTTPResource)
	if err != nil {
		return fmt.Errorf("failed to load PKL: %w", err)
	}

	pklRes, ok := res.(*pklHTTP.HTTPImpl)
	if !ok {
		return errors.New("failed to cast pklRes to *pklHTTP.Resource")
	}

	resources := pklRes.GetResources()
	if resources == nil {
		emptyMap := make(map[string]*pklHTTP.ResourceHTTPClient)
		resources = &emptyMap
	}
	existingResources := *resources

	resourceIDFile := utils.GenerateResourceIDFilename(resourceID, dr.RequestID)
	filePath := filepath.Join(dr.FilesDir, resourceIDFile)

	encodedURL := client.Url
	if !utils.IsBase64Encoded(encodedURL) {
		encodedURL = utils.EncodeBase64String(encodedURL)
	}

	timeoutDuration := client.TimeoutDuration
	if timeoutDuration == nil {
		timeoutDuration = &pkl.Duration{
			Value: 60,
			Unit:  pkl.Second,
		}
	}

	timestamp := client.Timestamp
	if timestamp == nil {
		timestamp = &pkl.Duration{
			Value: float64(time.Now().Unix()),
			Unit:  pkl.Nanosecond,
		}
	}

	existingResources[resourceID] = &pklHTTP.ResourceHTTPClient{
		Method:          client.Method,
		Url:             encodedURL,
		Data:            client.Data,
		Headers:         client.Headers,
		Response:        client.Response,
		File:            &filePath,
		Timestamp:       timestamp,
		TimeoutDuration: timeoutDuration,
	}

	var pklContent strings.Builder
	pklContent.WriteString(fmt.Sprintf("extends \"package://schema.kdeps.com/core@%s#/HTTP.pkl\"\n\n", schema.SchemaVersion(dr.Context)))
	pklContent.WriteString("resources {\n")

	for id, res := range existingResources {
		pklContent.WriteString(fmt.Sprintf("  [\"%s\"] {\n", id))
		pklContent.WriteString(fmt.Sprintf("    method = \"%s\"\n", res.Method))
		pklContent.WriteString(fmt.Sprintf("    url = \"%s\"\n", res.Url))

		if res.TimeoutDuration != nil {
			pklContent.WriteString(fmt.Sprintf("    timeoutDuration = %g.%s\n", res.TimeoutDuration.Value, res.TimeoutDuration.Unit.String()))
		} else {
			pklContent.WriteString("    timeoutDuration = 60.s\n")
		}

		if res.Timestamp != nil {
			pklContent.WriteString(fmt.Sprintf("    timestamp = %g.%s\n", res.Timestamp.Value, res.Timestamp.Unit.String()))
		}

		pklContent.WriteString("    data ")
		pklContent.WriteString(utils.EncodePklSlice(res.Data))
		pklContent.WriteString("    headers ")
		pklContent.WriteString(utils.EncodePklMap(res.Headers))
		pklContent.WriteString("    params ")
		pklContent.WriteString(utils.EncodePklMap(res.Params))
		pklContent.WriteString("    response {\n")
		pklContent.WriteString(encodeResponseHeaders(res.Response))
		pklContent.WriteString(encodeResponseBody(res.Response, dr, resourceID))
		pklContent.WriteString("    }\n")
		pklContent.WriteString(fmt.Sprintf("    file = \"%s\"\n", *res.File))
		pklContent.WriteString("  }\n")
	}
	pklContent.WriteString("}\n")

	if err := afero.WriteFile(dr.Fs, pklPath, []byte(pklContent.String()), 0o644); err != nil {
		return fmt.Errorf("failed to write PKL: %w", err)
	}

	evaluatedContent, err := evaluator.EvalPkl(dr.Fs, dr.Context, pklPath,
		fmt.Sprintf("extends \"package://schema.kdeps.com/core@%s#/HTTP.pkl\"\n\n", schema.SchemaVersion(dr.Context)), dr.Logger)
	if err != nil {
		return fmt.Errorf("failed to evaluate PKL: %w", err)
	}

	return afero.WriteFile(dr.Fs, pklPath, []byte(evaluatedContent), 0o644)
}

func encodeResponseHeaders(response *pklHTTP.ResponseBlock) string {
	if response == nil || response.Headers == nil {
		return "    headers {[\"\"] = \"\"}\n"
	}
	var builder strings.Builder
	builder.WriteString("    headers {\n")
	for k, v := range *response.Headers {
		builder.WriteString(fmt.Sprintf("      [\"%s\"] = #\"\"\"\n%s\n\"\"\"#\n", k, utils.EncodeValue(v)))
	}
	builder.WriteString("    }\n")
	return builder.String()
}

func encodeResponseBody(response *pklHTTP.ResponseBlock, dr *DependencyResolver, resourceID string) string {
	if response == nil || response.Body == nil {
		return "    body=\"\"\n"
	}
	if _, err := dr.WriteResponseBodyToFile(resourceID, response.Body); err != nil {
		dr.Logger.Fatalf("unable to write HTTP response body to file for resource %s", resourceID)
	}
	return fmt.Sprintf("    body = #\"\"\"\n%s\n\"\"\"#\n", utils.EncodeValue(*response.Body))
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
	timeout := 30 * time.Second
	if client.TimeoutDuration != nil {
		timeout = client.TimeoutDuration.GoDuration()
	}

	httpClient := &http.Client{
		Timeout: timeout,
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

	// Execute request
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request to %q failed: %w", client.Url, err)
	}
	defer resp.Body.Close()

	// Read response body with size limit
	maxBodySize := int64(10 * 1024 * 1024) // 10MB
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Initialize response object if needed
	if client.Response == nil {
		client.Response = &pklHTTP.ResponseBlock{}
	}

	// Store response data
	bodyStr := string(body)
	client.Response.Body = &bodyStr
	// client.Response.StatusCode = resp.StatusCode

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
		Value: float64(time.Now().Unix()),
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

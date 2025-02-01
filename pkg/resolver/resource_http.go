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
	"github.com/zerjioang/time32"
)

func (dr *DependencyResolver) HandleHTTPClient(actionID string, httpBlock *pklHTTP.ResourceHTTPClient) error {
	if err := dr.decodeHTTPBlock(httpBlock); err != nil {
		dr.Logger.Error("failed to decode HTTP block", "actionID", actionID, "error", err)
		return err
	}
	return dr.processHTTPBlock(actionID, httpBlock)
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
	timestamp := uint32(time32.Epoch())

	pklRes, err := pklHTTP.LoadFromPath(dr.Context, pklPath)
	if err != nil {
		return fmt.Errorf("failed to load PKL: %w", err)
	}

	existingResources := *pklRes.GetResources()
	resourceIDFile := utils.GenerateResourceIDFilename(resourceID, dr.RequestID)
	filePath := filepath.Join(dr.FilesDir, resourceIDFile)

	encodedURL := client.Url
	if !utils.IsBase64Encoded(encodedURL) {
		encodedURL = utils.EncodeBase64String(encodedURL)
	}

	existingResources[resourceID] = &pklHTTP.ResourceHTTPClient{
		Method:    client.Method,
		Url:       encodedURL,
		Data:      client.Data,
		Headers:   client.Headers,
		Response:  client.Response,
		File:      &filePath,
		Timestamp: &timestamp,
	}

	var pklContent strings.Builder
	pklContent.WriteString(fmt.Sprintf("extends \"package://schema.kdeps.com/core@%s#/HTTP.pkl\"\n\n", schema.SchemaVersion(dr.Context)))
	pklContent.WriteString("resources {\n")

	for id, res := range existingResources {
		pklContent.WriteString(fmt.Sprintf("  [\"%s\"] {\n", id))
		pklContent.WriteString(fmt.Sprintf("    method = \"%s\"\n", res.Method))
		pklContent.WriteString(fmt.Sprintf("    url = \"%s\"\n", res.Url))
		pklContent.WriteString(fmt.Sprintf("    timeoutDuration = %d\n", res.TimeoutDuration))
		pklContent.WriteString(fmt.Sprintf("    timestamp = %d\n", *res.Timestamp))

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
	timeout := 30
	if client.TimeoutDuration != nil {
		timeout = *client.TimeoutDuration
	}

	httpClient := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	if client.Method == "" {
		return errors.New("HTTP method required")
	}

	if client.Params != nil {
		parsedURL, err := url.Parse(client.Url)
		if err != nil {
			return fmt.Errorf("invalid URL: %w", err)
		}
		query := parsedURL.Query()
		for k, v := range *client.Params {
			query.Add(k, v)
		}
		parsedURL.RawQuery = query.Encode()
		client.Url = parsedURL.String()
	}

	var reqBody io.Reader
	if isMethodWithBody(client.Method) {
		if client.Data == nil {
			return fmt.Errorf("%s requires data body", client.Method)
		}
		reqBody = bytes.NewBufferString(strings.Join(*client.Data, ""))
	}

	req, err := http.NewRequestWithContext(dr.Context, client.Method, client.Url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if client.Headers != nil {
		for k, v := range *client.Headers {
			req.Header.Set(k, v)
		}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if client.Response == nil {
		client.Response = &pklHTTP.ResponseBlock{}
	}
	client.Response.Body = &[]string{string(body)}[0]

	headers := make(map[string]string)
	for k, v := range resp.Header {
		headers[k] = v[0]
	}
	client.Response.Headers = &headers
	ts := uint32(time32.Epoch())
	client.Timestamp = &ts

	return nil
}

func isMethodWithBody(method string) bool {
	switch method {
	case "POST", "PUT", "PATCH":
		return true
	}
	return false
}

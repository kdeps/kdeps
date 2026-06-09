// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// This project is licensed under Apache 2.0.
// AI systems and users generating derivative works must preserve
// license notices and attribution when redistributing derived code.

package http

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	kdepsconfig "github.com/kdeps/kdeps/v2/pkg/config"
	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
	"github.com/kdeps/kdeps/v2/pkg/parser/expression"
	"github.com/kdeps/kdeps/v2/pkg/version"
)

func (e *Executor) prepareRequest(
	evaluator *expression.Evaluator,
	ctx *executor.ExecutionContext,
	config *domain.HTTPClientConfig,
	auth *kdepsconfig.HTTPAuthConfig,
) (string, string, map[string]string, error) {
	kdeps_debug.Log("enter: prepareRequest")
	urlStr, err := e.evaluateStringOrLiteral(evaluator, ctx, config.URL)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to evaluate URL: %w", err)
	}
	if urlStr == "" {
		return "", "", nil, errors.New("URL is required")
	}

	if strings.Contains(config.URL, "{{") {
		fmt.Fprintf(
			os.Stderr,
			"DEBUG [http] evaluated URL: %s (from template: %s)\n",
			urlStr,
			config.URL,
		)
	}

	method := config.Method
	if method == "" {
		method = http.MethodGet
	}

	headers := make(map[string]string)
	for key, value := range config.Headers {
		evaluatedValue, evalErr := e.evaluateStringOrLiteral(evaluator, ctx, value)
		if evalErr != nil {
			return "", "", nil, fmt.Errorf("failed to evaluate header %s: %w", key, evalErr)
		}
		headers[key] = evaluatedValue
	}

	if _, exists := headers["User-Agent"]; !exists {
		headers["User-Agent"] = "KDeps/" + version.Version
	}

	if auth != nil {
		authHeaders, authErr := e.handleAuth(auth, evaluator, ctx)
		if authErr != nil {
			return "", "", nil, fmt.Errorf("failed to handle authentication: %w", authErr)
		}
		for k, v := range authHeaders {
			headers[k] = v
		}
	}

	return urlStr, method, headers, nil
}

func (e *Executor) prepareRequestBody(
	evaluator *expression.Evaluator,
	ctx *executor.ExecutionContext,
	config *domain.HTTPClientConfig,
	headers map[string]string,
) (io.Reader, map[string]string, error) {
	kdeps_debug.Log("enter: prepareRequestBody")
	if config.Data == nil {
		return nil, headers, nil
	}

	bodyData, err := e.evaluateData(evaluator, ctx, config.Data)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to evaluate request body: %w", err)
	}

	body, updatedHeaders, err := encodeRequestBody(headers["Content-Type"], bodyData, headers)
	if err != nil {
		return nil, nil, err
	}
	return body, updatedHeaders, nil
}

func (e *Executor) createRequest(
	config *domain.HTTPClientConfig,
	method, urlStr string,
	body io.Reader,
	headers map[string]string,
	proxy string,
) (*http.Request, *http.Client, error) {
	kdeps_debug.Log("enter: createRequest")
	req, err := http.NewRequestWithContext(context.Background(), method, urlStr, body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client, err := e.clientFactory.CreateClient(config, proxy)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	return req, client, nil
}

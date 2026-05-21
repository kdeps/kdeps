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

// Package mcp implements an MCP (Model Context Protocol) server that exposes
// kdeps built-in tools, native resource-type executors, and installed components
// as callable tools over JSON-RPC 2.0 / stdio transport.
package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"

	"github.com/kdeps/kdeps/v2/pkg/tools"
)

const (
	serverName      = "kdeps"
	serverVersion   = "1.0"
	protocolVersion = "2024-11-05"

	errCodeMethodNotFound = -32601
	errCodeInvalidParams  = -32602
	errCodeInternal       = -32603
)

// serverRequest is an inbound JSON-RPC 2.0 request.
type serverRequest struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string           `json:"method"`
	Params  *json.RawMessage `json:"params,omitempty"`
}

// serverResponse is an outbound JSON-RPC 2.0 response.
type serverResponse struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Result  interface{}      `json:"result,omitempty"`
	Error   *rpcError        `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// toolSchema is the schema for a single tool in tools/list response.
type toolSchema struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema inputSchema `json:"inputSchema"`
}

type inputSchema struct {
	Type       string                    `json:"type"`
	Properties map[string]propertySchema `json:"properties"`
	Required   []string                  `json:"required,omitempty"`
}

type propertySchema struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
}

// Server is a stdio-transport MCP server backed by a tools.Registry.
type Server struct {
	registry *tools.Registry
	in       *bufio.Scanner
	out      io.Writer
}

// NewServer creates an MCP server that reads from stdin and writes to stdout.
func NewServer(registry *tools.Registry) *Server {
	return &Server{
		registry: registry,
		in:       bufio.NewScanner(os.Stdin),
		out:      os.Stdout,
	}
}

// NewServerWithIO creates an MCP server with explicit I/O (for testing).
func NewServerWithIO(registry *tools.Registry, in io.Reader, out io.Writer) *Server {
	return &Server{
		registry: registry,
		in:       bufio.NewScanner(in),
		out:      out,
	}
}

// Serve reads JSON-RPC 2.0 requests from stdin until EOF.
func (s *Server) Serve() error {
	kdeps_debug.Log("enter: Serve")
	for s.in.Scan() {
		line := s.in.Text()
		if line == "" {
			continue
		}
		var req serverRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			// Ignore malformed lines.
			continue
		}
		resp := s.handle(req)
		if resp == nil {
			continue
		}
		if writeErr := s.write(resp); writeErr != nil {
			return writeErr
		}
	}
	return s.in.Err()
}

func (s *Server) handle(req serverRequest) *serverResponse {
	kdeps_debug.Log("enter: handle")
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "notifications/initialized":
		return nil // no response for notifications
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(req)
	default:
		return &serverResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &rpcError{Code: errCodeMethodNotFound, Message: fmt.Sprintf("method not found: %s", req.Method)},
		}
	}
}

func (s *Server) handleInitialize(req serverRequest) *serverResponse {
	return &serverResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"protocolVersion": protocolVersion,
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    serverName,
				"version": serverVersion,
			},
		},
	}
}

func (s *Server) handleToolsList(req serverRequest) *serverResponse {
	allTools := s.registry.List()
	schemas := make([]toolSchema, 0, len(allTools))
	for _, t := range allTools {
		props := map[string]propertySchema{}
		required := []string{}
		for name, param := range t.Parameters {
			props[name] = propertySchema{
				Type:        param.Type,
				Description: param.Description,
				Enum:        param.Enum,
			}
			if param.Required {
				required = append(required, name)
			}
		}
		schemas = append(schemas, toolSchema{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: inputSchema{
				Type:       "object",
				Properties: props,
				Required:   required,
			},
		})
	}
	return &serverResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  map[string]interface{}{"tools": schemas},
	}
}

func (s *Server) handleToolsCall(req serverRequest) *serverResponse {
	if req.Params == nil {
		return s.errResp(req.ID, errCodeInvalidParams, "missing params")
	}
	var params struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return s.errResp(req.ID, errCodeInvalidParams, err.Error())
	}

	t := s.registry.Get(params.Name)
	if t == nil {
		return s.errResp(req.ID, errCodeMethodNotFound, fmt.Sprintf("tool not found: %s", params.Name))
	}
	if t.Execute == nil {
		return s.errResp(req.ID, errCodeInternal, fmt.Sprintf("tool %q has no executor registered", params.Name))
	}

	result, err := t.Execute(params.Arguments)
	if err != nil {
		return &serverResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"content": []map[string]interface{}{
					{"type": "text", "text": err.Error()},
				},
				"isError": true,
			},
		}
	}
	return &serverResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": result},
			},
		},
	}
}

func (s *Server) errResp(id *json.RawMessage, code int, msg string) *serverResponse {
	return &serverResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &rpcError{Code: code, Message: msg},
	}
}

func (s *Server) write(resp *serverResponse) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("marshal response: %w", err)
	}
	data = append(data, '\n')
	_, err = s.out.Write(data)
	return err
}

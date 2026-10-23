// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package selftest_test

import (
	"context"
	"encoding/json"
	stdhttp "net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/selftest"
)

// jsonHandler answers every request with a fixed JSON body.
func jsonHandler(body string) stdhttp.Handler {
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Test", "yes")
		_, _ = w.Write([]byte(body))
	})
}

func runCases(t *testing.T, handler stdhttp.Handler, tests []domain.TestCase, opts selftest.Options) []selftest.Result {
	t.Helper()
	return selftest.Run(context.Background(), handler, tests, opts)
}

func TestRun_Passes(t *testing.T) {
	status := 200
	test := domain.TestCase{
		Name:    "answers",
		Request: domain.TestRequest{Path: "/x"},
		Assert: domain.TestAssert{
			Status:  status,
			Headers: map[string]string{"X-Test": "ye"},
			Body: &domain.TestBodyAssert{
				Contains: `"ok"`,
				JSONPath: []domain.TestJSONPath{
					{Path: "$.ok", Equals: true},
					{Path: "$.count", Equals: 2},
					{Path: "$.items[1]", Equals: "b"},
					{Path: "$.items[0]", Contains: "a"},
					{Path: "$.missing", Exists: boolPtr(false)},
				},
			},
		},
	}
	results := runCases(t, jsonHandler(`{"ok":true,"count":2,"items":["aa","b"]}`),
		[]domain.TestCase{test}, selftest.Options{})
	require.Len(t, results, 1)
	assert.True(t, results[0].Passed(), "unexpected failure: %v", results[0].Err)
	assert.Equal(t, 200, results[0].Status)
}

func TestRun_ReportsFailures(t *testing.T) {
	cases := []struct {
		name string
		test domain.TestCase
		want string
	}{
		{
			name: "status",
			test: domain.TestCase{
				Name:    "status",
				Request: domain.TestRequest{Path: "/"},
				Assert:  domain.TestAssert{Status: 404},
			},
			want: "expected status 404, got 200",
		},
		{
			name: "contains",
			test: domain.TestCase{
				Name:    "contains",
				Request: domain.TestRequest{Path: "/"},
				Assert:  domain.TestAssert{Body: &domain.TestBodyAssert{Contains: "nope"}},
			},
			want: "does not contain",
		},
		{
			name: "equals",
			test: domain.TestCase{
				Name:    "equals",
				Request: domain.TestRequest{Path: "/"},
				Assert:  domain.TestAssert{Body: &domain.TestBodyAssert{Equals: "nope"}},
			},
			want: "does not equal",
		},
		{
			name: "header",
			test: domain.TestCase{
				Name:    "header",
				Request: domain.TestRequest{Path: "/"},
				Assert:  domain.TestAssert{Headers: map[string]string{"X-Test": "no"}},
			},
			want: "header",
		},
		{
			name: "jsonPath value",
			test: domain.TestCase{
				Name:    "jsonPath",
				Request: domain.TestRequest{Path: "/"},
				Assert: domain.TestAssert{Body: &domain.TestBodyAssert{
					JSONPath: []domain.TestJSONPath{{Path: "$.ok", Equals: false}},
				}},
			},
			want: "expected false",
		},
		{
			name: "jsonPath missing",
			test: domain.TestCase{
				Name:    "missing",
				Request: domain.TestRequest{Path: "/"},
				Assert: domain.TestAssert{Body: &domain.TestBodyAssert{
					JSONPath: []domain.TestJSONPath{{Path: "$.absent", Equals: 1}},
				}},
			},
			want: "path not found",
		},
		{
			name: "jsonPath exists",
			test: domain.TestCase{
				Name:    "exists",
				Request: domain.TestRequest{Path: "/"},
				Assert: domain.TestAssert{Body: &domain.TestBodyAssert{
					JSONPath: []domain.TestJSONPath{{Path: "$.absent", Exists: boolPtr(true)}},
				}},
			},
			want: "expected exists=true",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			results := runCases(t, jsonHandler(`{"ok":true}`), []domain.TestCase{tc.test}, selftest.Options{})
			require.Len(t, results, 1)
			require.Error(t, results[0].Err)
			assert.Contains(t, results[0].Err.Error(), tc.want)
			assert.NotEmpty(t, results[0].Body)
		})
	}
}

func TestRun_NonJSONBodyWithJSONPath(t *testing.T) {
	test := domain.TestCase{
		Name:    "not-json",
		Request: domain.TestRequest{Path: "/"},
		Assert: domain.TestAssert{Body: &domain.TestBodyAssert{
			JSONPath: []domain.TestJSONPath{{Path: "$.ok", Equals: true}},
		}},
	}
	results := runCases(t, jsonHandler("plain text"), []domain.TestCase{test}, selftest.Options{})
	require.Len(t, results, 1)
	require.Error(t, results[0].Err)
	assert.Contains(t, results[0].Err.Error(), "not JSON")
}

func TestRun_SendsRequest(t *testing.T) {
	var got struct {
		method string
		path   string
		query  string
		header string
		auth   string
		body   map[string]interface{}
	}
	handler := stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		got.method = r.Method
		got.path = r.URL.Path
		got.query = r.URL.Query().Get("q")
		got.header = r.Header.Get("X-Case")
		got.auth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&got.body)
		w.WriteHeader(stdhttp.StatusCreated)
	})
	test := domain.TestCase{
		Name: "request-shape",
		Request: domain.TestRequest{
			Method:  "post",
			Path:    "api/v1/chat",
			Query:   map[string]string{"q": "hello"},
			Headers: map[string]string{"X-Case": "one"},
			Body:    map[string]interface{}{"message": "hi"},
		},
		Assert: domain.TestAssert{Status: 201},
	}
	results := runCases(t, handler, []domain.TestCase{test}, selftest.Options{AuthToken: "tok"})
	require.True(t, results[0].Passed(), "unexpected failure: %v", results[0].Err)

	assert.Equal(t, stdhttp.MethodPost, got.method)
	assert.Equal(t, "/api/v1/chat", got.path)
	assert.Equal(t, "hello", got.query)
	assert.Equal(t, "one", got.header)
	assert.Equal(t, "Bearer tok", got.auth)
	assert.Equal(t, "hi", got.body["message"])
}

func TestRun_ExplicitCredentialWins(t *testing.T) {
	var auth string
	handler := stdhttp.HandlerFunc(func(_ stdhttp.ResponseWriter, r *stdhttp.Request) {
		auth = r.Header.Get("Authorization")
	})
	test := domain.TestCase{
		Name:    "own-credential",
		Request: domain.TestRequest{Path: "/", Headers: map[string]string{"Authorization": "Bearer own"}},
	}
	results := runCases(t, handler, []domain.TestCase{test}, selftest.Options{AuthToken: "tok"})
	require.True(t, results[0].Passed())
	assert.Equal(t, "Bearer own", auth)
}

func TestRun_DefaultMethodIsGet(t *testing.T) {
	var method string
	handler := stdhttp.HandlerFunc(func(_ stdhttp.ResponseWriter, r *stdhttp.Request) {
		method = r.Method
	})
	test := domain.TestCase{Name: "default-get", Request: domain.TestRequest{Path: "/"}}
	results := runCases(t, handler, []domain.TestCase{test}, selftest.Options{})
	require.True(t, results[0].Passed())
	assert.Equal(t, stdhttp.MethodGet, method)
}

func TestRun_Timeout(t *testing.T) {
	release := make(chan struct{})
	handler := stdhttp.HandlerFunc(func(_ stdhttp.ResponseWriter, _ *stdhttp.Request) {
		<-release
	})
	t.Cleanup(func() { close(release) })

	test := domain.TestCase{
		Name:    "slow",
		Request: domain.TestRequest{Path: "/"},
		Timeout: "10ms",
	}
	started := time.Now()
	results := runCases(t, handler, []domain.TestCase{test}, selftest.Options{})
	require.Len(t, results, 1)
	require.Error(t, results[0].Err)
	assert.Contains(t, results[0].Err.Error(), "timed out")
	assert.Less(t, time.Since(started), 2*time.Second)
}

func TestRun_InvalidTimeout(t *testing.T) {
	test := domain.TestCase{Name: "bad-timeout", Request: domain.TestRequest{Path: "/"}, Timeout: "soon"}
	results := runCases(t, jsonHandler("{}"), []domain.TestCase{test}, selftest.Options{})
	require.Len(t, results, 1)
	require.Error(t, results[0].Err)
	assert.Contains(t, results[0].Err.Error(), "invalid timeout")
}

func TestRun_StopOnFailure(t *testing.T) {
	fail := domain.TestCase{
		Name:    "fails",
		Request: domain.TestRequest{Path: "/"},
		Assert:  domain.TestAssert{Status: 500},
	}
	after := domain.TestCase{Name: "after", Request: domain.TestRequest{Path: "/"}}

	stopped := runCases(t, jsonHandler("{}"), []domain.TestCase{fail, after},
		selftest.Options{StopOnFailure: true})
	require.Len(t, stopped, 1)

	all := runCases(t, jsonHandler("{}"), []domain.TestCase{fail, after}, selftest.Options{})
	require.Len(t, all, 2)
	assert.True(t, all[1].Passed())
}

func TestRun_DefaultTimeoutApplied(t *testing.T) {
	handler := stdhttp.HandlerFunc(func(_ stdhttp.ResponseWriter, _ *stdhttp.Request) {
		time.Sleep(50 * time.Millisecond)
	})
	test := domain.TestCase{Name: "inherits-default", Request: domain.TestRequest{Path: "/"}}
	results := runCases(t, handler, []domain.TestCase{test},
		selftest.Options{DefaultTimeout: time.Second})
	require.True(t, results[0].Passed(), "unexpected failure: %v", results[0].Err)
}

func boolPtr(v bool) *bool { return &v }

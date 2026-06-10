// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package docker_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dockerpkg "github.com/kdeps/kdeps/v2/pkg/infra/docker"
)

func mockDockerHost(t *testing.T) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	host := strings.TrimPrefix(server.URL, "http://")
	t.Setenv("DOCKER_HOST", "tcp://"+host)
}

func TestNewClient_ShortMode_Success(t *testing.T) {
	mockDockerHost(t)

	client, err := dockerpkg.NewClient()
	require.NoError(t, err)
	require.NotNil(t, client)
	require.NotNil(t, client.Cli)
}

func TestNewBuilder_ShortMode_Success(t *testing.T) {
	mockDockerHost(t)

	builder, err := dockerpkg.NewBuilderWithOS("alpine")
	require.NoError(t, err)
	require.NotNil(t, builder)
	assert.Equal(t, "alpine", builder.BaseOS)
}

func TestNewBuilderWithOS_ShortMode_Success(t *testing.T) {
	mockDockerHost(t)

	builder, err := dockerpkg.NewBuilderWithOS("ubuntu")
	require.NoError(t, err)
	require.NotNil(t, builder)
	assert.Equal(t, "ubuntu", builder.BaseOS)
}

func TestNewBuilderWithOS_ShortMode_InvalidOS(t *testing.T) {
	mockDockerHost(t)

	_, err := dockerpkg.NewBuilderWithOS("fedora")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid base OS")
}

func TestClient_Close_ShortMode(t *testing.T) {
	c := newMockDockerClient(t, func(_ *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, map[string]string{"status": "ok"}), nil
	})

	err := c.Close()
	require.NoError(t, err)
}

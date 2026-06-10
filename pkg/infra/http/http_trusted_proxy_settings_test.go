// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package http

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func TestTrustedProxiesFromSettings_mergesAPIServerAndWebServer(t *testing.T) {
	settings := domain.WorkflowSettings{
		APIServer: &domain.APIServerConfig{
			TrustedProxies: []string{"10.0.0.0/8"},
		},
		WebServer: &domain.WebServerConfig{
			TrustedProxies: []string{"172.16.0.0/12"},
		},
	}
	assert.Equal(t, []string{"10.0.0.0/8", "172.16.0.0/12"}, trustedProxiesFromSettings(settings))
}

func TestTrustedProxiesFromSettings_webServerOnly(t *testing.T) {
	settings := domain.WorkflowSettings{
		WebServer: &domain.WebServerConfig{
			TrustedProxies: []string{"192.168.1.1"},
		},
	}
	assert.Equal(t, []string{"192.168.1.1"}, trustedProxiesFromSettings(settings))
}

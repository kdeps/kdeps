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
	"os"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
)

// applySecurityMiddleware wires auth, rate-limit, and body-limit middleware
// from the workflow's APIServer config.
func (s *Server) applySecurityMiddleware() {
	kdeps_debug.Log("enter: applySecurityMiddleware")
	if s.Workflow == nil || s.Workflow.Settings.APIServer == nil {
		return
	}
	api := s.Workflow.Settings.APIServer
	trustedProxies := trustedProxiesFromSettings(s.Workflow.Settings)
	if invalid := invalidTrustedProxyEntries(trustedProxies); len(invalid) > 0 && s.logger != nil {
		s.logger.Warn("ignored invalid trustedProxies entries", "entries", invalid)
	}
	if token := os.Getenv("KDEPS_API_AUTH_TOKEN"); token != "" {
		s.Router.Use(AuthMiddleware(token))
	} else if s.logger != nil {
		s.logger.Warn(
			"API server running without authentication; set KDEPS_API_AUTH_TOKEN or api_auth_token in ~/.kdeps/config.yaml",
		)
	}
	if api.RateLimit != nil && api.RateLimit.RequestsPerMinute > 0 {
		burst := api.RateLimit.Burst
		if burst <= 0 {
			burst = api.RateLimit.RequestsPerMinute
		}
		s.Router.Use(RateLimitMiddleware(api.RateLimit.RequestsPerMinute, burst, trustedProxies))
	}
	maxBody := api.MaxBodyBytes
	if maxBody <= 0 {
		maxBody = MaxUploadSize
	}
	s.Router.Use(BodyLimitMiddleware(maxBody))
	if api.MaxConcurrent > 0 {
		s.Router.Use(ConcurrentLimitMiddleware(api.MaxConcurrent))
	}
}

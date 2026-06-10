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

package validator_test

import (
	"testing"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/validator"
)

// TestWorkflowValidator_ValidateSettings_WebServerPort tests the WebServer port validation branch.
func TestWorkflowValidator_ValidateSettings_WebServerPort(t *testing.T) {
	v := validator.NewWorkflowValidator(nil)

	// Valid WebServer port
	t.Run("valid WebServer port", func(t *testing.T) {
		w := &domain.Workflow{
			Settings: domain.WorkflowSettings{
				WebServer: &domain.WebServerConfig{PortNum: 8080},
			},
		}
		if err := v.ValidateSettings(w); err != nil {
			t.Errorf("unexpected error for valid WebServer port: %v", err)
		}
	})

	// Invalid WebServer port (too high)
	t.Run("invalid WebServer port too high", func(t *testing.T) {
		w := &domain.Workflow{
			Settings: domain.WorkflowSettings{
				WebServer: &domain.WebServerConfig{PortNum: 70000},
			},
		}
		if err := v.ValidateSettings(w); err == nil {
			t.Error("expected error for invalid WebServer port, got nil")
		}
	})

	// Invalid WebServer port (too low)
	t.Run("invalid WebServer port too low", func(t *testing.T) {
		w := &domain.Workflow{
			Settings: domain.WorkflowSettings{
				WebServer: &domain.WebServerConfig{PortNum: -1},
			},
		}
		if err := v.ValidateSettings(w); err == nil {
			t.Error("expected error for invalid WebServer port, got nil")
		}
	})
}

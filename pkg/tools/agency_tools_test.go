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

package tools_test

import (
	"testing"

	"github.com/kdeps/kdeps/v3/pkg/domain"
	"github.com/kdeps/kdeps/v3/pkg/tools"
)

// ---- marshalResult branches via AgentToolDef ----

// ---- AgentToolDefWithName ----

// A workflow tool takes the keys its resources read with input(): a model
// called a greeter that reads input('name') with {"input": "Ada"} because
// "input" was the only parameter offered, and the workflow got no name.
func TestAgentToolParameters_FromInputReferences(t *testing.T) {
	greeter := &domain.Workflow{Resources: []*domain.Resource{{
		ActionID: "greet",
		Component: &domain.ComponentCallConfig{Name: "exec", With: map[string]interface{}{
			"command": `echo "Hello, {{ input('name') }}"`,
		}},
	}}}
	params := tools.AgentToolParameters(greeter)
	if _, ok := params["name"]; !ok || len(params) != 1 || !params["name"].Required {
		t.Fatalf("greeter params = %+v", params)
	}

	two := &domain.Workflow{Resources: []*domain.Resource{{
		ActionID: "ask",
		Chat:     &domain.ChatConfig{Prompt: `{{ input("question") }} for {{ input('user') }}`},
	}}}
	params = tools.AgentToolParameters(two)
	if len(params) != 2 || params["question"].Required || params["user"].Required {
		t.Fatalf("two-key params = %+v", params)
	}

	for _, wf := range []*domain.Workflow{nil, {}} {
		if _, ok := tools.AgentToolParameters(wf)["input"]; !ok {
			t.Fatalf("no input() references: the single input parameter")
		}
	}
}

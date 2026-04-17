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

package validator_test

import (
	"testing"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/validator"
)

func newValidator() *validator.WorkflowValidator {
	return &validator.WorkflowValidator{}
}

func TestValidateTelephonyActionConfig_EmptyAction(t *testing.T) {
	v := newValidator()
	err := v.ValidateTelephonyActionConfig(&domain.TelephonyActionConfig{Action: ""})
	if err == nil {
		t.Error("expected error for empty action")
	}
}

func TestValidateTelephonyActionConfig_InvalidAction(t *testing.T) {
	v := newValidator()
	err := v.ValidateTelephonyActionConfig(&domain.TelephonyActionConfig{Action: "spin"})
	if err == nil {
		t.Error("expected error for invalid action 'spin'")
	}
}

func TestValidateTelephonyActionConfig_ValidActions(t *testing.T) {
	validActions := []struct {
		action string
		config *domain.TelephonyActionConfig
	}{
		{"answer", &domain.TelephonyActionConfig{Action: "answer"}},
		{"say", &domain.TelephonyActionConfig{Action: "say"}},
		{"ask", &domain.TelephonyActionConfig{Action: "ask", Limit: 1}},
		{"menu", &domain.TelephonyActionConfig{
			Action:  "menu",
			Matches: []domain.TelephonyMatch{{Keys: []string{"1"}}},
		}},
		{"dial", &domain.TelephonyActionConfig{Action: "dial", To: []string{"+1111"}}},
		{"record", &domain.TelephonyActionConfig{Action: "record"}},
		{"mute", &domain.TelephonyActionConfig{Action: "mute"}},
		{"unmute", &domain.TelephonyActionConfig{Action: "unmute"}},
		{"hangup", &domain.TelephonyActionConfig{Action: "hangup"}},
		{"reject", &domain.TelephonyActionConfig{Action: "reject"}},
		{"redirect", &domain.TelephonyActionConfig{Action: "redirect"}},
	}
	v := newValidator()
	for _, tc := range validActions {
		t.Run(tc.action, func(t *testing.T) {
			if err := v.ValidateTelephonyActionConfig(tc.config); err != nil {
				t.Errorf("action=%q: unexpected error: %v", tc.action, err)
			}
		})
	}
}

func TestValidateTelephonyActionConfig_AskNoGrammar(t *testing.T) {
	v := newValidator()
	err := v.ValidateTelephonyActionConfig(&domain.TelephonyActionConfig{
		Action: "ask",
		// no grammar, grammarUrl, limit, or matches
	})
	if err == nil {
		t.Error("expected error for ask without grammar/limit")
	}
}

func TestValidateTelephonyActionConfig_AskWithGrammar(t *testing.T) {
	v := newValidator()
	err := v.ValidateTelephonyActionConfig(&domain.TelephonyActionConfig{
		Action:  "ask",
		Grammar: "<grammar>test</grammar>",
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateTelephonyActionConfig_AskWithGrammarURL(t *testing.T) {
	v := newValidator()
	err := v.ValidateTelephonyActionConfig(&domain.TelephonyActionConfig{
		Action:     "ask",
		GrammarURL: "https://example.com/g.grxml",
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateTelephonyActionConfig_AskWithLimit(t *testing.T) {
	v := newValidator()
	err := v.ValidateTelephonyActionConfig(&domain.TelephonyActionConfig{
		Action: "ask",
		Limit:  6,
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateTelephonyActionConfig_MenuNoGrammar(t *testing.T) {
	v := newValidator()
	err := v.ValidateTelephonyActionConfig(&domain.TelephonyActionConfig{
		Action: "menu",
		// no matches or grammar
	})
	if err == nil {
		t.Error("expected error for menu without matches/grammar")
	}
}

func TestValidateTelephonyActionConfig_MenuWithMatches(t *testing.T) {
	v := newValidator()
	err := v.ValidateTelephonyActionConfig(&domain.TelephonyActionConfig{
		Action: "menu",
		Matches: []domain.TelephonyMatch{
			{Keys: []string{"1"}},
			{Keys: []string{"2"}},
		},
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateTelephonyActionConfig_DialNoTarget(t *testing.T) {
	v := newValidator()
	err := v.ValidateTelephonyActionConfig(&domain.TelephonyActionConfig{
		Action: "dial",
		// no To
	})
	if err == nil {
		t.Error("expected error for dial without To")
	}
}

func TestValidateTelephonyActionConfig_DialWithTarget(t *testing.T) {
	v := newValidator()
	err := v.ValidateTelephonyActionConfig(&domain.TelephonyActionConfig{
		Action: "dial",
		To:     []string{"sip:agent@pbx.example.com"},
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- Integration: ValidateResource with telephony ---------------------------

func TestValidateResourceTelephony(t *testing.T) {
	res := &domain.Resource{
		APIVersion: "kdeps.io/v1",
		Kind:       "Resource",
		Metadata:   domain.ResourceMetadata{ActionID: "answer", Name: "Answer"},
		Run: domain.RunConfig{
			Telephony: &domain.TelephonyActionConfig{Action: "answer"},
		},
	}
	wf := &domain.Workflow{
		APIVersion: "kdeps.io/v1",
		Kind:       "Workflow",
		Metadata:   domain.WorkflowMetadata{Name: "test", Version: "1.0.0"},
		Resources:  []*domain.Resource{res},
	}
	v := newValidator()
	if err := v.ValidateResource(res, wf); err != nil {
		t.Errorf("ValidateResource: unexpected error: %v", err)
	}
}

func TestValidateResourceTelephonyInvalidAction(t *testing.T) {
	res := &domain.Resource{
		Metadata: domain.ResourceMetadata{ActionID: "bad", Name: "Bad"},
		Run: domain.RunConfig{
			Telephony: &domain.TelephonyActionConfig{Action: "bad_action"},
		},
	}
	wf := &domain.Workflow{
		Metadata:  domain.WorkflowMetadata{Name: "test", Version: "1.0.0"},
		Resources: []*domain.Resource{res},
	}
	v := newValidator()
	err := v.ValidateResource(res, wf)
	if err == nil {
		t.Error("expected error for invalid telephony action in ValidateResource")
	}
}

func TestValidateResourceTelephonyPlusPrimaryIsError(t *testing.T) {
	res := &domain.Resource{
		Metadata: domain.ResourceMetadata{ActionID: "conflict", Name: "Conflict"},
		Run: domain.RunConfig{
			Telephony: &domain.TelephonyActionConfig{Action: "hangup"},
			Exec:      &domain.ExecConfig{Command: "echo"},
		},
	}
	wf := &domain.Workflow{
		Metadata:  domain.WorkflowMetadata{Name: "test", Version: "1.0.0"},
		Resources: []*domain.Resource{res},
	}
	v := newValidator()
	err := v.ValidateResource(res, wf)
	if err == nil {
		t.Error("expected error: telephony and exec set simultaneously")
	}
}

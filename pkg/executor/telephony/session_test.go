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

package telephony_test

import (
	"testing"

	"github.com/kdeps/kdeps/v2/pkg/executor/telephony"
)

func TestNewSession(t *testing.T) {
	s := telephony.NewSession()
	if s == nil {
		t.Fatal("NewSession returned nil")
	}
	if s.Response == nil {
		t.Error("NewSession: Response should not be nil")
	}
	if s.SID != "" || s.From != "" || s.To != "" {
		t.Error("NewSession: fields should be empty")
	}
}

func TestNewSessionFromBodyFull(t *testing.T) {
	body := map[string]any{
		"CallSid":      "CA123",
		"From":         "+14155551234",
		"To":           "+18005551000",
		"CallStatus":   "in-progress",
		"Digits":       "1",
		"SpeechResult": "hello",
		"Confidence":   "0.95",
	}
	s := telephony.NewSessionFromBody(body)
	if s.SID != "CA123" {
		t.Errorf("CallID = %q, want CA123", s.SID)
	}
	if s.From != "+14155551234" {
		t.Errorf("From = %q, want +14155551234", s.From)
	}
	if s.To != "+18005551000" {
		t.Errorf("To = %q, want +18005551000", s.To)
	}
	if s.Status != "in-progress" {
		t.Errorf("Status = %q, want in-progress", s.Status)
	}
	if s.Digits != "1" {
		t.Errorf("Digits = %q, want 1", s.Digits)
	}
	if s.SpeechResult != "hello" {
		t.Errorf("SpeechResult = %q, want hello", s.SpeechResult)
	}
	if s.Confidence != 0.95 {
		t.Errorf("Confidence = %v, want 0.95", s.Confidence)
	}
}

func TestNewSessionFromBodyConfidenceFloat(t *testing.T) {
	body := map[string]any{
		"Confidence": 0.87,
	}
	s := telephony.NewSessionFromBody(body)
	if s.Confidence != 0.87 {
		t.Errorf("Confidence = %v, want 0.87", s.Confidence)
	}
}

func TestNewSessionFromBodyNil(t *testing.T) {
	s := telephony.NewSessionFromBody(nil)
	if s == nil {
		t.Fatal("NewSessionFromBody(nil) returned nil")
	}
	if s.SID != "" {
		t.Errorf("expected empty CallID for nil body")
	}
}

func TestNewSessionFromBodyEmpty(t *testing.T) {
	s := telephony.NewSessionFromBody(map[string]any{})
	if s.From != "" || s.To != "" {
		t.Errorf("expected empty From/To for empty body")
	}
}

func TestSessionAccessors(t *testing.T) {
	s := telephony.NewSessionFromBody(map[string]any{
		"CallSid":      "CAxxx",
		"From":         "+1111",
		"To":           "+2222",
		"CallStatus":   "ringing",
		"Digits":       "5",
		"SpeechResult": "yes",
		"Confidence":   "0.9",
	})
	if s.CallID() != "CAxxx" {
		t.Errorf("CallId() = %q", s.CallID())
	}
	if s.FromNumber() != "+1111" {
		t.Errorf("FromNumber() = %q", s.FromNumber())
	}
	if s.ToNumber() != "+2222" {
		t.Errorf("ToNumber() = %q", s.ToNumber())
	}
	if s.CallStatus() != "ringing" {
		t.Errorf("CallStatus() = %q", s.CallStatus())
	}
	if s.LastDigits() != "5" {
		t.Errorf("LastDigits() = %q", s.LastDigits())
	}
	if s.LastSpeech() != "yes" {
		t.Errorf("LastSpeech() = %q", s.LastSpeech())
	}
	if s.LastConfidence() != 0.9 {
		t.Errorf("LastConfidence() = %v", s.LastConfidence())
	}
}

func TestSessionResultStatusNoResult(t *testing.T) {
	s := telephony.NewSession()
	if s.ResultStatus() != "" {
		t.Errorf("ResultStatus() should be empty before ask/menu, got %q", s.ResultStatus())
	}
}

func TestSessionResultStatus(t *testing.T) {
	s := telephony.NewSession()
	s.LastResult = &telephony.Result{Status: telephony.StatusMatch, Utterance: "2"}
	if s.ResultStatus() != "match" {
		t.Errorf("ResultStatus() = %q, want match", s.ResultStatus())
	}
	if s.Utterance() != "2" {
		t.Errorf("Utterance() = %q, want 2", s.Utterance())
	}
	if !s.IsMatch() {
		t.Error("IsMatch() should be true")
	}
}

func TestSessionIsMatchNoResult(t *testing.T) {
	s := telephony.NewSession()
	if s.IsMatch() {
		t.Error("IsMatch() should be false when no result")
	}
	if s.Utterance() != "" {
		t.Errorf("Utterance() should be empty when no result")
	}
}

func TestSessionTwiML(t *testing.T) {
	s := telephony.NewSession()
	s.Response.AddSay("hello", "")
	twiml := s.TwiML()
	if twiml == "" {
		t.Error("TwiML() should not be empty")
	}
}

func TestSessionToEnvMap(t *testing.T) {
	s := telephony.NewSessionFromBody(map[string]any{
		"CallSid": "CA999",
		"From":    "+13005551111",
		"To":      "+18005559999",
	})
	env := s.ToEnvMap()

	// Check all expected keys exist and are callable.
	tests := []struct {
		key  string
		want any
	}{
		{"callId", "CA999"},
		{"from", "+13005551111"},
		{"to", "+18005559999"},
		{"status", ""},
		{"utterance", ""},
		{"digits", ""},
		{"speech", ""},
	}
	for _, tc := range tests {
		fn, ok := env[tc.key]
		if !ok {
			t.Errorf("env missing key %q", tc.key)
			continue
		}
		if f, isStr := fn.(func() string); isStr {
			if got := f(); got != tc.want {
				t.Errorf("env[%q]() = %q, want %q", tc.key, got, tc.want)
			}
		}
	}

	// Check confidence returns float64.
	confFn, ok := env["confidence"].(func() float64)
	if !ok {
		t.Error("env[confidence] should be func() float64")
	} else if confFn() != 0 {
		t.Errorf("confidence = %v, want 0", confFn())
	}

	// Check match returns bool.
	matchFn, ok := env["match"].(func() bool)
	if !ok {
		t.Error("env[match] should be func() bool")
	} else if matchFn() {
		t.Error("match should be false initially")
	}
}

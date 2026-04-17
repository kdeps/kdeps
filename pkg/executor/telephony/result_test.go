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

func TestResultMatch(t *testing.T) {
	tests := []struct {
		name   string
		status telephony.ResultStatus
		want   bool
	}{
		{"match", telephony.StatusMatch, true},
		{"nomatch", telephony.StatusNoMatch, false},
		{"noinput", telephony.StatusNoInput, false},
		{"hangup", telephony.StatusHangup, false},
		{"stop", telephony.StatusStop, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := &telephony.Result{Status: tc.status}
			if r.Match() != tc.want {
				t.Errorf("Match() = %v, want %v", r.Match(), tc.want)
			}
		})
	}
}

func TestResultString(t *testing.T) {
	r := &telephony.Result{Utterance: "123"}
	if r.String() != "123" {
		t.Errorf("String() = %q, want %q", r.String(), "123")
	}
}

func TestResultToMap(t *testing.T) {
	r := &telephony.Result{
		Status:         telephony.StatusMatch,
		Mode:           telephony.ModeDTMF,
		Utterance:      "1",
		Interpretation: "1",
		Confidence:     1.0,
	}
	m := r.ToMap()
	if m["status"] != "match" {
		t.Errorf("status = %v, want match", m["status"])
	}
	if m["mode"] != "dtmf" {
		t.Errorf("mode = %v, want dtmf", m["mode"])
	}
	if m["utterance"] != "1" {
		t.Errorf("utterance = %v, want 1", m["utterance"])
	}
	if m["match"] != true {
		t.Errorf("match = %v, want true", m["match"])
	}
	if m["confidence"] != 1.0 {
		t.Errorf("confidence = %v, want 1.0", m["confidence"])
	}
}

func TestResultToMapNoMatch(t *testing.T) {
	r := &telephony.Result{Status: telephony.StatusNoInput}
	m := r.ToMap()
	if m["match"] != false {
		t.Errorf("match = %v, want false", m["match"])
	}
	if m["status"] != "noinput" {
		t.Errorf("status = %v, want noinput", m["status"])
	}
}

func TestStatusConstants(t *testing.T) {
	if telephony.StatusMatch != "match" {
		t.Error("StatusMatch should be 'match'")
	}
	if telephony.StatusNoMatch != "nomatch" {
		t.Error("StatusNoMatch should be 'nomatch'")
	}
	if telephony.StatusNoInput != "noinput" {
		t.Error("StatusNoInput should be 'noinput'")
	}
	if telephony.StatusHangup != "hangup" {
		t.Error("StatusHangup should be 'hangup'")
	}
	if telephony.StatusStop != "stop" {
		t.Error("StatusStop should be 'stop'")
	}
	if telephony.ModeDTMF != "dtmf" {
		t.Error("ModeDTMF should be 'dtmf'")
	}
	if telephony.ModeSpeech != "speech" {
		t.Error("ModeSpeech should be 'speech'")
	}
}

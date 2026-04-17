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
	"strings"
	"testing"

	"github.com/kdeps/kdeps/v2/pkg/executor/telephony"
)

func TestBuildDigitGrammar(t *testing.T) {
	g := telephony.BuildDigitGrammar(6)
	if !strings.Contains(g, `repeat="1-6"`) {
		t.Errorf("expected repeat=1-6 in grammar, got: %s", g)
	}
	if !strings.Contains(g, `mode="dtmf"`) {
		t.Errorf("expected mode=dtmf in grammar")
	}
	if !strings.Contains(g, `root="digits"`) {
		t.Errorf("expected root=digits in grammar")
	}
	// Should include all DTMF digits
	for _, d := range []string{"0", "1", "9", "#", "*"} {
		if !strings.Contains(g, ">"+d+"<") {
			t.Errorf("expected digit %q in grammar", d)
		}
	}
}

func TestBuildDigitGrammarLimit1(t *testing.T) {
	g := telephony.BuildDigitGrammar(1)
	if !strings.Contains(g, `repeat="1-1"`) {
		t.Errorf("expected repeat=1-1, got: %s", g)
	}
}

func TestBuildMenuGrammar(t *testing.T) {
	g := telephony.BuildMenuGrammar([]string{"1", "2", "3"})
	if !strings.Contains(g, `root="options"`) {
		t.Errorf("expected root=options")
	}
	// Should contain tag indices 0, 1, 2
	if !strings.Contains(g, "<tag>0</tag>1") {
		t.Errorf("expected tag 0 for key 1, got: %s", g)
	}
	if !strings.Contains(g, "<tag>1</tag>2") {
		t.Errorf("expected tag 1 for key 2")
	}
	if !strings.Contains(g, "<tag>2</tag>3") {
		t.Errorf("expected tag 2 for key 3")
	}
}

func TestBuildMenuGrammarSingle(t *testing.T) {
	g := telephony.BuildMenuGrammar([]string{"9"})
	if !strings.Contains(g, "<tag>0</tag>9") {
		t.Errorf("expected single key grammar, got: %s", g)
	}
}

func TestBuildGrammarsInline(t *testing.T) {
	grammars := telephony.BuildGrammars("<grammar>test</grammar>", "", 0)
	if len(grammars) != 1 {
		t.Fatalf("expected 1 grammar, got %d", len(grammars))
	}
	if grammars[0].Value != "<grammar>test</grammar>" {
		t.Errorf("unexpected grammar value: %s", grammars[0].Value)
	}
	if grammars[0].ContentType != "application/srgs+xml" {
		t.Errorf("unexpected content type: %s", grammars[0].ContentType)
	}
	if grammars[0].URL != "" {
		t.Errorf("url should be empty for inline grammar")
	}
}

func TestBuildGrammarsURL(t *testing.T) {
	grammars := telephony.BuildGrammars("", "https://example.com/grammar.grxml", 0)
	if len(grammars) != 1 {
		t.Fatalf("expected 1 grammar, got %d", len(grammars))
	}
	if grammars[0].URL != "https://example.com/grammar.grxml" {
		t.Errorf("unexpected URL: %s", grammars[0].URL)
	}
	if grammars[0].Value != "" {
		t.Errorf("value should be empty for URL grammar")
	}
}

func TestBuildGrammarsLimit(t *testing.T) {
	grammars := telephony.BuildGrammars("", "", 4)
	if len(grammars) != 1 {
		t.Fatalf("expected 1 grammar, got %d", len(grammars))
	}
	if !strings.Contains(grammars[0].Value, `repeat="1-4"`) {
		t.Errorf("expected limit grammar with repeat=1-4, got: %s", grammars[0].Value)
	}
}

func TestBuildGrammarsBoth(t *testing.T) {
	grammars := telephony.BuildGrammars("<g/>", "https://example.com/g.grxml", 0)
	if len(grammars) != 2 {
		t.Fatalf("expected 2 grammars (inline + url), got %d", len(grammars))
	}
}

func TestBuildGrammarsEmpty(t *testing.T) {
	grammars := telephony.BuildGrammars("", "", 0)
	if len(grammars) != 0 {
		t.Errorf("expected 0 grammars when all empty, got %d", len(grammars))
	}
}

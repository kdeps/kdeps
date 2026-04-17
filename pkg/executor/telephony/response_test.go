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

func TestResponseBuilderEmpty(t *testing.T) {
	rb := telephony.NewResponseBuilder()
	twiml, err := rb.ToTwiML()
	if err != nil {
		t.Fatalf("ToTwiML error: %v", err)
	}
	if !strings.Contains(twiml, "<Response>") {
		t.Errorf("expected <Response> in empty TwiML, got: %s", twiml)
	}
	if rb.NodeCount() != 0 {
		t.Errorf("expected 0 nodes, got %d", rb.NodeCount())
	}
}

func TestResponseBuilderSay(t *testing.T) {
	rb := telephony.NewResponseBuilder()
	rb.AddSay("Hello world", "")
	twiml, err := rb.ToTwiML()
	if err != nil {
		t.Fatalf("ToTwiML error: %v", err)
	}
	if !strings.Contains(twiml, "<Say>Hello world</Say>") {
		t.Errorf("expected <Say>Hello world</Say>, got: %s", twiml)
	}
	if rb.NodeCount() != 1 {
		t.Errorf("expected 1 node")
	}
}

func TestResponseBuilderSayWithVoice(t *testing.T) {
	rb := telephony.NewResponseBuilder()
	rb.AddSay("Hi", "alice")
	twiml, err := rb.ToTwiML()
	if err != nil {
		t.Fatalf("ToTwiML error: %v", err)
	}
	if !strings.Contains(twiml, `voice="alice"`) {
		t.Errorf("expected voice=alice in TwiML, got: %s", twiml)
	}
}

func TestResponseBuilderPlay(t *testing.T) {
	rb := telephony.NewResponseBuilder()
	rb.AddPlay("https://example.com/beep.mp3")
	twiml, err := rb.ToTwiML()
	if err != nil {
		t.Fatalf("ToTwiML error: %v", err)
	}
	if !strings.Contains(twiml, "<Play>https://example.com/beep.mp3</Play>") {
		t.Errorf("expected Play node, got: %s", twiml)
	}
}

func TestResponseBuilderGatherDTMF(t *testing.T) {
	rb := telephony.NewResponseBuilder()
	rb.AddGather(telephony.GatherOptions{
		Input:     "dtmf",
		NumDigits: 1,
		Timeout:   5,
		Say:       "Press 1 for sales",
	})
	twiml, err := rb.ToTwiML()
	if err != nil {
		t.Fatalf("ToTwiML error: %v", err)
	}
	if !strings.Contains(twiml, "<Gather") {
		t.Errorf("expected <Gather> in TwiML, got: %s", twiml)
	}
	if !strings.Contains(twiml, `numDigits="1"`) {
		t.Errorf("expected numDigits=1")
	}
	if !strings.Contains(twiml, `timeout="5"`) {
		t.Errorf("expected timeout=5")
	}
	if !strings.Contains(twiml, "Press 1 for sales") {
		t.Errorf("expected say text in Gather")
	}
}

func TestResponseBuilderGatherPlay(t *testing.T) {
	rb := telephony.NewResponseBuilder()
	rb.AddGather(telephony.GatherOptions{
		Input: "dtmf",
		Audio: "https://example.com/menu.mp3",
	})
	twiml, err := rb.ToTwiML()
	if err != nil {
		t.Fatalf("ToTwiML error: %v", err)
	}
	if !strings.Contains(twiml, "<Play>https://example.com/menu.mp3</Play>") {
		t.Errorf("expected Play in Gather, got: %s", twiml)
	}
}

func TestResponseBuilderGatherFinishOnKey(t *testing.T) {
	rb := telephony.NewResponseBuilder()
	rb.AddGather(telephony.GatherOptions{
		Input:       "dtmf",
		FinishOnKey: "#",
	})
	twiml, err := rb.ToTwiML()
	if err != nil {
		t.Fatalf("ToTwiML error: %v", err)
	}
	if !strings.Contains(twiml, `finishOnKey="#"`) {
		t.Errorf("expected finishOnKey=# in TwiML, got: %s", twiml)
	}
}

func TestResponseBuilderDial(t *testing.T) {
	rb := telephony.NewResponseBuilder()
	rb.AddDial(telephony.DialOptions{
		To:      []string{"+14155551212"},
		Timeout: 30,
	})
	twiml, err := rb.ToTwiML()
	if err != nil {
		t.Fatalf("ToTwiML error: %v", err)
	}
	if !strings.Contains(twiml, "<Dial") {
		t.Errorf("expected <Dial> in TwiML, got: %s", twiml)
	}
	if !strings.Contains(twiml, "+14155551212") {
		t.Errorf("expected phone number in Dial, got: %s", twiml)
	}
	if !strings.Contains(twiml, `timeout="30"`) {
		t.Errorf("expected timeout=30 in Dial")
	}
}

func TestResponseBuilderDialMultiple(t *testing.T) {
	rb := telephony.NewResponseBuilder()
	rb.AddDial(telephony.DialOptions{
		To: []string{"+1111", "+2222"},
	})
	twiml, err := rb.ToTwiML()
	if err != nil {
		t.Fatalf("ToTwiML error: %v", err)
	}
	if !strings.Contains(twiml, "+1111") || !strings.Contains(twiml, "+2222") {
		t.Errorf("expected both numbers in Dial, got: %s", twiml)
	}
}

func TestResponseBuilderDialSIP(t *testing.T) {
	rb := telephony.NewResponseBuilder()
	rb.AddDial(telephony.DialOptions{
		To: []string{"sip:agent@pbx.example.com", "+14155551212"},
	})
	twiml, err := rb.ToTwiML()
	if err != nil {
		t.Fatalf("ToTwiML error: %v", err)
	}
	if !strings.Contains(twiml, "<Sip>sip:agent@pbx.example.com</Sip>") {
		t.Errorf("expected <Sip> element for SIP URI, got: %s", twiml)
	}
	if !strings.Contains(twiml, "<Number>+14155551212</Number>") {
		t.Errorf("expected <Number> element for E.164, got: %s", twiml)
	}
}

func TestResponseBuilderRecord(t *testing.T) {
	rb := telephony.NewResponseBuilder()
	rb.AddRecord(telephony.RecordOptions{
		MaxLength: 60,
		PlayBeep:  true,
	})
	twiml, err := rb.ToTwiML()
	if err != nil {
		t.Fatalf("ToTwiML error: %v", err)
	}
	if !strings.Contains(twiml, "<Record") {
		t.Errorf("expected <Record> in TwiML, got: %s", twiml)
	}
	if !strings.Contains(twiml, `maxLength="60"`) {
		t.Errorf("expected maxLength=60")
	}
}

func TestResponseBuilderHangup(t *testing.T) {
	rb := telephony.NewResponseBuilder()
	rb.AddHangup()
	twiml, err := rb.ToTwiML()
	if err != nil {
		t.Fatalf("ToTwiML error: %v", err)
	}
	if !strings.Contains(twiml, "<Hangup") {
		t.Errorf("expected <Hangup> in TwiML, got: %s", twiml)
	}
}

func TestResponseBuilderReject(t *testing.T) {
	rb := telephony.NewResponseBuilder()
	rb.AddReject("busy")
	twiml, err := rb.ToTwiML()
	if err != nil {
		t.Fatalf("ToTwiML error: %v", err)
	}
	if !strings.Contains(twiml, `reason="busy"`) {
		t.Errorf("expected reason=busy in Reject, got: %s", twiml)
	}
}

func TestResponseBuilderRedirect(t *testing.T) {
	rb := telephony.NewResponseBuilder()
	rb.AddRedirect("https://example.com/ivr2")
	twiml, err := rb.ToTwiML()
	if err != nil {
		t.Fatalf("ToTwiML error: %v", err)
	}
	if !strings.Contains(twiml, "<Redirect>https://example.com/ivr2</Redirect>") {
		t.Errorf("expected Redirect URL, got: %s", twiml)
	}
}

func TestResponseBuilderMuteUnmute(t *testing.T) {
	rb := telephony.NewResponseBuilder()
	rb.AddMute()
	rb.AddUnmute()
	if rb.NodeCount() != 2 {
		t.Errorf("expected 2 nodes, got %d", rb.NodeCount())
	}
	twiml, err := rb.ToTwiML()
	if err != nil {
		t.Fatalf("ToTwiML error: %v", err)
	}
	if !strings.Contains(twiml, "<Mute") {
		t.Errorf("expected <Mute> in TwiML")
	}
	if !strings.Contains(twiml, "<Unmute") {
		t.Errorf("expected <Unmute> in TwiML")
	}
}

func TestResponseBuilderXMLHeader(t *testing.T) {
	rb := telephony.NewResponseBuilder()
	twiml, err := rb.ToTwiML()
	if err != nil {
		t.Fatalf("ToTwiML error: %v", err)
	}
	if !strings.HasPrefix(twiml, "<?xml") {
		t.Errorf("expected XML header at start, got: %s", twiml[:30])
	}
}

func TestResponseBuilderNodeCount(t *testing.T) {
	rb := telephony.NewResponseBuilder()
	if rb.NodeCount() != 0 {
		t.Errorf("expected 0 nodes initially")
	}
	rb.AddSay("one", "")
	rb.AddSay("two", "")
	if rb.NodeCount() != 2 {
		t.Errorf("expected 2 nodes, got %d", rb.NodeCount())
	}
}

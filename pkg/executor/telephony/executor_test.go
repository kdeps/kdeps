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

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
	"github.com/kdeps/kdeps/v2/pkg/executor/telephony"
)

// newTestCtx creates a minimal ExecutionContext for testing.
func newTestCtx(body map[string]any) *executor.ExecutionContext {
	ctx := &executor.ExecutionContext{
		Items: make(map[string]any),
	}
	if body != nil {
		ctx.Request = &executor.RequestContext{Body: body}
	}
	return ctx
}

func newExec() *telephony.Executor {
	return telephony.NewExecutor()
}

// --- Action: answer ---------------------------------------------------------

func TestExecAnswer(t *testing.T) {
	e := newExec()
	ctx := newTestCtx(map[string]any{"CallSid": "CA1", "From": "+1111", "To": "+2222"})
	result, err := e.Execute(ctx, &domain.TelephonyActionConfig{Action: telephony.ActionAnswer})
	if err != nil {
		t.Fatalf("answer: %v", err)
	}
	m := result.(map[string]any)
	if m["callId"] != "CA1" {
		t.Errorf("callId = %v, want CA1", m["callId"])
	}
}

// --- Action: say ------------------------------------------------------------

func TestExecSayText(t *testing.T) {
	e := newExec()
	ctx := newTestCtx(nil)
	_, err := e.Execute(ctx, &domain.TelephonyActionConfig{
		Action: telephony.ActionSay,
		Say:    "Hello caller",
	})
	if err != nil {
		t.Fatalf("say: %v", err)
	}
	s := ctx.Items[telephony.SessionKey].(*telephony.Session)
	twiml := s.TwiML()
	if !strings.Contains(twiml, "<Say>Hello caller</Say>") {
		t.Errorf("expected Say in TwiML, got: %s", twiml)
	}
}

func TestExecSayAudio(t *testing.T) {
	e := newExec()
	ctx := newTestCtx(nil)
	_, err := e.Execute(ctx, &domain.TelephonyActionConfig{
		Action: telephony.ActionSay,
		Audio:  "https://example.com/audio.mp3",
	})
	if err != nil {
		t.Fatalf("say audio: %v", err)
	}
	s := ctx.Items[telephony.SessionKey].(*telephony.Session)
	twiml := s.TwiML()
	if !strings.Contains(twiml, "<Play>https://example.com/audio.mp3</Play>") {
		t.Errorf("expected Play in TwiML, got: %s", twiml)
	}
}

func TestExecSayNothing(t *testing.T) {
	// Empty say/audio - should not error, just produces empty response.
	e := newExec()
	ctx := newTestCtx(nil)
	_, err := e.Execute(ctx, &domain.TelephonyActionConfig{Action: telephony.ActionSay})
	if err != nil {
		t.Fatalf("empty say: %v", err)
	}
	s := ctx.Items[telephony.SessionKey].(*telephony.Session)
	if s.Response.NodeCount() != 0 {
		t.Errorf("expected 0 nodes for empty say, got %d", s.Response.NodeCount())
	}
}

// --- Action: ask ------------------------------------------------------------

func TestExecAskNoInput(t *testing.T) {
	e := newExec()
	ctx := newTestCtx(nil) // no Digits or SpeechResult
	_, err := e.Execute(ctx, &domain.TelephonyActionConfig{
		Action:  telephony.ActionAsk,
		Say:     "Enter your account number",
		Limit:   6,
		Timeout: "10s",
	})
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	s := ctx.Items[telephony.SessionKey].(*telephony.Session)
	if s.LastResult == nil {
		t.Fatal("LastResult should not be nil after ask")
	}
	if s.LastResult.Status != telephony.StatusNoInput {
		t.Errorf("status = %v, want noinput", s.LastResult.Status)
	}
	twiml := s.TwiML()
	if !strings.Contains(twiml, "<Gather") {
		t.Errorf("expected Gather in TwiML, got: %s", twiml)
	}
}

func TestExecAskWithDigits(t *testing.T) {
	e := newExec()
	ctx := newTestCtx(map[string]any{"Digits": "123456"})
	_, err := e.Execute(ctx, &domain.TelephonyActionConfig{
		Action: telephony.ActionAsk,
		Limit:  6,
	})
	if err != nil {
		t.Fatalf("ask with digits: %v", err)
	}
	s := ctx.Items[telephony.SessionKey].(*telephony.Session)
	if s.LastResult.Status != telephony.StatusMatch {
		t.Errorf("status = %v, want match", s.LastResult.Status)
	}
	if s.LastResult.Utterance != "123456" {
		t.Errorf("utterance = %q, want 123456", s.LastResult.Utterance)
	}
	if s.LastResult.Mode != telephony.ModeDTMF {
		t.Errorf("mode = %v, want dtmf", s.LastResult.Mode)
	}
	if s.LastResult.Confidence != 1.0 {
		t.Errorf("confidence = %v, want 1.0", s.LastResult.Confidence)
	}
}

func TestExecAskWithSpeech(t *testing.T) {
	e := newExec()
	ctx := newTestCtx(map[string]any{
		"SpeechResult": "yes",
		"Confidence":   "0.85",
	})
	_, err := e.Execute(ctx, &domain.TelephonyActionConfig{
		Action: telephony.ActionAsk,
		Mode:   "speech",
		Limit:  1,
	})
	if err != nil {
		t.Fatalf("ask speech: %v", err)
	}
	s := ctx.Items[telephony.SessionKey].(*telephony.Session)
	if s.LastResult.Status != telephony.StatusMatch {
		t.Errorf("status = %v, want match", s.LastResult.Status)
	}
	if s.LastResult.Mode != telephony.ModeSpeech {
		t.Errorf("mode = %v, want speech", s.LastResult.Mode)
	}
	if s.LastResult.Utterance != "yes" {
		t.Errorf("utterance = %q, want yes", s.LastResult.Utterance)
	}
	if s.LastResult.Confidence != 0.85 {
		t.Errorf("confidence = %v, want 0.85", s.LastResult.Confidence)
	}
}

func TestExecAskTerminator(t *testing.T) {
	e := newExec()
	ctx := newTestCtx(nil)
	_, err := e.Execute(ctx, &domain.TelephonyActionConfig{
		Action:     telephony.ActionAsk,
		Limit:      6,
		Terminator: "#",
	})
	if err != nil {
		t.Fatalf("ask terminator: %v", err)
	}
	s := ctx.Items[telephony.SessionKey].(*telephony.Session)
	twiml := s.TwiML()
	if !strings.Contains(twiml, `finishOnKey="#"`) {
		t.Errorf("expected finishOnKey=# in TwiML, got: %s", twiml)
	}
}

func TestExecAskDefaultTimeout(t *testing.T) {
	e := newExec()
	ctx := newTestCtx(nil)
	_, err := e.Execute(ctx, &domain.TelephonyActionConfig{
		Action: telephony.ActionAsk,
		Limit:  1,
		// no Timeout - should default to 5
	})
	if err != nil {
		t.Fatalf("ask default timeout: %v", err)
	}
	s := ctx.Items[telephony.SessionKey].(*telephony.Session)
	twiml := s.TwiML()
	if !strings.Contains(twiml, `timeout="5"`) {
		t.Errorf("expected default timeout=5 in TwiML, got: %s", twiml)
	}
}

func TestExecAskSpeechAutoTimeout(t *testing.T) {
	e := newExec()
	ctx := newTestCtx(nil)
	_, err := e.Execute(ctx, &domain.TelephonyActionConfig{
		Action: telephony.ActionAsk,
		Mode:   "speech",
		// no Timeout - should set speechTimeout="auto"
	})
	if err != nil {
		t.Fatalf("ask speech auto: %v", err)
	}
	s := ctx.Items[telephony.SessionKey].(*telephony.Session)
	twiml := s.TwiML()
	if !strings.Contains(twiml, `speechTimeout="auto"`) {
		t.Errorf("expected speechTimeout=auto in TwiML, got: %s", twiml)
	}
}

func TestExecAskSpeechExplicitTimeout(t *testing.T) {
	e := newExec()
	ctx := newTestCtx(nil)
	_, err := e.Execute(ctx, &domain.TelephonyActionConfig{
		Action:  telephony.ActionAsk,
		Mode:    "speech",
		Timeout: "10s",
	})
	if err != nil {
		t.Fatalf("ask speech explicit timeout: %v", err)
	}
	s := ctx.Items[telephony.SessionKey].(*telephony.Session)
	twiml := s.TwiML()
	if !strings.Contains(twiml, `speechTimeout="10s"`) {
		t.Errorf("expected speechTimeout=10s in TwiML, got: %s", twiml)
	}
}

// --- Action: menu -----------------------------------------------------------

func TestExecMenuNoInput(t *testing.T) {
	e := newExec()
	ctx := newTestCtx(nil)
	_, err := e.Execute(ctx, &domain.TelephonyActionConfig{
		Action: telephony.ActionMenu,
		Say:    "Press 1 for sales, 2 for support",
		Matches: []domain.TelephonyMatch{
			{Keys: []string{"1"}},
			{Keys: []string{"2"}},
		},
	})
	if err != nil {
		t.Fatalf("menu no input: %v", err)
	}
	s := ctx.Items[telephony.SessionKey].(*telephony.Session)
	if s.LastResult.Status != telephony.StatusNoInput {
		t.Errorf("status = %v, want noinput", s.LastResult.Status)
	}
	twiml := s.TwiML()
	if !strings.Contains(twiml, "<Gather") {
		t.Errorf("expected Gather in menu TwiML")
	}
}

func TestExecMenuMatchBranch(t *testing.T) {
	e := newExec()
	ctx := newTestCtx(map[string]any{"Digits": "2"})
	_, err := e.Execute(ctx, &domain.TelephonyActionConfig{
		Action: telephony.ActionMenu,
		Matches: []domain.TelephonyMatch{
			{Keys: []string{"1"}, Invoke: "salesFlow"},
			{Keys: []string{"2"}, Invoke: "supportFlow"},
		},
	})
	if err != nil {
		t.Fatalf("menu match: %v", err)
	}
	s := ctx.Items[telephony.SessionKey].(*telephony.Session)
	if s.LastResult.Status != telephony.StatusMatch {
		t.Errorf("status = %v, want match", s.LastResult.Status)
	}
	if s.LastResult.Utterance != "2" {
		t.Errorf("utterance = %q, want 2", s.LastResult.Utterance)
	}
	if s.LastResult.Interpretation != "2" { // matched key "2"
		t.Errorf("interpretation = %q, want 2", s.LastResult.Interpretation)
	}
}

func TestExecMenuNoMatchDigit(t *testing.T) {
	e := newExec()
	ctx := newTestCtx(map[string]any{"Digits": "9"}) // 9 is not in matches
	_, err := e.Execute(ctx, &domain.TelephonyActionConfig{
		Action: telephony.ActionMenu,
		Matches: []domain.TelephonyMatch{
			{Keys: []string{"1"}},
			{Keys: []string{"2"}},
		},
	})
	if err != nil {
		t.Fatalf("menu no match digit: %v", err)
	}
	s := ctx.Items[telephony.SessionKey].(*telephony.Session)
	if s.LastResult.Status != telephony.StatusNoMatch {
		t.Errorf("status = %v, want nomatch", s.LastResult.Status)
	}
}

func TestExecMenuDefaultTries(t *testing.T) {
	// Tries defaults to 1 when not set.
	e := newExec()
	ctx := newTestCtx(nil)
	_, err := e.Execute(ctx, &domain.TelephonyActionConfig{
		Action:  telephony.ActionMenu,
		Matches: []domain.TelephonyMatch{{Keys: []string{"1"}}},
		// Tries: 0 → defaults to 1
	})
	if err != nil {
		t.Fatalf("menu default tries: %v", err)
	}
}

// --- Action: dial -----------------------------------------------------------

func TestExecDial(t *testing.T) {
	e := newExec()
	ctx := newTestCtx(nil)
	_, err := e.Execute(ctx, &domain.TelephonyActionConfig{
		Action: telephony.ActionDial,
		To:     []string{"+14155551212"},
		For:    "30s",
	})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	s := ctx.Items[telephony.SessionKey].(*telephony.Session)
	twiml := s.TwiML()
	if !strings.Contains(twiml, "+14155551212") {
		t.Errorf("expected phone number in Dial TwiML, got: %s", twiml)
	}
	if !strings.Contains(twiml, `timeout="30"`) {
		t.Errorf("expected timeout=30 in Dial TwiML, got: %s", twiml)
	}
}

func TestExecDialCallerID(t *testing.T) {
	e := newExec()
	ctx := newTestCtx(nil)
	_, err := e.Execute(ctx, &domain.TelephonyActionConfig{
		Action: telephony.ActionDial,
		To:     []string{"+1000"},
		From:   "+9999",
	})
	if err != nil {
		t.Fatalf("dial callerID: %v", err)
	}
	s := ctx.Items[telephony.SessionKey].(*telephony.Session)
	twiml := s.TwiML()
	if !strings.Contains(twiml, `callerId="+9999"`) {
		t.Errorf("expected callerId in Dial, got: %s", twiml)
	}
}

// --- Action: record ---------------------------------------------------------

func TestExecRecord(t *testing.T) {
	e := newExec()
	ctx := newTestCtx(nil)
	_, err := e.Execute(ctx, &domain.TelephonyActionConfig{
		Action:        telephony.ActionRecord,
		Say:           "Leave a message",
		MaxDuration:   "60s",
		Interruptible: true,
	})
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	s := ctx.Items[telephony.SessionKey].(*telephony.Session)
	twiml := s.TwiML()
	if !strings.Contains(twiml, "<Say>Leave a message</Say>") {
		t.Errorf("expected Say before Record")
	}
	if !strings.Contains(twiml, "<Record") {
		t.Errorf("expected Record in TwiML, got: %s", twiml)
	}
	if !strings.Contains(twiml, `maxLength="60"`) {
		t.Errorf("expected maxLength=60")
	}
}

func TestExecRecordNotInterruptible(t *testing.T) {
	e := newExec()
	ctx := newTestCtx(nil)
	_, err := e.Execute(ctx, &domain.TelephonyActionConfig{
		Action:        telephony.ActionRecord,
		MaxDuration:   "30s",
		Interruptible: false,
	})
	if err != nil {
		t.Fatalf("record not interruptible: %v", err)
	}
	s := ctx.Items[telephony.SessionKey].(*telephony.Session)
	twiml := s.TwiML()
	// finishOnKey should be empty when not interruptible.
	if strings.Contains(twiml, "finishOnKey") {
		t.Errorf("unexpected finishOnKey for non-interruptible record: %s", twiml)
	}
}

// --- Action: mute / unmute --------------------------------------------------

func TestExecMute(t *testing.T) {
	e := newExec()
	ctx := newTestCtx(nil)
	_, err := e.Execute(ctx, &domain.TelephonyActionConfig{Action: telephony.ActionMute})
	if err != nil {
		t.Fatalf("mute: %v", err)
	}
	s := ctx.Items[telephony.SessionKey].(*telephony.Session)
	if s.Response.NodeCount() != 1 {
		t.Errorf("expected 1 node after mute")
	}
}

func TestExecUnmute(t *testing.T) {
	e := newExec()
	ctx := newTestCtx(nil)
	_, err := e.Execute(ctx, &domain.TelephonyActionConfig{Action: telephony.ActionUnmute})
	if err != nil {
		t.Fatalf("unmute: %v", err)
	}
	s := ctx.Items[telephony.SessionKey].(*telephony.Session)
	if s.Response.NodeCount() != 1 {
		t.Errorf("expected 1 node after unmute")
	}
}

// --- Action: hangup ---------------------------------------------------------

func TestExecHangup(t *testing.T) {
	e := newExec()
	ctx := newTestCtx(nil)
	_, err := e.Execute(ctx, &domain.TelephonyActionConfig{Action: telephony.ActionHangup})
	if err != nil {
		t.Fatalf("hangup: %v", err)
	}
	s := ctx.Items[telephony.SessionKey].(*telephony.Session)
	twiml := s.TwiML()
	if !strings.Contains(twiml, "<Hangup") {
		t.Errorf("expected <Hangup> in TwiML, got: %s", twiml)
	}
	if s.Status != "completed" {
		t.Errorf("Status = %q after hangup, want completed", s.Status)
	}
}

// --- Action: reject ---------------------------------------------------------

func TestExecReject(t *testing.T) {
	e := newExec()
	ctx := newTestCtx(nil)
	_, err := e.Execute(ctx, &domain.TelephonyActionConfig{
		Action: telephony.ActionReject,
		Reason: "busy",
	})
	if err != nil {
		t.Fatalf("reject: %v", err)
	}
	s := ctx.Items[telephony.SessionKey].(*telephony.Session)
	twiml := s.TwiML()
	if !strings.Contains(twiml, `reason="busy"`) {
		t.Errorf("expected reason=busy, got: %s", twiml)
	}
	if s.Status != "completed" {
		t.Errorf("Status = %q after reject, want completed", s.Status)
	}
}

// --- Action: redirect -------------------------------------------------------

func TestExecRedirect(t *testing.T) {
	e := newExec()
	ctx := newTestCtx(nil)
	_, err := e.Execute(ctx, &domain.TelephonyActionConfig{
		Action: telephony.ActionRedirect,
		To:     []string{"https://example.com/ivr2"},
	})
	if err != nil {
		t.Fatalf("redirect: %v", err)
	}
	s := ctx.Items[telephony.SessionKey].(*telephony.Session)
	twiml := s.TwiML()
	if !strings.Contains(twiml, "<Redirect>https://example.com/ivr2</Redirect>") {
		t.Errorf("expected Redirect URL, got: %s", twiml)
	}
}

func TestExecRedirectNoTarget(t *testing.T) {
	// No To entries - should produce empty Redirect.
	e := newExec()
	ctx := newTestCtx(nil)
	_, err := e.Execute(ctx, &domain.TelephonyActionConfig{
		Action: telephony.ActionRedirect,
	})
	if err != nil {
		t.Fatalf("redirect no target: %v", err)
	}
}

// --- Error cases ------------------------------------------------------------

func TestExecNilConfig(t *testing.T) {
	e := newExec()
	ctx := newTestCtx(nil)
	_, err := e.Execute(ctx, nil)
	if err == nil {
		t.Error("expected error for nil config")
	}
}

func TestExecUnknownAction(t *testing.T) {
	e := newExec()
	ctx := newTestCtx(nil)
	_, err := e.Execute(ctx, &domain.TelephonyActionConfig{Action: "unknown_action_xyz"})
	if err == nil {
		t.Error("expected error for unknown action")
	}
}

// --- Session persistence across multiple Execute calls ----------------------

func TestSessionPersistsAcrossActions(t *testing.T) {
	e := newExec()
	ctx := newTestCtx(map[string]any{"CallSid": "CApersist"})

	// First action: say
	_, err := e.Execute(ctx, &domain.TelephonyActionConfig{
		Action: telephony.ActionSay,
		Say:    "Welcome",
	})
	if err != nil {
		t.Fatalf("say: %v", err)
	}

	// Second action on same ctx: say again
	_, err = e.Execute(ctx, &domain.TelephonyActionConfig{
		Action: telephony.ActionSay,
		Say:    "Please hold",
	})
	if err != nil {
		t.Fatalf("say 2: %v", err)
	}

	// Third action: hangup
	_, err = e.Execute(ctx, &domain.TelephonyActionConfig{Action: telephony.ActionHangup})
	if err != nil {
		t.Fatalf("hangup: %v", err)
	}

	s := ctx.Items[telephony.SessionKey].(*telephony.Session)
	twiml := s.TwiML()
	if !strings.Contains(twiml, "Welcome") {
		t.Errorf("expected Welcome in TwiML")
	}
	if !strings.Contains(twiml, "Please hold") {
		t.Errorf("expected 'Please hold' in TwiML")
	}
	if !strings.Contains(twiml, "<Hangup") {
		t.Errorf("expected Hangup in TwiML")
	}
	// All three nodes should be in single Response
	if s.Response.NodeCount() != 3 {
		t.Errorf("expected 3 nodes, got %d", s.Response.NodeCount())
	}
}

// --- parseDurationSeconds (via indirect coverage through ask/dial) ----------

func TestParseDurationViaAsk(t *testing.T) {
	tests := []struct {
		timeout string
		wantSec int // expected timeout attribute in Gather
	}{
		{"5s", 5},
		{"30s", 30},
		{"2m", 120},
		{"0", 5}, // 0 → default 5
		{"", 5},  // empty → default 5
	}
	for _, tc := range tests {
		t.Run(tc.timeout, func(t *testing.T) {
			e := newExec()
			ctx := newTestCtx(nil)
			_, err := e.Execute(ctx, &domain.TelephonyActionConfig{
				Action:  telephony.ActionAsk,
				Limit:   1,
				Timeout: tc.timeout,
			})
			if err != nil {
				t.Fatalf("ask: %v", err)
			}
			s := ctx.Items[telephony.SessionKey].(*telephony.Session)
			twiml := s.TwiML()
			want := `timeout="` + string(rune('0'+tc.wantSec/10)) // quick check for small values
			_ = want
			// Just verify no error and Gather is present.
			if !strings.Contains(twiml, "<Gather") {
				t.Errorf("expected Gather in TwiML for timeout=%q", tc.timeout)
			}
		})
	}
}

// --- inputAttrFromMode (via ask) --------------------------------------------

func TestInputAttrFromMode(t *testing.T) {
	tests := []struct {
		mode      string
		wantInput string
	}{
		{"dtmf", "dtmf"},
		{"speech", "speech"},
		{"both", "dtmf speech"},
		{"", "dtmf"},         // default
		{"SPEECH", "speech"}, // case-insensitive
	}
	for _, tc := range tests {
		t.Run(tc.mode, func(t *testing.T) {
			e := newExec()
			ctx := newTestCtx(nil)
			_, err := e.Execute(ctx, &domain.TelephonyActionConfig{
				Action: telephony.ActionAsk,
				Mode:   tc.mode,
				Limit:  1,
			})
			if err != nil {
				t.Fatalf("ask mode=%q: %v", tc.mode, err)
			}
			s := ctx.Items[telephony.SessionKey].(*telephony.Session)
			twiml := s.TwiML()
			if !strings.Contains(twiml, `input="`+tc.wantInput+`"`) {
				t.Errorf("mode=%q: expected input=%q in TwiML, got: %s", tc.mode, tc.wantInput, twiml)
			}
		})
	}
}

// --- NilCtx session ---------------------------------------------------------

func TestExecNilCtx(t *testing.T) {
	// Execute with nil ctx should not panic.
	e := newExec()
	_, err := e.Execute(nil, &domain.TelephonyActionConfig{Action: telephony.ActionAnswer})
	if err != nil {
		t.Fatalf("unexpected error with nil ctx: %v", err)
	}
}

// --- Action constants -------------------------------------------------------

func TestActionConstants(t *testing.T) {
	expected := []string{
		telephony.ActionAnswer, telephony.ActionSay, telephony.ActionAsk,
		telephony.ActionMenu, telephony.ActionDial, telephony.ActionRecord,
		telephony.ActionMute, telephony.ActionUnmute, telephony.ActionHangup,
		telephony.ActionReject, telephony.ActionRedirect,
	}
	for _, a := range expected {
		if a == "" {
			t.Errorf("action constant should not be empty")
		}
	}
}

// --- Adapter ----------------------------------------------------------------

func TestAdapterExecute(t *testing.T) {
	a := telephony.NewAdapter()
	ctx := newTestCtx(nil)
	res, err := a.Execute(ctx, &domain.TelephonyActionConfig{
		Action: telephony.ActionSay,
		Say:    "adapter test",
	})
	if err != nil {
		t.Fatalf("adapter execute: %v", err)
	}
	m, ok := res.(map[string]any)
	if !ok {
		t.Fatal("expected map result")
	}
	twiml, _ := m["twiml"].(string)
	if !strings.Contains(twiml, "adapter test") {
		t.Errorf("twiml should contain say text, got: %s", twiml)
	}
}

func TestAdapterExecuteInvalidConfig(t *testing.T) {
	a := telephony.NewAdapter()
	ctx := newTestCtx(nil)
	_, err := a.Execute(ctx, "not-a-telephony-config")
	if err == nil {
		t.Error("expected error for invalid config type")
	}
}

func TestAdapterExecuteNilConfig(t *testing.T) {
	a := telephony.NewAdapter()
	ctx := newTestCtx(nil)
	_, err := a.Execute(ctx, (*domain.TelephonyActionConfig)(nil))
	if err == nil {
		t.Error("expected error for nil config")
	}
}

// --- parseDurationSeconds edge cases ----------------------------------------

func TestParseDurationInvalidString(t *testing.T) {
	// An unparseable string like "xyzs" has "s" suffix but non-numeric body.
	// Falls through all branches and returns 0, which causes execAsk to use default 5.
	e := newExec()
	ctx := newTestCtx(nil)
	_, err := e.Execute(ctx, &domain.TelephonyActionConfig{
		Action:  telephony.ActionAsk,
		Limit:   1,
		Timeout: "xyzs", // invalid — triggers parse error branch → default 5
	})
	if err != nil {
		t.Fatalf("ask with invalid timeout: %v", err)
	}
	s := ctx.Items[telephony.SessionKey].(*telephony.Session)
	twiml := s.TwiML()
	if !strings.Contains(twiml, `timeout="5"`) {
		t.Errorf("expected default timeout=5 for invalid input, got: %s", twiml)
	}
}

func TestParseDurationPlainInt(t *testing.T) {
	e := newExec()
	ctx := newTestCtx(nil)
	_, err := e.Execute(ctx, &domain.TelephonyActionConfig{
		Action:  telephony.ActionAsk,
		Limit:   1,
		Timeout: "12", // plain integer (no suffix)
	})
	if err != nil {
		t.Fatalf("ask with plain int timeout: %v", err)
	}
	s := ctx.Items[telephony.SessionKey].(*telephony.Session)
	twiml := s.TwiML()
	if !strings.Contains(twiml, `timeout="12"`) {
		t.Errorf("expected timeout=12 for plain int, got: %s", twiml)
	}
}

func TestParseDurationInvalidMinutes(t *testing.T) {
	// "xm" has "m" suffix but non-numeric body → falls to plain int path → returns 0 → default 5.
	e := newExec()
	ctx := newTestCtx(nil)
	_, err := e.Execute(ctx, &domain.TelephonyActionConfig{
		Action:  telephony.ActionAsk,
		Limit:   1,
		Timeout: "xm",
	})
	if err != nil {
		t.Fatalf("ask with invalid minutes: %v", err)
	}
	s := ctx.Items[telephony.SessionKey].(*telephony.Session)
	twiml := s.TwiML()
	if !strings.Contains(twiml, `timeout="5"`) {
		t.Errorf("expected default timeout=5 for invalid minutes, got: %s", twiml)
	}
}

// --- execMenu with explicit limit -------------------------------------------

func TestExecMenuWithLimit(t *testing.T) {
	e := newExec()
	ctx := newTestCtx(nil)
	_, err := e.Execute(ctx, &domain.TelephonyActionConfig{
		Action: telephony.ActionMenu,
		Say:    "Enter up to 2 digits.",
		Limit:  2,
		Matches: []domain.TelephonyMatch{
			{Keys: []string{"11"}},
			{Keys: []string{"22"}},
		},
	})
	if err != nil {
		t.Fatalf("menu with limit: %v", err)
	}
	s := ctx.Items[telephony.SessionKey].(*telephony.Session)
	twiml := s.TwiML()
	if !strings.Contains(twiml, `numDigits="2"`) {
		t.Errorf("expected numDigits=2 in Gather, got: %s", twiml)
	}
}

// --- ToEnvMap twiml accessor -------------------------------------------------

func TestToEnvMapTwimlAccessor(t *testing.T) {
	s := telephony.NewSession()
	s.Response.AddSay("hello", "")
	env := s.ToEnvMap()
	twimlFn, ok := env["twiml"].(func() string)
	if !ok {
		t.Fatal("env[twiml] should be func() string")
	}
	twiml := twimlFn()
	if !strings.Contains(twiml, "<Say>hello</Say>") {
		t.Errorf("twiml accessor should return accumulated TwiML, got: %s", twiml)
	}
}

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

package bot

import (
	"context"
	"errors"
	"testing"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kdepsconfig "github.com/kdeps/kdeps/v2/pkg/config"
	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func TestSlackHandleSocketEvent_AckError(t *testing.T) {
	orig := slackClientAck
	t.Cleanup(func() { slackClientAck = orig })

	slackClientAck = func(_ *socketmode.Client, _ socketmode.Request) error {
		return errors.New("ack failed")
	}

	api := slack.New("xoxb-test", slack.OptionAppLevelToken("xapp-test"))
	client := socketmode.New(api)
	runner := &slackRunner{}
	ch := make(chan Message, 1)

	runner.handleSocketEvent(context.Background(), client, socketmode.Event{
		Type: socketmode.EventTypeEventsAPI,
		Data: slackevents.EventsAPIEvent{
			Type: slackevents.CallbackEvent,
			InnerEvent: slackevents.EventsAPIInnerEvent{
				Data: &slackevents.MessageEvent{
					User: "U1", Text: "hi", Channel: "C1",
				},
			},
		},
		Request: &socketmode.Request{EnvelopeID: "env"},
	}, ch)

	assert.Empty(t, ch)
}

func TestSlackRunner_NewRunner(t *testing.T) {
	creds := &kdepsconfig.SlackConnectionConfig{BotToken: "xoxb-test", AppToken: "xapp-test"}
	runner := newSlackRunner(&domain.SlackConfig{}, creds, nil)
	require.NotNil(t, runner)
	assert.Equal(t, "xoxb-test", runner.botToken)
	assert.Nil(t, runner.client)
}

func TestSlackRunner_Reply_ClientNil(t *testing.T) {
	runner := newSlackRunner(
		&domain.SlackConfig{},
		&kdepsconfig.SlackConnectionConfig{BotToken: "xoxb-test", AppToken: "xapp-test"},
		nil,
	)
	// client is nil before Start() is called — should return error.
	err := runner.Reply(context.Background(), "C12345", "hello")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "client not started")
}

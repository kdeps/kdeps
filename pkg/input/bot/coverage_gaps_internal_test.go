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
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kdepsconfig "github.com/kdeps/kdeps/v2/pkg/config"
	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func TestCloseDiscordSession_DefaultClose(t *testing.T) {
	s, err := discordgo.New("Bot test-token")
	require.NoError(t, err)

	r := &discordRunner{logger: slog.Default()}
	r.closeDiscordSession(s)
}

func TestDiscordRunner_Start_SuccessAndClose(t *testing.T) {
	origNew := discordNewSession
	origOpen := discordSessionOpen
	origClose := discordSessionClose
	t.Cleanup(func() {
		discordNewSession = origNew
		discordSessionOpen = origOpen
		discordSessionClose = origClose
	})

	discordNewSession = func(_ string) (*discordgo.Session, error) {
		return &discordgo.Session{}, nil
	}
	discordSessionOpen = func(_ *discordgo.Session) error { return nil }
	discordSessionClose = func(_ *discordgo.Session) error {
		return errors.New("close failed")
	}

	r := &discordRunner{botToken: "token", logger: slog.Default()}
	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- r.Start(ctx, make(chan Message, 1))
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	err := <-errCh
	require.NoError(t, err)
}

func TestBuildBotRunners_NewError(t *testing.T) {
	orig := newBotRunner
	t.Cleanup(func() { newBotRunner = orig })

	newBotRunner = func(_ string, _ *domain.BotConfig, _ *kdepsconfig.BotConnectionConfig, _ *slog.Logger) (Runner, error) {
		return nil, errors.New("create failed")
	}

	botCfg := &domain.BotConfig{Discord: &domain.DiscordConfig{}}
	_, err := buildBotRunners(botCfg, nil, slog.Default())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create runner for discord")
}

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

func TestWhatsAppReply_MarshalError(t *testing.T) {
	orig := whatsAppJSONMarshal
	t.Cleanup(func() { whatsAppJSONMarshal = orig })
	whatsAppJSONMarshal = func(_ any) ([]byte, error) {
		return nil, errors.New("marshal failed")
	}

	r := &whatsAppRunner{accessToken: "tok", phoneNumberID: "123"}
	err := r.Reply(context.Background(), "1", "hi")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "whatsapp: marshal reply")
}

func TestWhatsAppReply_NewRequestError(t *testing.T) {
	orig := whatsAppNewRequest
	t.Cleanup(func() { whatsAppNewRequest = orig })
	whatsAppNewRequest = func(_ context.Context, _, _ string, _ io.Reader) (*http.Request, error) {
		return nil, errors.New("request failed")
	}

	r := &whatsAppRunner{accessToken: "tok", phoneNumberID: "123"}
	err := r.Reply(context.Background(), "1", "hi")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "whatsapp: build request")
}

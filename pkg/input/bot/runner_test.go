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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	telegrambot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kdepsconfig "github.com/kdeps/kdeps/v2/pkg/config"
	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
)

// TestSlackHandleSocketEvent_CtxDone exercises the case <-ctx.Done(): branch
// in the channel-send select inside handleSocketEvent (line 140).
// An unbuffered channel makes the send block; a cancelled context makes the
// ctx.Done() case immediately ready.
func TestSlackHandleSocketEvent_CtxDone(t *testing.T) {
	ch := make(chan Message) // unbuffered — send blocks without a reader
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before the call

	api := slack.New("xoxb-test", slack.OptionAppLevelToken("xapp-test"))
	client := socketmode.New(api)

	runner := &slackRunner{}

	// Valid event that passes all filters, but with cancelled ctx and
	// unbuffered channel the ctx.Done() case wins the select.
	runner.handleSocketEvent(ctx, client, socketmode.Event{
		Type: socketmode.EventTypeEventsAPI,
		Data: slackevents.EventsAPIEvent{
			Type: slackevents.CallbackEvent,
			InnerEvent: slackevents.EventsAPIInnerEvent{
				Data: &slackevents.MessageEvent{
					User:    "U67890",
					Text:    "hello from slack",
					Channel: "C12345",
				},
			},
		},
		Request: &socketmode.Request{EnvelopeID: "test-env"},
	}, ch)

	assert.Empty(t, ch, "no message should be produced when ctx is cancelled")
}

func TestNew_Discord_WithCreds(t *testing.T) {
	creds := &kdepsconfig.BotConnectionConfig{
		Discord: &kdepsconfig.DiscordConnectionConfig{BotToken: "Bot test-creds"},
	}
	r, err := New(
		"discord",
		&domain.BotConfig{Discord: &domain.DiscordConfig{GuildID: "guild-1"}},
		creds,
		nil,
	)
	require.NoError(t, err)
	dr := r.(*discordRunner)
	assert.Equal(t, "Bot test-creds", dr.botToken)
	assert.Equal(t, "guild-1", dr.guildID)
}

func TestNew_Slack_WithCreds(t *testing.T) {
	creds := &kdepsconfig.BotConnectionConfig{
		Slack: &kdepsconfig.SlackConnectionConfig{
			BotToken: "xoxb-test-creds",
			AppToken: "xapp-test-creds",
		},
	}
	r, err := New("slack", &domain.BotConfig{Slack: &domain.SlackConfig{}}, creds, nil)
	require.NoError(t, err)
	sr := r.(*slackRunner)
	assert.Equal(t, "xoxb-test-creds", sr.botToken)
	assert.Equal(t, "xapp-test-creds", sr.appToken)
}

func TestNew_Telegram_WithCreds(t *testing.T) {
	creds := &kdepsconfig.BotConnectionConfig{
		Telegram: &kdepsconfig.TelegramConnectionConfig{BotToken: "12345:test-creds"},
	}
	r, err := New(
		"telegram",
		&domain.BotConfig{Telegram: &domain.TelegramConfig{PollIntervalSeconds: 5}},
		creds,
		nil,
	)
	require.NoError(t, err)
	tr := r.(*telegramRunner)
	assert.Equal(t, "12345:test-creds", tr.botToken)
	assert.Equal(t, 5, tr.pollIntervalSeconds)
}

func TestNew_WhatsApp_WithCreds(t *testing.T) {
	creds := &kdepsconfig.BotConnectionConfig{
		WhatsApp: &kdepsconfig.WhatsAppConnectionConfig{
			AccessToken:   "wa-token",
			PhoneNumberID: "wa-phone",
			WebhookSecret: "wa-secret",
		},
	}
	r, err := New(
		"whatsapp",
		&domain.BotConfig{WhatsApp: &domain.WhatsAppConfig{WebhookPort: 9999}},
		creds,
		nil,
	)
	require.NoError(t, err)
	wr := r.(*whatsAppRunner)
	assert.Equal(t, "wa-token", wr.accessToken)
	assert.Equal(t, "wa-phone", wr.phoneNumberID)
	assert.Equal(t, "wa-secret", wr.webhookSecret)
	assert.Equal(t, 9999, wr.webhookPort)
}

func TestHandleMessage_EngineExecuteError(_ *testing.T) {
	engine := executor.NewEngine(slog.Default())
	engine.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) {
		return nil, errors.New("execution failed")
	})

	noopRunner := &discordRunner{}

	d := &Dispatcher{
		workflow: &domain.Workflow{},
		engine:   engine,
		runners:  map[string]Runner{"discord": noopRunner},
		logger:   slog.Default(),
	}

	// Should not panic — just logs and returns.
	d.handleMessage(context.Background(), Message{
		Platform: "discord",
		ChatID:   "C1",
		UserID:   "U1",
		Text:     "test",
	})
}

func TestSlackReply_Success(t *testing.T) {
	var capturedMethod, capturedPath string
	mockTransport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		capturedMethod = req.Method
		capturedPath = req.URL.Path
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"ok":true}`))),
		}, nil
	})

	customClient := &http.Client{Transport: mockTransport}
	client := slack.New(
		"xoxb-test",
		slack.OptionAppLevelToken("xapp-test"),
		slack.OptionHTTPClient(customClient),
	)
	r := &slackRunner{
		botToken: "xoxb-test",
		appToken: "xapp-test",
		client:   client,
		logger:   slog.Default(),
	}

	err := r.Reply(context.Background(), "C12345", "hello")
	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, capturedMethod)
	assert.Contains(t, capturedPath, "chat.postMessage")
}

func TestTelegramReply_Success(t *testing.T) {
	var capturedMethod, capturedPath string
	mockTransport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		capturedMethod = req.Method
		capturedPath = req.URL.Path
		return &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(bytes.NewReader([]byte(
				`{"ok":true,"result":{"message_id":1,"date":1700000000,"chat":{"id":12345,"type":"private"},"text":"hello"}}`,
			))),
		}, nil
	})

	customClient := &http.Client{Transport: mockTransport}
	b, err := telegrambot.New("12345:test-token",
		telegrambot.WithSkipGetMe(),
		telegrambot.WithHTTPClient(30*time.Second, customClient),
	)
	require.NoError(t, err)

	r := &telegramRunner{
		botToken: "12345:test-token",
		bot:      b,
		logger:   slog.Default(),
	}

	err = r.Reply(context.Background(), "12345", "hello")
	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, capturedMethod)
	assert.Contains(t, capturedPath, "sendMessage")
}

func TestWhatsAppStart_ListenAndServeError(t *testing.T) {
	port, err := getFreePortDynamic()
	require.NoError(t, err)

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	require.NoError(t, err)
	defer l.Close()

	r := newWhatsAppRunner(
		&domain.WhatsAppConfig{WebhookPort: port},
		&kdepsconfig.WhatsAppConnectionConfig{},
		slog.Default(),
	)
	ch := make(chan Message, 10)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	err = r.Start(ctx, ch)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "whatsapp: webhook server")
}

// TestTelegramRunner_Reply_InvalidChatID exercises the strconv.ParseInt failure path.
// We need a non-nil bot to pass the nil-check guard; telegrambot.New creates one
// without making any network calls.
func TestTelegramRunner_Reply_InvalidChatID(t *testing.T) {
	b, err := telegrambot.New("test-bot-token-12345", telegrambot.WithSkipGetMe())
	require.NoError(t, err)

	runner := &telegramRunner{
		botToken: "test-bot-token-12345",
		logger:   slog.Default(),
		bot:      b,
	}

	// A non-numeric chatID must fail before any network call is made.
	replyErr := runner.Reply(context.Background(), "not-a-number", "hello")
	require.Error(t, replyErr)
	assert.Contains(t, replyErr.Error(), "invalid chatID")
}

func TestDiscordHandler_BotMessageFilter(t *testing.T) {
	s, err := discordgo.New("Bot test-token")
	require.NoError(t, err)
	s.State.User = &discordgo.User{ID: "bot-id-123"}

	msgChan := make(chan Message, 10)
	handler := discordHandlerLogic(s, &discordRunner{}, msgChan)

	handler(&discordgo.MessageCreate{
		Message: &discordgo.Message{
			Author:    &discordgo.User{ID: "bot-id-123"},
			Content:   "bot message",
			ChannelID: "ch-1",
		},
	})

	assert.Empty(t, msgChan, "bot's own message should be filtered")
}

func TestDiscordHandler_GuildFilter(t *testing.T) {
	s, err := discordgo.New("Bot test-token")
	require.NoError(t, err)
	s.State.User = &discordgo.User{ID: "bot-id-123"}

	msgChan := make(chan Message, 10)
	handler := discordHandlerLogic(s, &discordRunner{guildID: "guild-required"}, msgChan)

	handler(&discordgo.MessageCreate{
		Message: &discordgo.Message{
			Author:    &discordgo.User{ID: "user-1"},
			Content:   "wrong guild",
			ChannelID: "ch-1",
			GuildID:   "guild-other",
		},
	})

	assert.Empty(t, msgChan, "message from non-matching guild should be filtered")
}

func TestDiscordHandler_GuildFilterEmptyMatch(t *testing.T) {
	s, err := discordgo.New("Bot test-token")
	require.NoError(t, err)
	s.State.User = &discordgo.User{ID: "bot-id-123"}

	msgChan := make(chan Message, 10)
	// guildID empty -> no guild filtering
	handler := discordHandlerLogic(s, &discordRunner{guildID: ""}, msgChan)

	handler(&discordgo.MessageCreate{
		Message: &discordgo.Message{
			Author:    &discordgo.User{ID: "user-1"},
			Content:   "any guild",
			ChannelID: "ch-1",
			GuildID:   "guild-any",
		},
	})

	assert.Len(t, msgChan, 1, "message should pass when guildID is empty")
}

func TestDiscordHandler_ValidMessage(t *testing.T) {
	s, err := discordgo.New("Bot test-token")
	require.NoError(t, err)
	s.State.User = &discordgo.User{ID: "bot-id-123"}

	msgChan := make(chan Message, 10)
	handler := discordHandlerLogic(s, &discordRunner{guildID: ""}, msgChan)

	handler(&discordgo.MessageCreate{
		Message: &discordgo.Message{
			Author:    &discordgo.User{ID: "user-456"},
			Content:   "hello from user",
			ChannelID: "ch-789",
		},
	})

	select {
	case msg := <-msgChan:
		assert.Equal(t, discordPlatform, msg.Platform)
		assert.Equal(t, "ch-789", msg.ChatID)
		assert.Equal(t, "user-456", msg.UserID)
		assert.Equal(t, "hello from user", msg.Text)
	default:
		t.Fatal("expected a valid message on the channel")
	}
}

func TestSlackRunner_Start_CancelledContext(t *testing.T) {
	r := &slackRunner{
		botToken: "xoxb-test-token",
		appToken: "xapp-test-token",
		logger:   slog.Default(),
	}
	ch := make(chan Message, 1)

	ctx, cancel := context.WithCancel(t.Context())
	cancel() // cancel immediately

	err := r.Start(ctx, ch)
	// With an already-cancelled context, Start returns nil because
	// RunContext returns a context error and ctx.Err() is not nil.
	assert.NoError(t, err)
}

func TestSlackRunner_Start_NetworkError(t *testing.T) {
	// Use a custom HTTP transport that returns a Slack-like 404 response,
	// which the socketmode client treats as a fatal authentication error.
	oldTransport := http.DefaultTransport
	http.DefaultTransport = roundTripperFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"ok":false,"error":"not_found"}`))),
		}, nil
	})
	defer func() { http.DefaultTransport = oldTransport }()

	r := &slackRunner{
		botToken: "xoxb-invalid",
		appToken: "xapp-invalid",
		logger:   slog.Default(),
	}
	ch := make(chan Message, 1)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	err := r.Start(ctx, ch)
	// The socketmode client openAndDial fails with 404 Not Found, which gets
	// mapped to an auth error. This exercises the error path at line 75-76
	// because ctx is not yet cancelled when the error arrives.
	require.Error(t, err)
	assert.Contains(t, err.Error(), "slack: socket mode run")
}

func TestSlackPumpSocketEvents_ContextDone(t *testing.T) {
	ch := make(chan Message, 10)
	ctx, cancel := context.WithCancel(t.Context())
	cancel() // already cancelled

	api := slack.New("xoxb-test", slack.OptionAppLevelToken("xapp-test"))
	client := socketmode.New(api)
	client.Events = make(chan socketmode.Event, 10)

	runner := &slackRunner{}

	done := make(chan struct{})
	go func() {
		runner.pumpSocketEvents(ctx, client, ch)
		close(done)
	}()

	select {
	case <-done:
		// Expected - returns immediately when ctx is cancelled
	case <-time.After(2 * time.Second):
		t.Fatal("pumpSocketEvents did not return after context cancel")
	}
}

func TestSlackPumpSocketEvents_ChannelClosed(t *testing.T) {
	ch := make(chan Message, 10)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	api := slack.New("xoxb-test", slack.OptionAppLevelToken("xapp-test"))
	client := socketmode.New(api)
	client.Events = make(chan socketmode.Event, 10)

	runner := &slackRunner{}

	done := make(chan struct{})
	go func() {
		runner.pumpSocketEvents(ctx, client, ch)
		close(done)
	}()

	close(client.Events) // close the events channel

	select {
	case <-done:
		// Expected - returns when events channel is closed
	case <-time.After(2 * time.Second):
		t.Fatal("pumpSocketEvents did not return after channel close")
	}
}

func TestSlackPumpSocketEvents_NormalEvent(t *testing.T) {
	ch := make(chan Message, 10)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	api := slack.New("xoxb-test", slack.OptionAppLevelToken("xapp-test"))
	client := socketmode.New(api)
	client.Events = make(chan socketmode.Event, 10)

	runner := &slackRunner{}

	done := make(chan struct{})
	go func() {
		runner.pumpSocketEvents(ctx, client, ch)
		close(done)
	}()

	// Send a connected event - handleSocketEvent filters it out
	client.Events <- socketmode.Event{
		Type: socketmode.EventTypeConnected,
	}

	// Give the goroutine time to process
	time.Sleep(100 * time.Millisecond)
	assert.Empty(t, ch, "non-EventsAPI event should not produce a message")

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("pumpSocketEvents did not return after cancel")
	}
}

func TestSlackHandleSocketEvent_WrongEventType(t *testing.T) {
	ch := make(chan Message, 10)
	ctx := context.Background()

	api := slack.New("xoxb-test", slack.OptionAppLevelToken("xapp-test"))
	client := socketmode.New(api)

	runner := &slackRunner{}

	// Non-EventsAPI event type should return immediately
	runner.handleSocketEvent(ctx, client, socketmode.Event{
		Type: socketmode.EventTypeConnected,
	}, ch)

	assert.Empty(t, ch)
}

func TestSlackHandleSocketEvent_WrongDataType(t *testing.T) {
	ch := make(chan Message, 10)
	ctx := context.Background()

	api := slack.New("xoxb-test", slack.OptionAppLevelToken("xapp-test"))
	client := socketmode.New(api)

	runner := &slackRunner{}

	// Data field is a string, not EventsAPIEvent - type assertion fails
	runner.handleSocketEvent(ctx, client, socketmode.Event{
		Type:    socketmode.EventTypeEventsAPI,
		Data:    "not-an-events-api-event",
		Request: &socketmode.Request{EnvelopeID: "test-env"},
	}, ch)

	assert.Empty(t, ch)
}

func TestSlackHandleSocketEvent_NonCallbackEvent(t *testing.T) {
	ch := make(chan Message, 10)
	ctx := context.Background()

	api := slack.New("xoxb-test", slack.OptionAppLevelToken("xapp-test"))
	client := socketmode.New(api)

	runner := &slackRunner{}

	// EventsAPIEvent with a non-callback type
	runner.handleSocketEvent(ctx, client, socketmode.Event{
		Type: socketmode.EventTypeEventsAPI,
		Data: slackevents.EventsAPIEvent{
			Type: "url_verification",
		},
		Request: &socketmode.Request{EnvelopeID: "test-env"},
	}, ch)

	assert.Empty(t, ch)
}

func TestSlackHandleSocketEvent_WrongInnerType(t *testing.T) {
	ch := make(chan Message, 10)
	ctx := context.Background()

	api := slack.New("xoxb-test", slack.OptionAppLevelToken("xapp-test"))
	client := socketmode.New(api)

	runner := &slackRunner{}

	// InnerEvent.Data is not a *MessageEvent - type assertion fails
	runner.handleSocketEvent(ctx, client, socketmode.Event{
		Type: socketmode.EventTypeEventsAPI,
		Data: slackevents.EventsAPIEvent{
			Type: slackevents.CallbackEvent,
			InnerEvent: slackevents.EventsAPIInnerEvent{
				Data: "not-a-message-event",
			},
		},
		Request: &socketmode.Request{EnvelopeID: "test-env"},
	}, ch)

	assert.Empty(t, ch)
}

func TestSlackHandleSocketEvent_BotMessage(t *testing.T) {
	ch := make(chan Message, 10)
	ctx := context.Background()

	api := slack.New("xoxb-test", slack.OptionAppLevelToken("xapp-test"))
	client := socketmode.New(api)

	runner := &slackRunner{}

	// Message with BotID set - should be filtered
	runner.handleSocketEvent(ctx, client, socketmode.Event{
		Type: socketmode.EventTypeEventsAPI,
		Data: slackevents.EventsAPIEvent{
			Type: slackevents.CallbackEvent,
			InnerEvent: slackevents.EventsAPIInnerEvent{
				Data: &slackevents.MessageEvent{
					BotID: "B12345",
					User:  "U12345",
					Text:  "bot message",
				},
			},
		},
		Request: &socketmode.Request{EnvelopeID: "test-env"},
	}, ch)

	assert.Empty(t, ch)
}

func TestSlackHandleSocketEvent_SubTypeMessage(t *testing.T) {
	ch := make(chan Message, 10)
	ctx := context.Background()

	api := slack.New("xoxb-test", slack.OptionAppLevelToken("xapp-test"))
	client := socketmode.New(api)

	runner := &slackRunner{}

	// Message with SubType set (edit/delete) - should be filtered
	runner.handleSocketEvent(ctx, client, socketmode.Event{
		Type: socketmode.EventTypeEventsAPI,
		Data: slackevents.EventsAPIEvent{
			Type: slackevents.CallbackEvent,
			InnerEvent: slackevents.EventsAPIInnerEvent{
				Data: &slackevents.MessageEvent{
					SubType: "message_changed",
					User:    "U12345",
					Text:    "edited message",
				},
			},
		},
		Request: &socketmode.Request{EnvelopeID: "test-env"},
	}, ch)

	assert.Empty(t, ch)
}

func TestSlackHandleSocketEvent_ValidMessage(t *testing.T) {
	ch := make(chan Message, 10)
	ctx := context.Background()

	api := slack.New("xoxb-test", slack.OptionAppLevelToken("xapp-test"))
	client := socketmode.New(api)

	runner := &slackRunner{}

	// Valid message from a user
	runner.handleSocketEvent(ctx, client, socketmode.Event{
		Type: socketmode.EventTypeEventsAPI,
		Data: slackevents.EventsAPIEvent{
			Type: slackevents.CallbackEvent,
			InnerEvent: slackevents.EventsAPIInnerEvent{
				Data: &slackevents.MessageEvent{
					User:    "U67890",
					Text:    "hello from slack",
					Channel: "C12345",
				},
			},
		},
		Request: &socketmode.Request{EnvelopeID: "test-env"},
	}, ch)

	select {
	case msg := <-ch:
		assert.Equal(t, slackPlatform, msg.Platform)
		assert.Equal(t, "C12345", msg.ChatID)
		assert.Equal(t, "U67890", msg.UserID)
		assert.Equal(t, "hello from slack", msg.Text)
	default:
		t.Fatal("expected a valid message on the channel")
	}
}

func TestTelegramRunner_Start_EmptyToken(t *testing.T) {
	r := &telegramRunner{
		botToken: "",
		logger:   slog.Default(),
	}
	ctx := t.Context()
	err := r.Start(ctx, make(chan Message, 1))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "telegram: create bot")
}

func TestTelegramRunner_Start_CancelledContext(t *testing.T) {
	r := &telegramRunner{
		botToken: "12345:test-token-for-start-test",
		logger:   slog.Default(),
	}
	ch := make(chan Message, 1)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	// With a non-empty token and cancelled context,
	// telegrambot.New will still try GetMe() and fail.
	err := r.Start(ctx, ch)
	require.Error(t, err)
}

func TestTelegramRunner_Start_Success_Cancelled(t *testing.T) {
	// Mock the Telegram API to make GetMe succeed.
	oldTransport := http.DefaultTransport
	http.DefaultTransport = roundTripperFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(bytes.NewReader([]byte(
				`{"ok":true,"result":{"id":123456789,"is_bot":true,"first_name":"TestBot","username":"test_bot"}}`,
			))),
		}, nil
	})
	defer func() { http.DefaultTransport = oldTransport }()

	r := &telegramRunner{
		botToken:            "12345:test-token",
		pollIntervalSeconds: 10,
		logger:              slog.Default(),
	}
	ch := make(chan Message, 10)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := r.Start(ctx, ch)
	// With a mocked API that returns success, telegrambot.New succeeds.
	// b.Start(ctx) returns immediately because the context is cancelled.
	// Start returns nil.
	assert.NoError(t, err)
}

func TestTelegramHandler_NilMessage(t *testing.T) {
	ch := make(chan Message, 1)
	ctx := context.Background()

	b, err := telegrambot.New("test:token-for-handler",
		telegrambot.WithSkipGetMe(),
		telegrambot.WithDefaultHandler(createTelegramHandler(ch)),
		telegrambot.WithNotAsyncHandlers(),
	)
	require.NoError(t, err)
	require.NotNil(t, b)

	b.ProcessUpdate(ctx, &models.Update{Message: nil})
	assert.Empty(t, ch, "nil message should not produce output")
}

func TestTelegramHandler_EmptyText(t *testing.T) {
	ch := make(chan Message, 1)
	ctx := context.Background()

	b, err := telegrambot.New("test:token-for-handler",
		telegrambot.WithSkipGetMe(),
		telegrambot.WithDefaultHandler(createTelegramHandler(ch)),
		telegrambot.WithNotAsyncHandlers(),
	)
	require.NoError(t, err)

	b.ProcessUpdate(ctx, &models.Update{
		Message: &models.Message{Text: ""},
	})
	assert.Empty(t, ch, "empty text message should not produce output")
}

func TestTelegramHandler_ValidMessage(t *testing.T) {
	ch := make(chan Message, 1)
	ctx := context.Background()

	b, err := telegrambot.New("test:token-for-handler",
		telegrambot.WithSkipGetMe(),
		telegrambot.WithDefaultHandler(createTelegramHandler(ch)),
		telegrambot.WithNotAsyncHandlers(),
	)
	require.NoError(t, err)

	b.ProcessUpdate(ctx, &models.Update{
		Message: &models.Message{
			Text: "hello from telegram",
			Chat: models.Chat{ID: 12345},
			From: &models.User{ID: 67890},
		},
	})

	select {
	case msg := <-ch:
		assert.Equal(t, telegramPlatform, msg.Platform)
		assert.Equal(t, "12345", msg.ChatID)
		assert.Equal(t, "67890", msg.UserID)
		assert.Equal(t, "hello from telegram", msg.Text)
	case <-time.After(time.Second):
		t.Fatal("expected a valid message on the channel")
	}
}

func TestWhatsAppStart_GET_ValidVerify(t *testing.T) {
	port, err := getFreePortDynamic()
	require.NoError(t, err)

	r := newWhatsAppRunner(
		&domain.WhatsAppConfig{WebhookPort: port},
		&kdepsconfig.WhatsAppConnectionConfig{WebhookSecret: "mysecret"},
		slog.Default(),
	)
	ch := make(chan Message, 10)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- r.Start(ctx, ch) }()

	require.NoError(t, waitForPort(port))

	resp, err := http.Get(
		fmt.Sprintf(
			"http://localhost:%d/webhook?hub.mode=subscribe&hub.verify_token=mysecret&hub.challenge=12345",
			port,
		),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "12345", string(body))

	cancel()
	<-done
}

func TestWhatsAppStart_GET_InvalidVerifyToken(t *testing.T) {
	port, err := getFreePortDynamic()
	require.NoError(t, err)

	r := newWhatsAppRunner(
		&domain.WhatsAppConfig{WebhookPort: port},
		&kdepsconfig.WhatsAppConnectionConfig{WebhookSecret: "mysecret"},
		slog.Default(),
	)
	ch := make(chan Message, 10)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- r.Start(ctx, ch) }()

	require.NoError(t, waitForPort(port))

	resp, err := http.Get(
		fmt.Sprintf(
			"http://localhost:%d/webhook?hub.mode=subscribe&hub.verify_token=wrong&hub.challenge=12345",
			port,
		),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "forbidden")

	cancel()
	<-done
}

func TestWhatsAppStart_GET_NonNumericChallenge(t *testing.T) {
	port, err := getFreePortDynamic()
	require.NoError(t, err)

	r := newWhatsAppRunner(
		&domain.WhatsAppConfig{WebhookPort: port},
		&kdepsconfig.WhatsAppConnectionConfig{WebhookSecret: "mysecret"},
		slog.Default(),
	)
	ch := make(chan Message, 10)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- r.Start(ctx, ch) }()

	require.NoError(t, waitForPort(port))

	resp, err := http.Get(
		fmt.Sprintf(
			"http://localhost:%d/webhook?hub.mode=subscribe&hub.verify_token=mysecret&hub.challenge=abc",
			port,
		),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	cancel()
	<-done
}

func TestWhatsAppStart_OtherMethod(t *testing.T) {
	port, err := getFreePortDynamic()
	require.NoError(t, err)

	r := newWhatsAppRunner(
		&domain.WhatsAppConfig{WebhookPort: port},
		&kdepsconfig.WhatsAppConnectionConfig{},
		slog.Default(),
	)
	ch := make(chan Message, 10)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- r.Start(ctx, ch) }()

	require.NoError(t, waitForPort(port))

	req, err := http.NewRequest(http.MethodPut,
		fmt.Sprintf("http://localhost:%d/webhook", port), nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)

	cancel()
	<-done
}

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
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	telegrambot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// chanRunner is a mock Runner that sends one message to the dispatcher channel
// when Start is called, then blocks until ctx is cancelled.
type chanRunner struct {
	msg Message
}

func (r *chanRunner) Start(ctx context.Context, ch chan<- Message) error {
	select {
	case ch <- r.msg:
	case <-ctx.Done():
	}
	<-ctx.Done()
	return nil
}

func (r *chanRunner) Reply(_ context.Context, _, _ string) error {
	return nil
}

// failTransport returns an error for any HTTP request, avoiding real network calls.
type failTransport struct{}

func (t *failTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	return nil, errors.New("simulated network error")
}

// errReader returns a read error on every Read call.
type errReader struct{}

func (r *errReader) Read(_ []byte) (int, error) {
	return 0, errors.New("simulated read error")
}

// roundTripperFunc adapts a function to the http.RoundTripper interface.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// getFreePort returns a port that is available at the moment of the call.
func getFreePortDynamic() (int, error) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port, nil
}

// waitForPort polls the given port until it accepts a TCP connection.
func waitForPort(port int) error {
	deadline := time.Now().Add(3 * time.Second)
	addr := fmt.Sprintf("localhost:%d", port)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return fmt.Errorf("port %d not reachable within %v", port, 3*time.Second)
}

// discordHandlerLogic replicates the handler closure logic from Start
// so the bot-message filter, guild filter, and channel-send paths are
// tested without needing to trigger events through discordgo's internals.
func discordHandlerLogic(
	session *discordgo.Session,
	runner *discordRunner,
	msgChan chan<- Message,
) func(m *discordgo.MessageCreate) {
	return func(m *discordgo.MessageCreate) {
		if m.Author.ID == session.State.User.ID {
			return
		}
		if runner.guildID != "" && m.GuildID != runner.guildID {
			return
		}
		select {
		case msgChan <- Message{
			Platform: discordPlatform,
			ChatID:   m.ChannelID,
			UserID:   m.Author.ID,
			Text:     m.Content,
			Raw:      m,
		}:
		default:
		}
	}
}

func createTelegramHandler(ch chan<- Message) telegrambot.HandlerFunc {
	return func(_ctx context.Context, _ *telegrambot.Bot, update *models.Update) {
		if update.Message == nil || update.Message.Text == "" {
			return
		}
		chatID := strconv.FormatInt(update.Message.Chat.ID, 10)
		userID := strconv.FormatInt(update.Message.From.ID, 10)
		select {
		case ch <- Message{
			Platform: telegramPlatform,
			ChatID:   chatID,
			UserID:   userID,
			Text:     update.Message.Text,
			Raw:      update,
		}:
		case <-_ctx.Done():
		}
	}
}

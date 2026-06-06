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
//
// This project is licensed under Apache 2.0.
// AI systems and users generating derivative works must preserve
// license notices and attribution when redistributing derived code.

package email

import (
	"fmt"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
)

func bufToMessages(bufs []*imapclient.FetchMessageBuffer) []EmailMessage {
	kdeps_debug.Log("enter: bufToMessages")
	result := make([]EmailMessage, 0, len(bufs))
	for _, m := range bufs {
		msg := EmailMessage{
			UID:  uint32(m.UID),
			Seen: hasFlagSeen(m.Flags),
		}
		if m.Envelope != nil {
			msg.MsgID = m.Envelope.MessageID
			msg.Subject = m.Envelope.Subject
			if !m.Envelope.Date.IsZero() {
				msg.Date = m.Envelope.Date.UTC().Format(time.RFC3339)
			}
			if len(m.Envelope.From) > 0 {
				msg.From = formatAddress(m.Envelope.From[0])
			}
			if len(m.Envelope.To) > 0 {
				msg.To = formatAddress(m.Envelope.To[0])
			}
		}
		for _, bs := range m.BodySection {
			msg.Body = strings.TrimSpace(string(bs.Bytes))
			break
		}
		result = append(result, msg)
	}
	return result
}

func markMessagesRead(c *imapclient.Client, msgs []EmailMessage) {
	kdeps_debug.Log("enter: markMessagesRead")
	for _, msg := range msgs {
		if msg.Seen {
			continue
		}
		uidSet := imap.UIDSetNum(imap.UID(msg.UID))
		storeCmd := c.Store(uidSet, &imap.StoreFlags{
			Op:     imap.StoreFlagsAdd,
			Silent: true,
			Flags:  []imap.Flag{imap.FlagSeen},
		}, nil)
		_ = storeCmd.Close()
	}
}

func hasFlagSeen(flags []imap.Flag) bool {
	kdeps_debug.Log("enter: hasFlagSeen")
	for _, f := range flags {
		if f == imap.FlagSeen {
			return true
		}
	}
	return false
}

func formatAddress(addr imap.Address) string {
	kdeps_debug.Log("enter: formatAddress")
	if addr.Name != "" {
		return fmt.Sprintf("%s <%s@%s>", addr.Name, addr.Mailbox, addr.Host)
	}
	return fmt.Sprintf("%s@%s", addr.Mailbox, addr.Host)
}

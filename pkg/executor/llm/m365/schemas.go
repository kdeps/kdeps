package m365

import "encoding/json"

// SignalR frame type codes used on the wire.
const (
	frameInvocation       = 1 // server->client streamed update ("update" target)
	frameStreamItem       = 2 // final item carrying authoritative metadata
	frameCompletion       = 3 // invocation completion
	frameStreamInvocation = 4 // client->server streaming invocation (chat)
	framePing             = 6 // keep-alive
	frameClose            = 7 // server close
)

// This file mirrors the JSON shapes exchanged with the chat service. Fields are
// optional on the wire, so pointers/omitempty are used where absence is
// meaningful and zero-values would be ambiguous.

// JWTClaims is the subset of access-token claims the client needs. oid and tid
// form the "{oid}@{tid}" segment of the chat WebSocket URL.
type JWTClaims struct {
	Aud  string `json:"aud"`
	Iss  string `json:"iss"`
	OID  string `json:"oid"`
	TID  string `json:"tid"`
	Exp  int64  `json:"exp"`
	Name string `json:"name,omitempty"`
	UPN  string `json:"upn,omitempty"`
}

// signalRHandshakeResponse is the first frame the server sends after the
// handshake. A non-empty Error means the handshake failed.
type signalRHandshakeResponse struct {
	Error string `json:"error,omitempty"`
}

// throttlingInfo carries per-conversation message quota counters.
type throttlingInfo struct {
	MaxNumUserMessagesInConversation            int `json:"maxNumUserMessagesInConversation"`
	NumUserMessagesInConversation               int `json:"numUserMessagesInConversation"`
	NumLongDocSummaryUserMessagesInConversation int `json:"numLongDocSummaryUserMessagesInConversation"`
}

// classifierScore is a per-component safety/quality score attached to a bot
// message. Values above roughly 2e-3 correlate with the turn being suppressed.
type classifierScore struct {
	Component string  `json:"component"`
	Score     float64 `json:"score"`
}

// botMessage is one message in an update or final-item frame.
type botMessage struct {
	Text              string            `json:"text,omitempty"`
	Author            string            `json:"author,omitempty"`
	MessageID         string            `json:"messageId,omitempty"`
	MessageType       string            `json:"messageType,omitempty"`
	ContentOrigin     string            `json:"contentOrigin,omitempty"`
	Scores            []classifierScore `json:"scores,omitempty"`
	TurnCount         *int              `json:"turnCount,omitempty"`
	TurnState         string            `json:"turnState,omitempty"`
	AdaptiveCards     []any             `json:"adaptiveCards,omitempty"`
	SourceAttribution []any             `json:"sourceAttributions,omitempty"`
}

// deltaUpdate is a token-level streaming chunk written at the current cursor.
type deltaUpdate struct {
	WriteAtCursor string `json:"writeAtCursor"`
}

// messageUpdate is a full-text snapshot of the conversation's messages.
type messageUpdate struct {
	Messages []botMessage `json:"messages"`
}

// throttlingUpdate wraps a throttle counter refresh.
type throttlingUpdate struct {
	Throttling throttlingInfo `json:"throttling"`
}

// streamItem is the "type:2" final-state frame carrying authoritative metadata.
type streamItem struct {
	Item *struct {
		Messages   []botMessage    `json:"messages"`
		Throttling *throttlingInfo `json:"throttling"`
		TurnState  string          `json:"turnState"`
	} `json:"item"`
}

// signalRFrame is the common envelope used to route incoming frames by type.
type signalRFrame struct {
	Type      int               `json:"type"`
	Target    string            `json:"target,omitempty"`
	Arguments []json.RawMessage `json:"arguments,omitempty"`
	Error     string            `json:"error,omitempty"`
}

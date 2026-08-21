package m365

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
)

// rs is the SignalR record separator terminating every frame on the wire.
const rs = "\x1E"

// chatWSBase is the chat WebSocket base URL. It is a var (not a const) only so
// tests can point the transport at a local server.
//
//nolint:gochecknoglobals // effectively constant; overridden only in tests
var chatWSBase = "wss://substrate.office.com/m365Copilot/Chathub"

// deltaBufferSize is the streamed-delta channel capacity; sized so early tokens
// are never dropped before the consumer attaches.
const deltaBufferSize = 64

// stopFrame mirrors the UI's cancel button: a type:1 invocation targeting
// "stop" on the same socket. The server acks with a type:3 completion.
const stopFrame = `{"arguments":[{}],"invocationId":"1","target":"stop","type":1}` + rs

// codeInterpreterOptionsSets unlock the server-side Python sandbox. They apply
// only on the agent-less path; with an agent attached the model routes tool
// calls through the agent instead.
//
//nolint:gochecknoglobals // static option list, read-only
var codeInterpreterOptionsSets = []string{
	"cwc_code_interpreter",
	"cwc_code_interpreter_amsfix",
	"cwc_code_interpreter_citation_fix",
	"code_interpreter_interactive_charts",
	"code_interpreter_matplotlib_patching",
}

// variants is the feature-flag list the chat URL carries as one query param.
const variants = "feature.EnableMcpServerWidgets,feature.EnableLuForChatCIQ," +
	"feature.enableChatCIQPlugin,feature.EnableRequestPlugins,feature.EnableSensitivityLabels," +
	"feature.EnableUnsupportedUrlDetector,feature.IsCustomEngineCopilotEnabled,feature.bizchatfluxv3," +
	"feature.enablechatpages,feature.enableCodeCanvas,feature.turnOnWorkTabRecommendation," +
	"feature.turnOffWorkTabUpsellFromClient,feature.turnOnDARecommendation," +
	"feature.IsStreamingModeInChatRequestEnabled,feature.IncludeSourceAttributionsConcise," +
	"feature.SkipPublishEmptyMessage,feature.EnableDeduplicatingSourceAttributions," +
	"feature.Enable3PActionProgressMessages,feature.enableClientWebRtc," +
	"feature.EnableMeetingRecapOfSeriesMeetingWithCiq,feature.EnableReferencesListCompleteSignal," +
	"feature.StorageMessageSplitDisabled,feature.EnableCuaTakeControlApi,feature.cwcallowedos," +
	"feature.disabledisallowedmsgs,feature.enableCitationsForSynthesisData," +
	"feature.enableGenerateGraphicArtOptionsSet,feature.cdximagen," +
	"feature.EnableUpdatedUXForConfirmationDialog," +
	"feature.EnableClientFileURLSupportForOfficeWebPaidCopilot," +
	"feature.EnableDesignEditorImageGrounding,feature.EnableDesignerEditor," +
	"feature.OfficeWebToHelix,feature.OfficeDesktopToHelix,feature.M365TeamsHubToHelix," +
	"feature.OwaHubToHelix,feature.MonarchHubToHelix,feature.Win32OutlookHubToHelix," +
	"feature.MacOutlookHubToHelix,Agt_bizchat_enableGpt5ForHelix"

// foldStreamText folds one incoming piece of streamed text — a token delta
// (passed as answer+writeAtCursor) or a full-text snapshot — into the running
// answer, returning the new answer and the suffix to stream.
//
// The service mixes token deltas with full-text snapshots, and the first token
// often arrives only as a snapshot, so naive concatenation drops the head:
//   - next no longer than answer: no growth, emit nothing.
//   - next extends answer: adopt it, emit the appended suffix.
//   - next is anything else (not a prefix continuation): adopt it as the
//     authoritative buffer but emit nothing, since already-streamed bytes can't
//     be retracted.
//
// Every emitted suffix is thus a true prefix of the final answer.
func foldStreamText(answer, next string) (string, string, bool) {
	if len(next) <= len(answer) {
		return answer, "", false
	}
	if strings.HasPrefix(next, answer) {
		return next, next[len(answer):], true
	}
	return next, "", false
}

// Throttle reports the conversation's message quota usage.
type Throttle struct {
	Current int
	Max     int
}

// CopilotSession is a persistent conversation. It reuses the same
// session/conversation IDs across turns and opens a fresh WebSocket per message.
type CopilotSession struct {
	sessionID      string
	conversationID string
	agentID        string
	wantedAgent    bool
	hasTools       bool
	turnCount      int
}

// CopilotSessionOptions configures a new session. Empty IDs are generated.
type CopilotSessionOptions struct {
	AgentID        string
	SessionID      string
	ConversationID string
	// WantedAgent records whether the caller asked for tool-agent behavior
	// this turn, independent of whether an agent id was actually resolved.
	WantedAgent bool
	// HasTools records whether the caller has real fenced tools to route
	// calls through this turn. This -- not WantedAgent -- gates the server's
	// own code interpreter: WantedAgent is false on a Claude-tone turn (which
	// never wants an M365 agent) even when real tools exist, and AgentID ==
	// "" is separately ambiguous on its own (it means either "no tools were
	// requested, agentless chat is correct" or "tools were requested but
	// getOrCreateAgent silently failed to provision one" -- model.go's Run
	// treats that failure as non-fatal and falls back agentless). Only when
	// there are truly no tools at all should the server's code interpreter
	// unlock; otherwise the model can silently answer from Microsoft's empty
	// sandbox filesystem instead of kdeps' real local-filesystem tools, which
	// still went out in the prompt and get ignored.
	HasTools bool
}

// NewCopilotSession creates a session, generating any IDs left unset.
func NewCopilotSession(opts CopilotSessionOptions) *CopilotSession {
	kdeps_debug.Log("enter: NewCopilotSession")
	s := &CopilotSession{
		sessionID:      opts.SessionID,
		conversationID: opts.ConversationID,
		agentID:        opts.AgentID,
		wantedAgent:    opts.WantedAgent,
		hasTools:       opts.HasTools,
	}
	if s.sessionID == "" {
		s.sessionID = uuid.NewString()
	}
	if s.conversationID == "" {
		s.conversationID = uuid.NewString()
	}
	return s
}

// TurnCount returns the number of turns sent on this session.
func (s *CopilotSession) TurnCount() int { return s.turnCount }

// Chat sends text and returns a stream of the answer. The returned stream's
// Deltas channel is closed when the turn ends; check Err afterwards.
func (s *CopilotSession) Chat(ctx context.Context, token, text, model string) (*CopilotStream, error) {
	kdeps_debug.Log("enter: Chat")

	isFirst := s.turnCount == 0
	s.turnCount++

	claims, err := DecodeJWT(token)
	if err != nil {
		return nil, err
	}
	requestID := uuid.NewString()

	params := url.Values{}
	params.Set("chatsessionid", requestID)
	params.Set("clientrequestid", requestID)
	params.Set("X-SessionId", s.sessionID)
	params.Set("ConversationId", s.conversationID)
	params.Set("access_token", token)
	params.Set("variants", variants)
	params.Set("source", "officeweb")
	params.Set("product", "office")
	params.Set("agentHost", "Bizchat.FullScreen")
	params.Set("licenseType", "Starter")
	params.Set("agent", "web")
	params.Set("scenario", "OfficeWebIncludedCopilot")

	wsURL := fmt.Sprintf("%s/%s@%s?%s", chatWSBase, claims.OID, claims.TID, params.Encode())

	header := http.Header{}
	header.Set("Origin", "https://m365.cloud.microsoft")
	header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:148.0) Gecko/20100101 Firefox/148.0")
	header.Set("Accept-Language", "en-US,en;q=0.9")
	header.Set("Cache-Control", "no-cache")
	header.Set("Pragma", "no-cache")

	conn, resp, err := websocket.DefaultDialer.DialContext(ctx, wsURL, header)
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		return nil, fmt.Errorf("m365: dial chat socket: %w", err)
	}

	stream := &CopilotStream{
		deltas:    make(chan string, deltaBufferSize),
		thinking:  make(chan string, deltaBufferSize),
		thinkOnce: map[string]bool{},
		maxScores: map[string]float64{},
	}

	args := s.buildChatArgs(requestID, text, model, isFirst)
	go stream.run(ctx, conn, args)
	return stream, nil
}

// buildChatArgs assembles the arguments[0] object of the chat invocation.
func (s *CopilotSession) buildChatArgs(requestID, text, model string, isFirst bool) map[string]any {
	optionsSets := []string{}
	if s.agentID == "" && !s.hasTools && os.Getenv("M365_NO_CODE_INTERPRETER") == "" {
		optionsSets = append(optionsSets, codeInterpreterOptionsSets...)
	}
	if extra := os.Getenv("M365_EXTRA_OPTIONSSETS"); extra != "" {
		for part := range strings.SplitSeq(extra, ",") {
			if p := strings.TrimSpace(part); p != "" {
				optionsSets = append(optionsSets, p)
			}
		}
	}

	args := map[string]any{
		"source":                   "officeweb",
		"clientCorrelationId":      requestID,
		"sessionId":                s.sessionID,
		"optionsSets":              optionsSets,
		"streamingMode":            "ConciseWithPadding",
		"spokenTextMode":           "None",
		"options":                  map[string]any{},
		"extraExtensionParameters": map[string]any{},
		"allowedMessageTypes": []string{
			"Chat", "Suggestion", "InternalSearchQuery", "Disengaged",
			"InternalLoaderMessage", "Progress", "RenderCardRequest", "SemanticSerp",
			"GenerateContentQuery", "SearchQuery", "ConfirmationCard", "DeveloperLogs",
			"EndOfRequest", "ReferencesListComplete", "GeneratedCode",
		},
		"sliceIds":         []string{},
		"threadLevelGptId": emptyToNil(s.agentID),
		"traceId":          requestID,
		"isStartOfSession": isFirst,
		"clientInfo": map[string]any{
			"clientPlatform":   "mcmcopilot-web",
			"clientAppName":    "office",
			"clientEntryPoint": "mcmcopilot-officeweb",
			"clientSessionId":  s.sessionID,
			"clientAppType":    "web",
			"deviceOs":         "linux",
			"deviceType":       "desktop",
		},
		"message": map[string]any{
			"author":                "user",
			"inputMethod":           "Keyboard",
			"text":                  text,
			"entityAnnotationTypes": []string{"People", "File", "Event", "Email", "TeamsMessage"},
			"requestId":             requestID,
			"locationInfo":          map[string]any{"timeZoneOffset": 1, "timeZone": "Europe/Copenhagen"},
			"locale":                "en-gb",
			"messageType":           "Chat",
			"experienceType":        "Default",
			"adaptiveCards":         []any{},
			"clientPreferences":     map[string]any{},
		},
		"isSbsSupported":            true,
		"tone":                      GetToneForModel(model),
		"renderReferencesBehindEOS": true,
		"disconnectBehavior":        "continue",
	}

	if s.agentID != "" {
		args["gpts"] = []map[string]any{{
			"id":      s.agentID,
			"source":  "MOS3",
			"version": "1.0.0",
			"clientOverrides": map[string]any{
				"capabilities":                  []any{},
				"deepResearchModels@odata.type": "Collection(String)",
			},
		}}
	} else {
		args["plugins"] = []map[string]any{{"Id": "BingWebSearch", "Source": "BuiltIn"}}
	}

	return args
}

// emptyToNil returns nil for an empty agent id so threadLevelGptId serializes
// as null rather than "".
func emptyToNil(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// nowISO returns an RFC3339 timestamp for the metrics frame.
func nowISO() string { return time.Now().UTC().Format("2006-01-02T15:04:05.000Z") }

// buildSendPayload combines the chat frame and metrics frame into one write.
func buildSendPayload(args map[string]any) (string, error) {
	chatMsg := map[string]any{
		"arguments":    []any{args},
		"invocationId": "0",
		"target":       "chat",
		"type":         frameStreamInvocation,
	}
	ts := nowISO()
	metrics := map[string]any{
		"arguments": []any{map[string]any{
			"Timestamps": map[string]any{
				"ConnectionStart":       ts,
				"UserInputStart":        ts,
				"ConnectionEstablished": ts,
				"UserInputSubmit":       ts,
			},
		}},
		"target": "metrics",
		"type":   frameInvocation,
	}
	chatJSON, err := json.Marshal(chatMsg)
	if err != nil {
		return "", err
	}
	metricsJSON, err := json.Marshal(metrics)
	if err != nil {
		return "", err
	}
	return string(chatJSON) + rs + string(metricsJSON) + rs, nil
}

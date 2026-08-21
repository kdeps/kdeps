package m365

import (
	"context"

	"github.com/google/uuid"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
)

// ModelSession is the stateful conversation layer above CopilotSession. It holds
// stable session/conversation ids, lazily resolves a server-side tool-calling
// agent, recreates the transport session when the agent-ness of a turn changes,
// and reconnects once on a transport error.
//
// Attaching an agent forces the server tone to GPT-5, so a caller that wants a
// real Claude_* tone must run with agents disabled (useAgent=false).
type ModelSession struct {
	sessionID      string
	conversationID string

	useAgent bool

	// resolveToken returns a chat access token; overridable for tests.
	resolveToken func(ctx context.Context) (string, error)
	// resolveAgent returns (or provisions) the tool-calling agent id.
	resolveAgent func(ctx context.Context, forceRefresh bool) (string, error)

	copilotSession *CopilotSession
	cachedAgentID  string
	// currentAgentID is the agent id the live copilotSession was built for
	// (empty means an agent-less session).
	currentAgentID string
	// currentWantedAgent is whether the live copilotSession was built for a
	// turn that wanted agent behavior, tracked separately from
	// currentAgentID because both can independently flip: e.g. agent
	// resolution keeps failing (currentAgentID stays "") while a caller that
	// previously ran tool-less now wants tools.
	currentWantedAgent bool
	// currentHasTools is whether the live copilotSession was built for a turn
	// that had real fenced tools available, tracked separately from
	// currentWantedAgent so a Claude-tone turn (which never wants an M365
	// agent) still suppresses the server's own code interpreter when it has
	// real tools to route through instead.
	currentHasTools bool
	agentResolved   bool
}

// ModelSessionOptions configures a ModelSession.
type ModelSessionOptions struct {
	// UseAgentSet distinguishes an explicit UseAgent=false from the zero value;
	// when false, UseAgent defaults to true.
	UseAgentSet bool
	UseAgent    bool
	// GetToken overrides the default token source (used in tests).
	GetToken func(ctx context.Context) (string, error)
	// GetAgent overrides the default agent resolver (used in tests).
	GetAgent func(ctx context.Context, forceRefresh bool) (string, error)
}

// NewModelSession creates a session with fresh session/conversation ids.
func NewModelSession(opts ModelSessionOptions) *ModelSession {
	kdeps_debug.Log("enter: NewModelSession")
	m := &ModelSession{
		sessionID:      uuid.NewString(),
		conversationID: uuid.NewString(),
		useAgent:       true,
		resolveToken:   opts.GetToken,
		resolveAgent:   opts.GetAgent,
	}
	if opts.UseAgentSet {
		m.useAgent = opts.UseAgent
	}
	if m.resolveToken == nil {
		m.resolveToken = getToken
	}
	if m.resolveAgent == nil {
		m.resolveAgent = getOrCreateAgent
	}
	return m
}

// ConversationID returns the current server conversation id.
func (m *ModelSession) ConversationID() string { return m.conversationID }

// TurnCount returns the number of turns sent on the live transport session.
func (m *ModelSession) TurnCount() int {
	if m.copilotSession == nil {
		return 0
	}
	return m.copilotSession.TurnCount()
}

// NewConversation rotates the conversation id and drops the transport session.
// Disengage/throttle state keys on the conversation id, so a clean retry needs a
// fresh one.
func (m *ModelSession) NewConversation() {
	kdeps_debug.Log("enter: ModelSession.NewConversation")
	m.conversationID = uuid.NewString()
	m.reset()
}

// reset drops the transport session and forgets the resolved agent.
func (m *ModelSession) reset() {
	m.copilotSession = nil
	m.cachedAgentID = ""
	m.currentAgentID = ""
	m.agentResolved = false
}

// Run sends one prompt and returns the streamed answer. wantAgentTurn lets a
// caller suppress the agent for a single turn (e.g. to keep a Claude_* tone).
// hasTools records whether the caller has real fenced tools this turn,
// independent of wantAgentTurn -- it keeps the server's own code interpreter
// off whenever real tools exist, even on a turn that doesn't want an M365
// agent.
func (m *ModelSession) Run(
	ctx context.Context, text, model string, wantAgentTurn, hasTools bool,
) (*CopilotStream, error) {
	kdeps_debug.Log("enter: ModelSession.Run")

	token, err := m.resolveToken(ctx)
	if err != nil {
		return nil, err
	}

	wantAgent := m.useAgent && wantAgentTurn
	agentForTurn := ""
	if wantAgent {
		if !m.agentResolved {
			// Failure to provision an agent is non-fatal: fall back to agent-less.
			if id, aerr := m.resolveAgent(ctx, false); aerr == nil {
				m.cachedAgentID = id
			}
			m.agentResolved = true
		}
		agentForTurn = m.cachedAgentID
	}

	if m.copilotSession == nil || m.currentAgentID != agentForTurn ||
		m.currentWantedAgent != wantAgent || m.currentHasTools != hasTools {
		m.copilotSession = NewCopilotSession(CopilotSessionOptions{
			AgentID:        agentForTurn,
			SessionID:      m.sessionID,
			ConversationID: m.conversationID,
			WantedAgent:    wantAgent,
			HasTools:       hasTools,
		})
		m.currentAgentID = agentForTurn
		m.currentWantedAgent = wantAgent
		m.currentHasTools = hasTools
	}

	stream, err := m.copilotSession.Chat(ctx, token, text, model)
	if err != nil {
		// Reconnect once with the same ids and retry.
		m.copilotSession = NewCopilotSession(CopilotSessionOptions{
			AgentID:        agentForTurn,
			SessionID:      m.sessionID,
			ConversationID: m.conversationID,
			WantedAgent:    wantAgent,
			HasTools:       hasTools,
		})
		m.currentAgentID = agentForTurn
		m.currentWantedAgent = wantAgent
		m.currentHasTools = hasTools
		return m.copilotSession.Chat(ctx, token, text, model)
	}
	return stream, nil
}

// RefreshAgent re-validates the tool-calling agent against the tenant, escaping
// the case where a cached agent id was deleted server-side. It returns true when
// the agent id changed, so the caller can resend the original prompt.
func (m *ModelSession) RefreshAgent(ctx context.Context) (bool, error) {
	kdeps_debug.Log("enter: ModelSession.RefreshAgent")
	if !m.useAgent {
		return false, nil
	}
	id, err := m.resolveAgent(ctx, true)
	if err != nil {
		return false, err
	}
	if id != m.cachedAgentID {
		m.cachedAgentID = id
		m.copilotSession = nil
		m.currentAgentID = ""
		return true, nil
	}
	return false, nil
}

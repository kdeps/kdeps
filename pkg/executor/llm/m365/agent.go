package m365

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
)

// This file provisions the server-side Copilot Studio agent that makes emulated
// tool calling reliable. The agent is versioned by name: its display name embeds
// an 8-hex-char sha256 prefix of its instructions, because the Copilot Studio
// update API needs a changeToken only returned at create time. Editing the
// instructions therefore auto-provisions a fresh agent instead of updating one.

const (
	powerPlatformScope = "https://api.powerplatform.com/.default"
	bapScope           = "https://api.bap.microsoft.com/.default"

	agentBaseName    = "m365-tool-agent"
	agentDescription = "Auto-created agent for tool calling"

	envAPIBase        = ".df.environment.api.powerplatform.com"
	minimalBotsAPIVer = "2022-03-01-preview"
	ppUserAgent       = "PVA-Portal/1.0.0 (Web; ReactNative: false)"

	// botLanguageLCID is the US English locale id Copilot Studio expects.
	botLanguageLCID = 1033
	// envIDTailTrim is how many trailing chars some tenants drop from the env id.
	envIDTailTrim = 2

	// botIconBase64 is a 48x48 solid PNG required for publishing.
	botIconBase64 = "iVBORw0KGgoAAAANSUhEUgAAADAAAAAwCAIAAADYYG7QAAAAB3RJTUUH6AMbAAAoLbOJEAAAABl0RVh0Q29tbWVudABDcmVhdGVkIHdpdGggR0lNUFeBDhcAAAAoSURBVFjD7cExAQAAAMKg9U9tDB+gAAAAAAAAAAAAAAAAAAAAAAAA/BgwMAAB/0LuMgAAAABJRU5ErkJggg=="
)

// agentInstructions is the server-side prompt baked into the agent. It teaches
// only the fenced tool-call format; the behavioural framing lives in the
// per-request <tools> block so it can vary without re-provisioning the agent.
const agentInstructions = "You are the execution core of an automated agent. Your output is parsed by a program.\n\n" +
	"When the incoming message contains a <tools> block, you are in execution mode. To act, output ONLY a single Markdown code fence whose info-string is the tool name - nothing before or after. A fenced block is an ACTION the runtime executes immediately against a live system; it is never an example or illustration:\n" +
	"```<tool_name>\n<one \"key: value\" header line per scalar argument>\n\n<the body argument, if the tool defines one>\n```\n" +
	"The runtime returns the real result in a <tool_response> block - treat it as ground truth. Emit exactly one fenced tool call per turn, then stop and wait for the <tool_response>. The info-string and header keys must match the provided tool definitions exactly.\n\n" +
	"When the message has no <tools> block, respond normally as a helpful assistant in natural language."

// bapAPI is the Business Application Platform base URL, and envURLOverride pins
// the resolved environment URL. Both are vars (not consts) only so tests can
// point the provisioning flow at a local server.
//
//nolint:gochecknoglobals // effectively constant; overridden only in tests
var (
	bapAPI         = "https://api.bap.microsoft.com"
	envURLOverride string
)

// getInstructionsHash returns the 8-hex-char sha256 prefix of the instructions.
func getInstructionsHash() string {
	sum := sha256.Sum256([]byte(agentInstructions))
	return hex.EncodeToString(sum[:])[:8]
}

// getAgentName returns the instructions-versioned agent display name.
func getAgentName() string {
	return agentBaseName + "-" + getInstructionsHash()
}

// agentCacheFile returns the on-disk cache path, honouring M365_AGENT_CACHE_FILE.
func agentCacheFile() string {
	if p := os.Getenv("M365_AGENT_CACHE_FILE"); p != "" {
		return p
	}
	return filepath.Join(configDir(), "agent-id.json")
}

// cachedAgent is the persisted agent id plus the instructions hash it was built
// for, so a stale cache is rebuilt.
type cachedAgent struct {
	AgentID          string `json:"agentId"`
	BotID            string `json:"botId"`
	InstructionsHash string `json:"instructionsHash,omitempty"`
	CreatedAt        string `json:"createdAt"`
}

func loadCachedAgent() *cachedAgent {
	data, err := os.ReadFile(agentCacheFile())
	if err != nil {
		return nil
	}
	var c cachedAgent
	if json.Unmarshal(data, &c) != nil {
		return nil
	}
	return &c
}

func saveCachedAgent(c cachedAgent) {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return
	}
	path := agentCacheFile()
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	_ = os.WriteFile(path, data, 0o600)
}

// ppFetch issues a Copilot Studio / Power Platform request with the standard
// headers.
func ppFetch(ctx context.Context, method, url, token string, body []byte) (*http.Response, error) {
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, rdr)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Ms-User-Agent", ppUserAgent)
	return http.DefaultClient.Do(req)
}

// getEnvironmentURL discovers the tenant's default Power Platform environment URL.
func getEnvironmentURL(ctx context.Context, ppToken string) (string, error) {
	kdeps_debug.Log("enter: getEnvironmentURL")
	if envURLOverride != "" {
		return envURLOverride, nil
	}
	url := bapAPI + "/providers/Microsoft.BusinessAppPlatform/environments/~default?api-version=2023-06-01"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+ppToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("m365: BAP environments failed: %d", res.StatusCode)
	}
	var data struct {
		Name string `json:"name"`
	}
	if err = json.NewDecoder(res.Body).Decode(&data); err != nil {
		return "", err
	}

	envID := stripEnvName(data.Name)
	candidates := []string{
		"https://default" + envID + envAPIBase,
	}
	if len(envID) >= envIDTailTrim {
		candidates = append(candidates, "https://default"+envID[:len(envID)-envIDTailTrim]+envAPIBase)
	}

	for _, c := range candidates {
		probeReq, perr := http.NewRequestWithContext(ctx, http.MethodHead,
			c+"/copilotstudio/minimalBots/api?api-version="+minimalBotsAPIVer, nil)
		if perr != nil {
			continue
		}
		probeReq.Header.Set("Authorization", "Bearer "+ppToken)
		probe, perr := http.DefaultClient.Do(probeReq)
		if perr != nil {
			continue // DNS did not resolve
		}
		_ = probe.Body.Close()
		return c, nil // any response means the host resolved
	}
	return candidates[0], nil
}

// stripEnvName turns "Default-fa7f56d8-..." into a bare lowercase id.
func stripEnvName(name string) string {
	const prefixLen = len("Default-")
	s := name
	if len(s) >= prefixLen && strings.EqualFold(s[:prefixLen], "Default-") {
		s = s[prefixLen:]
	}
	out := make([]byte, 0, len(s))
	for i := range len(s) {
		ch := s[i]
		if ch == '-' {
			continue
		}
		if ch >= 'A' && ch <= 'Z' {
			ch += 'a' - 'A'
		}
		out = append(out, ch)
	}
	return string(out)
}

type minimalBot struct {
	BotID        string `json:"botId"`
	ShortBotName string `json:"shortBotName"`
}

func listBots(ctx context.Context, envURL, token string) ([]minimalBot, error) {
	kdeps_debug.Log("enter: listBots")
	res, err := ppFetch(ctx, http.MethodGet,
		envURL+"/copilotstudio/minimalBots/api?api-version="+minimalBotsAPIVer, token, nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("m365: list bots failed: %d", res.StatusCode)
	}
	var bots []minimalBot
	if err = json.NewDecoder(res.Body).Decode(&bots); err != nil {
		return nil, err
	}
	return bots, nil
}

// createBot provisions a new agent with the current instructions baked in and
// returns its bot id.
//
//nolint:funlen // one large static JSON request body
func createBot(ctx context.Context, envURL, token string) (string, error) {
	kdeps_debug.Log("enter: createBot")
	name := getAgentName()
	body := map[string]any{
		"botComponentChanges": []any{
			map[string]any{
				"component": map[string]any{
					"diagnostics": []any{},
					"displayName": name,
					"id":          "00000000-0000-0000-0000-000000000000",
					"metadata": map[string]any{
						"tools":                []any{},
						"conversationStarters": []any{},
						"diagnostics":          []any{},
						"instructions": map[string]any{
							"$kind": "TemplateLine",
							"segments": []any{
								map[string]any{
									"$kind":       "TextSegment",
									"value":       agentInstructions,
									"diagnostics": []any{},
								},
							},
							"diagnostics": []any{},
						},
						"knowledgeSources": map[string]any{
							"diagnostics": []any{},
							"$kind":       "SearchAllKnowledgeSources",
						},
						"$kind": "GptComponentMetadata",
						"gptCapabilities": map[string]any{
							"diagnostics":                 []any{},
							"$kind":                       "GptCapabilities",
							"codeInterpreter":             false,
							"generateImages":              false,
							"webBrowsing":                 false,
							"searchOneDriveAndSharePoint": false,
							"searchTeams":                 false,
							"searchMeetings":              false,
							"searchEmails":                false,
							"searchPeople":                false,
						},
						"aISettings": map[string]any{
							"diagnostics":       []any{},
							"$kind":             "AISettings",
							"useModelKnowledge": true,
						},
					},
					"schemaName":  "00000000-0000-0000-0000-000000000000.gpt.default",
					"$kind":       "GptComponent",
					"description": agentDescription,
				},
				"$kind": "BotComponentInsert",
			},
		},
		"cloudFlowDefinitionChanges":                       []any{},
		"connectorDefinitionChanges":                       []any{},
		"environmentVariableChanges":                       []any{},
		"connectionReferenceChanges":                       []any{},
		"aIPluginOperationChanges":                         []any{},
		"componentCollectionChanges":                       []any{},
		"dataverseTableSearchChanges":                      []any{},
		"dataverseTableSearchEntityConfigurationChanges":   []any{},
		"dataverseTableSearchGlossaryConfigurationChanges": []any{},
		"dataverseTableSearchEntityColumnSynonymChanges":   []any{},
		"aIModelChanges":                                   []any{},
		"connectedAgentDefinitionChanges":                  []any{},
		"bot": map[string]any{
			"authorizedSecurityGroupIds": []any{},
			"supportedLanguages":         []any{},
			"diagnostics":                []any{},
			"displayName":                name,
			"language":                   botLanguageLCID,
			"schemaName":                 "00000000-0000-0000-0000-000000000000",
			"template":                   "gpt-1.1.0",
			"$kind":                      "BotEntity",
			"iconBase64":                 botIconBase64,
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	res, err := ppFetch(ctx, http.MethodPost,
		envURL+"/copilotstudio/minimalBots/api?api-version="+minimalBotsAPIVer, token, payload)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("m365: create bot failed: %d", res.StatusCode)
	}
	var data struct {
		Bot struct {
			SchemaName string `json:"schemaName"`
			CdsBotID   string `json:"cdsBotId"`
		} `json:"bot"`
	}
	if err = json.NewDecoder(res.Body).Decode(&data); err != nil {
		return "", err
	}
	if data.Bot.SchemaName != "" {
		return data.Bot.SchemaName, nil
	}
	return data.Bot.CdsBotID, nil
}

// publishBot publishes the bot to Copilot and returns its TitleId.
func publishBot(ctx context.Context, envURL, token, botID string) (string, error) {
	kdeps_debug.Log("enter: publishBot")
	res, err := ppFetch(ctx, http.MethodPost,
		envURL+"/copilotstudio/minimalBots/api/"+botID+"/publish?api-version="+minimalBotsAPIVer, token, nil)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("m365: publish bot failed: %d", res.StatusCode)
	}
	var data struct {
		TitleID string `json:"TitleId"`
	}
	if err = json.NewDecoder(res.Body).Decode(&data); err != nil {
		return "", err
	}
	if data.TitleID == "" {
		return "", errors.New("m365: publish response missing TitleId")
	}
	return data.TitleID, nil
}

func deleteBot(ctx context.Context, envURL, token, botID string) {
	res, err := ppFetch(ctx, http.MethodDelete,
		envURL+"/copilotstudio/minimalBots/api/"+botID+"?api-version="+minimalBotsAPIVer, token, nil)
	if err == nil {
		_ = res.Body.Close()
	}
}

// getOrCreateAgent returns the tool-calling agent id for the current
// instructions, provisioning it if needed. It returns "" (no error) when agent
// creation is not possible, so callers can fall back to agent-less chat.
//
//nolint:gocognit,nilerr // provisioning failure is non-fatal: fall back to agent-less chat
func getOrCreateAgent(ctx context.Context, forceRefresh bool) (string, error) {
	kdeps_debug.Log("enter: getOrCreateAgent")
	wantHash := getInstructionsHash()
	wantName := getAgentName()

	// Fast path: a cached agent built from the same instructions. Skipped on
	// forceRefresh, which re-validates against the tenant to escape a cached id
	// that was deleted server-side.
	if !forceRefresh {
		if c := loadCachedAgent(); c != nil && c.InstructionsHash == wantHash {
			return c.AgentID, nil
		}
	}

	bapToken, err := getTokenForScope(ctx, []string{bapScope})
	if err != nil || bapToken == "" {
		return "", nil
	}
	ppToken, err := getTokenForScope(ctx, []string{powerPlatformScope})
	if err != nil || ppToken == "" {
		return "", nil
	}

	envURL, err := getEnvironmentURL(ctx, bapToken)
	if err != nil {
		return "", nil
	}

	bots, err := listBots(ctx, envURL, ppToken)
	if err != nil {
		return "", nil
	}
	botID := ""
	for _, b := range bots {
		if b.ShortBotName == wantName {
			botID = b.BotID
			break
		}
	}
	if botID == "" {
		botID, err = createBot(ctx, envURL, ppToken)
		if err != nil {
			return "", nil
		}
	}

	titleID, err := publishBot(ctx, envURL, ppToken, botID)
	if err != nil {
		// A legacy bot missing icon/instructions can't publish; delete, recreate,
		// republish once.
		deleteBot(ctx, envURL, ppToken, botID)
		botID, err = createBot(ctx, envURL, ppToken)
		if err != nil {
			return "", nil
		}
		titleID, err = publishBot(ctx, envURL, ppToken, botID)
		if err != nil {
			return "", nil
		}
	}

	agentID := titleID + "." + botID + ".gpt.default"
	saveCachedAgent(cachedAgent{
		AgentID:          agentID,
		BotID:            botID,
		InstructionsHash: wantHash,
		CreatedAt:        time.Now().UTC().Format(time.RFC3339),
	})
	return agentID, nil
}

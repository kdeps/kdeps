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

package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kdeps/kdeps/v2/pkg/config"
	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
	executorLLM "github.com/kdeps/kdeps/v2/pkg/executor/llm"
	"github.com/kdeps/kdeps/v2/pkg/knowledge"
	kdepstools "github.com/kdeps/kdeps/v2/pkg/tools"
)

// knowledgeRecallMarker starts the per-turn recall system item.
const knowledgeRecallMarker = "<knowledge-recall>"

// errKnowledgeOff is what the knowledge tools return while /knowledge off.
var errKnowledgeOff = errors.New("the knowledge vault is off for this session (/knowledge on)")

// knowledgeState is the loop's view of the configured knowledge vault.
type knowledgeState struct {
	ctx      context.Context // parent of background builds (the session's root)
	cfg      config.KnowledgeConfig
	vault    knowledge.Vault
	ws       knowledge.Workspace
	dir      string // generated graph files, under ~/.kdeps
	mu       sync.Mutex
	last     knowledge.Recall
	building atomic.Bool
	builds   sync.WaitGroup // background builds in flight
	// notedTurns is the session turn count the last session note covered.
	notedTurns int
}

// newKnowledgeState returns nil without a parent context or knowledge
// config (library and test loops) or when it is disabled. An empty vault means the default
// (~/.kdeps/notes); a vault that does not exist yet is set up.
func newKnowledgeState(ctx context.Context, cfg *config.KnowledgeConfig) *knowledgeState {
	if ctx == nil || cfg == nil || (cfg.Enabled != nil && !*cfg.Enabled) {
		return nil
	}
	home, _ := os.UserHomeDir()
	vault := knowledge.DefaultVault()
	if strings.TrimSpace(cfg.Vault) != "" {
		vault = knowledge.ExpandPath(cfg.Vault, home)
	}
	ws := vault
	if cfg.Workspace != "" {
		ws = knowledge.ExpandPath(cfg.Workspace, home)
	}
	k := &knowledgeState{
		ctx:   ctx,
		cfg:   *cfg,
		vault: knowledge.Vault{Root: vault},
		ws:    knowledge.Workspace{Root: ws},
		dir:   knowledge.GraphDir(vault),
	}
	if _, err := os.Stat(vault); errors.Is(err, os.ErrNotExist) {
		// First session: create the folders, KDEPS.md and schema.md. A
		// missing templates asset still leaves the folders.
		_, _ = knowledge.Init(k.vault, k.ws)
	}
	return k
}

// knowledgeOn reports whether the vault is configured and not switched off
// with /knowledge off.
func (l *Loop) knowledgeOn() bool {
	return l.knowledge != nil && !l.knowledgeOff
}

// SetKnowledge switches recall, session notes and the knowledge tools on or
// off for this loop (/knowledge on|off), refreshing the system preamble.
func (l *Loop) SetKnowledge(on bool) {
	l.knowledgeOff = !on
	l.InvalidateSystemPreamble()
}

// KnowledgeEnabled reports whether the knowledge vault is in use.
func (l *Loop) KnowledgeEnabled() bool { return l.knowledgeOn() }

func (k *knowledgeState) sessionNotes() bool {
	return k.cfg.SessionNotes == nil || *k.cfg.SessionNotes
}

// build rebuilds the graph (changed files only) unless a build is running.
func (k *knowledgeState) build() error {
	if !k.building.CompareAndSwap(false, true) {
		return errors.New("a knowledge build is already running")
	}
	defer k.building.Store(false)
	_, _, err := knowledge.Build(k.ctx, k.vault, k.ws, k.dir, time.Now())
	return err
}

// buildAsync rebuilds in the background, for after a note is written.
func (k *knowledgeState) buildAsync() {
	k.builds.Go(func() { _ = k.build() })
}

// waitBuilds waits for background builds to finish.
func (k *knowledgeState) waitBuilds() { k.builds.Wait() }

// idx returns the recall index, reloaded when graph.json changes.
func (k *knowledgeState) idx() *knowledge.Index { return knowledge.SharedIndex(k.dir) }

// recall finds the notes related to text.
func (k *knowledgeState) recall(text string) knowledge.Recall {
	ix := k.idx()
	if ix == nil {
		return knowledge.Recall{}
	}
	r := ix.Recall(text, k.cfg.Recall.Hops, k.cfg.Recall.BudgetTokens)
	k.mu.Lock()
	k.last = r
	k.mu.Unlock()
	return r
}

// overview lists every card's status and next step, read from the cards:
// the routing table's status overview, never stored anywhere else.
func (k *knowledgeState) overview() string {
	cards, err := k.ws.Cards()
	if err != nil || len(cards) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(
		"## Status overview (from the cards in " + k.ws.Root + "/records)\n\n| Card | Type | Status | Next | Updated |\n|---|---|---|---|---|\n",
	)
	for _, c := range cards {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n", c.Name, c.Type, c.Status, c.Next, c.Updated)
	}
	if wd, innerErr := os.Getwd(); innerErr == nil {
		if c := k.ws.CardForDir(wd); c != nil {
			fmt.Fprintf(&b, "\nThe working directory belongs to card %q (%s).\n", c.Name, c.File)
		}
	}
	return b.String()
}

// knowledgePreamble is the vault's part of the system preamble: where it is
// and the card overview. The routing table (KDEPS.md) arrives through
// instruction discovery.
func (l *Loop) knowledgePreamble() string {
	if !l.knowledgeOn() {
		return ""
	}
	k := l.knowledge
	s := fmt.Sprintf("<knowledge-vault>\nThe knowledge vault is %s; its routing table and cards are in %s. "+
		"Each turn may include a <knowledge-recall> list of the notes related to the request: read only those you need. "+
		"Status lives in cards (knowledge_card_get/update), the story in notes (knowledge_note_save for a new idea).\n",
		k.vault.Root, k.ws.Root)
	if o := k.overview(); o != "" {
		s += "\n" + o
	}
	return s + "</knowledge-vault>"
}

// instructionParts are the discovered instruction files (with the
// knowledge workspace's KDEPS.md) and the vault preamble.
func (l *Loop) instructionParts() []string {
	var extra []string
	if l.knowledgeOn() {
		extra = append(extra, l.knowledge.ws.Root)
	}
	var parts []string
	if instructions := discoverInstructions("", extra...); instructions != "" {
		parts = append(parts, instructions)
	}
	if kp := l.knowledgePreamble(); kp != "" {
		parts = append(parts, kp)
	}
	return parts
}

// knowledgeRecallItem is this turn's recall as a system item, or nil, and
// says on w (when set) how much it added.
func (l *Loop) knowledgeRecallItem(input string, w io.Writer) *domain.ScenarioItem {
	if !l.knowledgeOn() || strings.TrimSpace(input) == "" {
		return nil
	}
	k := l.knowledge
	r := k.recall(input)
	text := r.Format(k.vault.Root)
	if text == "" {
		return nil
	}
	recordContextSegment("knowledge", text)
	if w != nil {
		fmt.Fprintf(w, "recall: %d notes (~%d tok)\n", len(r.Hits), r.Tokens)
	}
	return &domain.ScenarioItem{Role: "system", Prompt: text}
}

// LastRecall is the most recent recall, for /recall.
func (l *Loop) LastRecall() (knowledge.Recall, string, bool) {
	if l.knowledge == nil {
		return knowledge.Recall{}, "", false
	}
	l.knowledge.mu.Lock()
	defer l.knowledge.mu.Unlock()
	return l.knowledge.last, l.knowledge.vault.Root, true
}

// writeSessionNote writes this session's note from a compaction summary
// and refreshes the graph. Errors are reported, never fatal.
func (l *Loop) writeSessionNote(summary string, w io.Writer) {
	k := l.knowledge
	if !l.knowledgeOn() || !k.sessionNotes() || strings.TrimSpace(summary) == "" {
		return
	}
	wd, _ := os.Getwd()
	id := l.sessionID
	if id == "" {
		id = l.sessionNoteID()
	}
	sn, err := knowledge.WriteSession(k.vault, k.ws, wd, id, summary, time.Now())
	if err != nil {
		if w != nil {
			fmt.Fprintf(w, "knowledge: session note not written: %v\n", err)
		}
		return
	}
	if l.session != nil {
		k.notedTurns = l.session.TurnCount()
	}
	if w != nil {
		fmt.Fprintf(w, "knowledge: session note %s\n", sn.File)
	}
	k.buildAsync()
}

// sessionNoteID names a session not saved to the store yet, stable for the
// loop's lifetime.
func (l *Loop) sessionNoteID() string {
	if l.noteID == "" {
		l.noteID = "unsaved-" + time.Now().Format("20060102-150405")
	}
	return l.noteID
}

// WriteSessionNoteNow writes the session note for the turns since the last
// one (at session end): one summary call over the session, the same call
// compaction makes. It does nothing when no note is due.
func (l *Loop) WriteSessionNoteNow(ctx context.Context, w io.Writer) {
	if !l.SessionNoteDue() {
		return
	}
	var fileOps []FileOpEntry
	if l.session != nil {
		fileOps = l.session.FileOps()
	}
	summary, err := l.summarizeConversation(ctx, serializeConversation(l.session.RawMessages(), fileOps))
	if err != nil {
		if w != nil {
			fmt.Fprintf(w, "knowledge: session note not written: %v\n", err)
		}
		return
	}
	l.writeSessionNote(summary, w)
}

// SessionNoteDue reports whether turns happened since the last session note.
func (l *Loop) SessionNoteDue() bool {
	k := l.knowledge
	return l.knowledgeOn() && k.sessionNotes() && l.session != nil && l.session.TurnCount() > k.notedTurns
}

// registerKnowledgeTools offers the card and note tools when a vault is set.
func (l *Loop) registerKnowledgeTools() {
	if l.registry == nil || l.knowledge == nil {
		return
	}
	k := l.knowledge
	l.registry.Register(defined(&kdepstools.Tool{
		Name: "knowledge_recall",
		Execute: l.whenKnowledgeOn(func(args map[string]any) (string, error) {
			q, _ := args[toolParamQuery].(string)
			if strings.TrimSpace(q) == "" {
				return "", errors.New("knowledge_recall: query is required")
			}
			if text := k.recall(q).Format(k.vault.Root); text != "" {
				return text, nil
			}
			return "No notes in the knowledge vault relate to that query.", nil
		}),
	}))
	l.registry.Register(defined(&kdepstools.Tool{
		Name: "knowledge_card_get",
		Execute: l.whenKnowledgeOn(func(args map[string]any) (string, error) {
			c, err := k.card(args)
			if err != nil {
				return "", err
			}
			return formatCard(c), nil
		}),
	}))
	l.registry.Register(defined(&kdepstools.Tool{
		Name: "knowledge_card_update",
		Execute: l.whenKnowledgeOn(func(args map[string]any) (string, error) {
			c, err := k.card(args)
			if err != nil {
				return "", err
			}
			status, _ := args["status"].(string)
			next, _ := args["next"].(string)
			if status == "" && next == "" {
				return "", errors.New("knowledge_card_update: give status and/or next")
			}
			if innerErr := c.Update(strings.TrimSpace(status), strings.TrimSpace(next), time.Now()); innerErr != nil {
				return "", innerErr
			}
			l.InvalidateSystemPreamble()
			return "Updated card.\n" + formatCard(c), nil
		}),
	}))
	l.registry.Register(defined(&kdepstools.Tool{
		Name: "knowledge_note_save",
		Execute: l.whenKnowledgeOn(func(args map[string]any) (string, error) {
			file, err := k.saveNote(args, time.Now())
			if err != nil {
				return "", err
			}
			k.buildAsync()
			return "Saved note " + file, nil
		}),
	}))
}

// whenKnowledgeOn makes a knowledge tool refuse while /knowledge off.
func (l *Loop) whenKnowledgeOn(fn func(map[string]any) (string, error)) func(map[string]any) (string, error) {
	return func(args map[string]any) (string, error) {
		if !l.knowledgeOn() {
			return "", errKnowledgeOff
		}
		return fn(args)
	}
}

// card resolves the card a tool call names, or the working directory's.
func (k *knowledgeState) card(args map[string]any) (*knowledge.Card, error) {
	if name, _ := args["name"].(string); strings.TrimSpace(name) != "" {
		return k.ws.Card(strings.TrimSpace(name))
	}
	wd, _ := os.Getwd()
	if c := k.ws.CardForDir(wd); c != nil {
		return c, nil
	}
	return nil, errors.New("no card covers the working directory; pass name")
}

func formatCard(c *knowledge.Card) string {
	var b strings.Builder
	fmt.Fprintf(
		&b,
		"card: %s (%s)\ntype: %s\nstatus: %s\nnext: %s\nupdated: %s\n",
		c.Name,
		c.File,
		c.Type,
		c.Status,
		c.Next,
		c.Updated,
	)
	for _, l := range c.Links {
		fmt.Fprintf(&b, "%s: %s\n", l.Source, l.Path)
	}
	return b.String()
}

// saveNote writes a new one-idea note with frontmatter. It refuses a title
// that already has a note: one home per fact, so update that note instead.
func (k *knowledgeState) saveNote(args map[string]any, now time.Time) (string, error) {
	str := func(key string) string { v, _ := args[key].(string); return strings.TrimSpace(v) }
	title, summary, body := str("title"), str("summary"), str("body")
	if title == "" || summary == "" || body == "" {
		return "", errors.New("knowledge_note_save: title, summary and body are required")
	}
	if strings.ContainsAny(title, `/\:*?"<>|`) {
		return "", errors.New("knowledge_note_save: title cannot contain / \\ : * ? \" < > |")
	}
	folder := k.vault.Folder(knowledge.RoleResources)
	if f := str("folder"); f != "" {
		folder = k.noteFolder(f)
		if folder == "" {
			return "", fmt.Errorf("knowledge_note_save: folder %q is outside the vault", f)
		}
	}
	notes, err := k.vault.Walk()
	if err != nil {
		return "", err
	}
	for _, n := range notes {
		if strings.EqualFold(n.Title, title) {
			return "", fmt.Errorf("knowledge_note_save: %s already exists (%s); edit it instead - one home per fact",
				title, filepath.Join(k.vault.Root, filepath.FromSlash(n.Path)))
		}
	}
	typ := str("type")
	if typ == "" {
		typ = "note"
	}
	var fm strings.Builder
	fmt.Fprintf(
		&fm,
		"---\ntype: %s\nstatus: draft\ncreated: %s\nsummary: %q\n",
		typ,
		now.Format(time.DateOnly),
		summary,
	)
	if tags := splitList(str("tags")); len(tags) > 0 {
		fmt.Fprintf(&fm, "tags: [%s]\n", strings.Join(tags, ", "))
	}
	if links := splitList(str("links")); len(links) > 0 {
		quoted := make([]string, len(links))
		for i, l := range links {
			quoted[i] = fmt.Sprintf("%q", "[["+strings.Trim(l, "[]")+"]]")
		}
		fmt.Fprintf(&fm, "links: [%s]\n", strings.Join(quoted, ", "))
	}
	fm.WriteString("---\n\n# " + title + "\n\n" + body + "\n")
	dir := filepath.Join(k.vault.Root, filepath.FromSlash(folder))
	if innerErr := os.MkdirAll(dir, 0o750); innerErr != nil {
		return "", innerErr
	}
	file := filepath.Join(dir, title+".md")
	return file, os.WriteFile(file, []byte(fm.String()), 0o600)
}

// noteFolder maps a PARA role or a vault-relative folder to a vault folder,
// "" when it would leave the vault.
func (k *knowledgeState) noteFolder(f string) string {
	for _, role := range knowledge.Roles {
		if strings.EqualFold(f, role) {
			return k.vault.Folder(role)
		}
	}
	clean := path.Clean(filepath.ToSlash(f))
	if path.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../") {
		return ""
	}
	return clean
}

func splitList(s string) []string {
	var out []string
	for _, x := range strings.Split(s, ",") {
		if x = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(x), "#")); x != "" {
			out = append(out, x)
		}
	}
	return out
}

// cmdRecall shows the last turn's recall (/recall) or recalls for a query
// (/recall <query>).
func (r *REPL) cmdRecall(args []string) {
	k := r.loop.knowledge
	if k == nil {
		fmt.Fprintln(os.Stdout, styleReplMeta.Render(
			"The knowledge vault is disabled in ~/.kdeps/config.yaml (knowledge.enabled: false)."))
		return
	}
	if r.loop.knowledgeOff {
		fmt.Fprintln(os.Stdout, styleReplMeta.Render("The knowledge vault is off (/knowledge on)."))
		return
	}
	rec, _, _ := r.loop.LastRecall()
	if q := strings.TrimSpace(strings.Join(args, " ")); q != "" {
		rec = k.recall(q)
	}
	if len(rec.Hits) == 0 {
		fmt.Fprintln(os.Stdout, styleReplMeta.Render("Nothing recalled."))
		return
	}
	fmt.Fprintln(os.Stdout, styleReplMeta.Render("matched: "+strings.Join(rec.Seeds, ", ")))
	for _, h := range rec.Hits {
		line := fmt.Sprintf("  %s  (score %.2f", filepath.Join(k.vault.Root, filepath.FromSlash(h.Note)), h.Score)
		if h.Via != "" {
			line += fmt.Sprintf(", %d hop from %s", h.Hops, h.Via)
		}
		fmt.Fprintln(os.Stdout, line+")")
		if h.Summary != "" {
			fmt.Fprintln(os.Stdout, styleReplDim.Render("      "+h.Summary))
		}
	}
	fmt.Fprintln(os.Stdout, styleReplMeta.Render(fmt.Sprintf("%d notes, ~%d tokens", len(rec.Hits), rec.Tokens)))
}

// cmdKnowledge shows or toggles the knowledge vault (/knowledge [on|off]).
// The choice persists across sessions.
func (r *REPL) cmdKnowledge(args []string) {
	k := r.loop.knowledge
	if k == nil {
		fmt.Fprintln(os.Stdout, styleReplMeta.Render(
			"The knowledge vault is disabled in ~/.kdeps/config.yaml (knowledge.enabled: false)."))
		return
	}
	if len(args) == 0 {
		state := "on"
		if !r.loop.knowledgeOn() {
			state = "off - /knowledge on to enable"
		}
		fmt.Fprintln(os.Stdout, styleReplMeta.Render(fmt.Sprintf(
			"knowledge vault is %s\n  vault: %s\n  graph: %s", state, k.vault.Root, knowledge.GraphPath(k.dir))))
		return
	}
	switch args[0] {
	case "extract":
		fmt.Fprintln(os.Stdout, styleReplMeta.Render("Reading changed notes with the local model..."))
		r.loop.ExtractKnowledge(r.ctx, os.Stdout, true)
	case toggleOn, toggleOff:
		on := args[0] == toggleOn
		r.loop.SetKnowledge(on)
		r.persistTuning()
		if on {
			k.buildAsync()
			fmt.Fprintln(os.Stdout, styleReplSuccess.Render(
				"knowledge vault on - each turn recalls related notes; this session writes its note"))
		} else {
			fmt.Fprintln(os.Stdout, styleReplSuccess.Render(
				"knowledge vault off - no recall, no session note, knowledge tools refuse"))
		}
	default:
		fmt.Fprintln(os.Stderr, styleReplError.Render("Usage: /knowledge [on|off|extract]"))
	}
}

// Extraction defaults.
const (
	knowledgeExtractActionID = "knowledge_extract"
	defaultExtractPerSession = 20
	knowledgeExtractTimeout  = 3 * time.Minute
	// knowledgeExtractCallTimeout bounds one extraction call.
	knowledgeExtractCallTimeout = "180s"
	// knowledgeExtractMaxTokens caps one extraction reply.
	knowledgeExtractMaxTokens = 1024
)

// KnowledgeAsk returns an ExtractFunc that asks a model one question
// through engines from newEngine (one per call, so calls can run in
// parallel), as a single-resource synthetic workflow with no tools and no
// recall.
func KnowledgeAsk(newEngine func() *executor.Engine, model, backend, baseURL string) knowledge.ExtractFunc {
	return func(ctx context.Context, system, user string) (string, error) {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		off := false
		chatCfg := &domain.ChatConfig{
			Model: model, Backend: backend, BaseURL: baseURL,
			Prompt:        user,
			LiteralPrompt: true,
			// A local model reading a long note, maybe while loading, needs
			// more than the chat default.
			Timeout:   knowledgeExtractCallTimeout,
			Knowledge: &off,
			Scenario:  []domain.ScenarioItem{{Role: "system", Prompt: system}},
		}
		// The reply is a small JSON object; a cap stops a small model that
		// rambles from holding a local server for minutes.
		maxTokens := knowledgeExtractMaxTokens
		chatCfg.MaxTokens = &maxTokens
		wf := &domain.Workflow{
			APIVersion: "kdeps.io/v1",
			Kind:       "Workflow",
			Metadata: domain.WorkflowMetadata{
				Name: "knowledge", Version: "0.0.0", TargetActionID: knowledgeExtractActionID,
			},
			Resources: []*domain.Resource{
				{ActionID: knowledgeExtractActionID, Name: "knowledge_extract", Chat: chatCfg},
			},
		}
		result, err := newEngine().Execute(wf, nil)
		if err != nil {
			return "", err
		}
		return formatLoopResult(result), nil
	}
}

// extractModel is the model extraction uses: the session's when it is a
// local model being served, else the configured one; "" skips extraction.
func (l *Loop) extractModel() (string, string, string) {
	k := l.knowledge
	if k.cfg.Extract.Model != "" {
		return k.cfg.Extract.Model, k.cfg.Extract.Backend, ""
	}
	switch l.config.Backend {
	case executorLLM.BackendFile, executorLLM.BackendGGUF, "ollama":
		if l.config.Backend == "ollama" || l.config.BaseURL != "" {
			return l.config.Model, l.config.Backend, l.config.BaseURL
		}
	}
	return "", "", ""
}

// ExtractKnowledge runs the model pass over notes that changed since their
// last extraction (at most per_session of them, unless all is set) and
// rebuilds the graph with what it found. It reports on w.
func (l *Loop) ExtractKnowledge(ctx context.Context, w io.Writer, all bool) {
	k := l.knowledge
	if !l.knowledgeOn() || (k.cfg.Extract.Enabled != nil && !*k.cfg.Extract.Enabled) ||
		l.config.KnowledgeEngine == nil {
		return
	}
	model, backend, baseURL := l.extractModel()
	if model == "" {
		if all && w != nil {
			fmt.Fprintln(w, "knowledge: no local model for extraction (set knowledge.extract.model)")
		}
		return
	}
	limit := k.cfg.Extract.PerSession
	if limit <= 0 {
		limit = defaultExtractPerSession
	}
	if all {
		limit = 0
	}
	ctx, cancel := context.WithTimeout(ctx, knowledgeExtractTimeout)
	defer cancel()
	k.waitBuilds()
	st, err := knowledge.Extract(
		ctx,
		k.vault,
		k.dir,
		KnowledgeAsk(l.config.KnowledgeEngine, model, backend, baseURL),
		knowledge.ExtractOptions{
			Model:   model,
			Workers: k.cfg.Extract.Workers,
			Limit:   limit,
			System:  harnessText("knowledge-extract-system"),
		},
	)
	if err != nil {
		if w != nil {
			fmt.Fprintf(w, "knowledge: extraction failed: %v\n", err)
		}
		return
	}
	if st.Done+st.Failed == 0 {
		return
	}
	if buildErr := k.build(); buildErr != nil && w != nil {
		fmt.Fprintf(w, "knowledge: graph not rebuilt: %v\n", buildErr)
	}
	if w != nil {
		fmt.Fprintf(w, "knowledge: %s read %d notes (%d failed, %d still pending)\n",
			model, st.Done+st.Failed, st.Failed, st.Pending-st.Done-st.Failed)
	}
}

// HarnessText returns a standalone harness prompt by name (for callers
// outside the loop, such as kdeps knowledge build --extract).
func HarnessText(name string) string { return harnessText(name) }

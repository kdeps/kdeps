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
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// turo (Tagalog: "point") — stream editor that converts prose to compact
// kartographer graphs. When present, ALL system prompt content flows through
// turo: memory, skills, instructions, tool guidance — everything. Turo's
// graph output becomes the system prompt.
//
// Independent open-source project: github.com/kdeps/turo

//nolint:gochecknoglobals // process-wide probe; turo cannot change under a running process
var (
	turoOnce    sync.Once
	turoEnabled bool
	turoPath    string
)

// turoState holds the runtime turo settings mutated by the /turo command.
// level is passed to the turo binary; filler/synonyms/gloss toggle its pipeline
// stages (all on by default); off is a runtime override that disables reduction
// even when the binary is installed.
type turoSettings struct {
	mu       sync.Mutex
	level    string
	filler   bool
	synonyms bool
	gloss    bool
	arrows   bool
	off      bool
	init     bool
}

//nolint:gochecknoglobals // process-wide runtime settings for the turo reducer
var turoState turoSettings

// turoReduceCache memoizes reductions by raw input so unchanged history messages
// are not re-piped through turo on every turn. Cleared when the level changes,
// since the level alters the output.
//
//nolint:gochecknoglobals // process-wide reduction memo
var turoReduceCache sync.Map

const turoTimeout = 5 * time.Second

// turoValidLevels are the compression levels accepted by the turo binary.
//
//nolint:gochecknoglobals // static lookup set
var turoValidLevels = map[string]bool{
	"lite": true, "full": true, "ultra": true, "wenyan": true,
}

func turoInit() {
	turoState.mu.Lock()
	defer turoState.mu.Unlock()
	if turoState.init {
		return
	}
	lvl := strings.ToLower(strings.TrimSpace(os.Getenv("TURO_LEVEL")))
	if !turoValidLevels[lvl] {
		lvl = "ultra"
	}
	turoState.level = lvl
	// Stages default on (matching the turo binary); env can pre-disable them.
	turoState.filler = !turoEnvOff("TURO_FILLER")
	turoState.synonyms = !turoEnvOff("TURO_SYNONYMS")
	turoState.gloss = !turoEnvOff("TURO_GLOSS")
	turoState.arrows = !turoEnvOff("TURO_ARROWS") // on by default (matches the turo binary)
	turoState.init = true
}

// turoEnvOff reports whether an environment variable is set to a falsey value.
func turoEnvOff(name string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "0", "false", "no", "off":
		return true
	}
	return false
}

// TuroLevel returns the active compression level (lite, full, ultra).
func TuroLevel() string {
	turoInit()
	turoState.mu.Lock()
	defer turoState.mu.Unlock()
	return turoState.level
}

// SetTuroLevel sets the compression level. Returns false for an unknown level.
func SetTuroLevel(level string) bool {
	level = strings.ToLower(strings.TrimSpace(level))
	if !turoValidLevels[level] {
		return false
	}
	turoInit()
	turoState.mu.Lock()
	turoState.level = level
	turoState.mu.Unlock()
	turoReduceCache.Clear()
	return true
}

// turoValidStages are the pipeline stages toggled by /turo <stage> on|off.
//
//nolint:gochecknoglobals // static lookup set
var turoValidStages = map[string]bool{"filler": true, "synonyms": true, "gloss": true, "arrows": true}

// TuroStage reports whether a pipeline stage is enabled. Unknown stages report
// false.
func TuroStage(name string) bool {
	turoInit()
	turoState.mu.Lock()
	defer turoState.mu.Unlock()
	switch name {
	case "filler":
		return turoState.filler
	case "synonyms":
		return turoState.synonyms
	case "gloss":
		return turoState.gloss
	case "arrows":
		return turoState.arrows
	}
	return false
}

// SetTuroStage enables or disables a pipeline stage. Returns false for an
// unknown stage.
func SetTuroStage(name string, on bool) bool {
	if !turoValidStages[name] {
		return false
	}
	turoInit()
	turoState.mu.Lock()
	switch name {
	case "filler":
		turoState.filler = on
	case "synonyms":
		turoState.synonyms = on
	case "gloss":
		turoState.gloss = on
	case "arrows":
		turoState.arrows = on
	}
	turoState.mu.Unlock()
	turoReduceCache.Clear()
	return true
}

// TuroRuntimeOff reports whether the /turo off override is active.
func TuroRuntimeOff() bool {
	turoState.mu.Lock()
	defer turoState.mu.Unlock()
	return turoState.off
}

// SetTuroRuntimeOff toggles the runtime override that disables reduction.
func SetTuroRuntimeOff(off bool) {
	turoState.mu.Lock()
	turoState.off = off
	turoState.mu.Unlock()
}

func turoDisabled() bool {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("KDEPS_TURO")), "off") {
		return true
	}
	return strings.TrimSpace(os.Getenv("TURO_DISABLED")) == "1"
}

func turoAvailable(context.Context) bool {
	turoOnce.Do(func() { turoEnabled, turoPath = probeTuro() })
	return turoEnabled
}

// turoActive reports whether reduction should run: the binary is installed AND
// the runtime /turo off override is not set.
func turoActive(ctx context.Context) bool {
	return turoAvailable(ctx) && !TuroRuntimeOff()
}

func probeTuro() (bool, string) {
	if turoDisabled() {
		return false, ""
	}
	if custom := strings.TrimSpace(os.Getenv("KDEPS_TURO_PATH")); custom != "" {
		if _, err := os.Stat(custom); err == nil {
			return true, custom
		}
		return false, ""
	}
	path, err := exec.LookPath("turo")
	if err != nil {
		return false, ""
	}
	return true, path
}

// turoReduce pipes text through turo and returns the reduced content-word
// output. Returns the input unchanged on any failure — turo is optional.
func turoReduce(ctx context.Context, text string) string {
	if !turoActive(ctx) || turoPath == "" || text == "" {
		return text
	}
	turoInit()
	turoState.mu.Lock()
	level, filler, synonyms, gloss, arrows := turoState.level, turoState.filler, turoState.synonyms, turoState.gloss, turoState.arrows
	turoState.mu.Unlock()

	runCtx, cancel := context.WithTimeout(ctx, turoTimeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, turoPath,
		"-level", level,
		fmt.Sprintf("-filler=%t", filler),
		fmt.Sprintf("-synonyms=%t", synonyms),
		fmt.Sprintf("-gloss=%t", gloss),
		fmt.Sprintf("-arrows=%t", arrows),
	)
	cmd.Stdin = strings.NewReader(text)
	cmd.Dir = "/"
	out, err := cmd.Output()
	if err != nil {
		return text
	}
	result := strings.TrimSpace(string(out))
	if result == "" {
		return text
	}
	return result
}

// turoReduceCached is turoReduce memoized by raw input. Used for conversation
// history, where old messages repeat every turn and re-spawning turo for each
// would be wasteful.
func turoReduceCached(ctx context.Context, text string) string {
	if !turoActive(ctx) || text == "" {
		return text
	}
	if v, ok := turoReduceCache.Load(text); ok {
		if s, isStr := v.(string); isStr {
			return s
		}
	}
	reduced := turoReduce(ctx, text)
	turoReduceCache.Store(text, reduced)
	return reduced
}

// --- turo gain integration: read the turo binary's token savings log ---

// turoGainPath resolves the gain.jsonl path using the same precedence as the
// turo binary: TURO_HOME, then OS config dir, then ~/.turo.
func turoGainPath() string {
	if d := os.Getenv("TURO_HOME"); d != "" {
		return filepath.Join(d, "gain.jsonl")
	}
	if d, err := os.UserConfigDir(); err == nil && d != "" {
		return filepath.Join(d, "turo", "gain.jsonl")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".turo", "gain.jsonl")
}

// totalTuroSavings reads gain.jsonl and sums (before - after) across every
// recorded reduction. Returns 0 on any error — analytics must not break the REPL.
func totalTuroSavings() int {
	f, err := os.Open(turoGainPath())
	if err != nil {
		return 0
	}
	defer func() { _ = f.Close() }()
	var total int
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var e struct {
			Before int `json:"before"`
			After  int `json:"after"`
		}
		if json.Unmarshal([]byte(line), &e) == nil {
			total += e.Before - e.After
		}
	}
	_ = sc.Err()
	return total
}

// turo savings cache — re-reading gain.jsonl on every modeline render is
// wasteful; cache for 60 seconds.
var (
	turoSavingsMu     sync.Mutex
	turoSavingsCached int
	turoSavingsAt     time.Time
)

// TuroTokensSaved returns the total tokens saved according to the turo binary's
// gain log. Result is cached for 60 seconds to avoid re-reading on every
// status-line render.
func TuroTokensSaved() int {
	turoSavingsMu.Lock()
	defer turoSavingsMu.Unlock()
	if time.Since(turoSavingsAt) < 60*time.Second {
		return turoSavingsCached
	}
	n := totalTuroSavings()
	turoSavingsCached = n
	turoSavingsAt = time.Now()
	return n
}

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
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

const (
	defaultListLevelIndent = 2
	defaultTermWidth       = 100
	maxTermWidth           = 120
)

// pi-inspired color palette for markdown rendering.
const (
	colorHeading   = "#FFD60A" // yellow (pi mdHeading)
	colorLink      = "#81A2BE" // blue-gray (pi mdLink)
	colorCode      = "#00E5FF" // cyan (pi mdCode = accent)
	colorCodeBlock = "#A8FF78" // green (pi mdCodeBlock)
	colorText      = "#CDD6F4" // primary text
	colorThinking  = "#888888" // gray (pi thinkingText)
	colorMuted     = "#555555" // dim
	colorBullet    = "#00E5FF" // cyan accent (pi mdListBullet)
	colorQuote     = "#888888" // gray (pi mdQuote)
	colorBorderHr  = "#333333" // hr separator

	// Syntax highlight colors matching kdeps visual language.
	colorSyntaxKeyword  = "#FF79C6" // pink keywords
	colorSyntaxFunction = "#61AFEF" // blue functions
	colorSyntaxString   = "#A8FF78" // green strings
	colorSyntaxComment  = "#676767" // dim gray comments
	colorSyntaxNumber   = "#FFD60A" // yellow numbers
	colorSyntaxType     = "#00E5FF" // cyan types
	colorSyntaxOp       = "#EF8080" // red/salmon operators
)

// thinkingRe matches <thinking>...</thinking> blocks (including multiline).
var thinkingRe = regexp.MustCompile(`(?s)<thinking>(.*?)</thinking>`)

//nolint:gochecknoglobals // lipgloss style for thinking block header
var styleThinkingLabel = lipgloss.NewStyle().
	Foreground(lipgloss.Color(colorThinking)).
	Italic(true)

// replStyleConfig returns a glamour StyleConfig with pi-inspired colors.
//
//nolint:funlen // long but straightforward style table
func replStyleConfig() ansi.StyleConfig {
	margin := func() *uint { u := uint(1); return &u }()
	return ansi.StyleConfig{
		Document: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BlockPrefix: "\n",
				BlockSuffix: "\n",
				Color:       strp(colorText),
			},
			Margin: margin,
		},
		BlockQuote: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:  strp(colorQuote),
				Italic: boolp(true),
			},
			Indent:      func() *uint { u := uint(1); return &u }(),
			IndentToken: strp("| "),
		},
		Paragraph: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: strp(colorText),
			},
		},
		List: ansi.StyleList{
			LevelIndent: defaultListLevelIndent,
		},
		// Headings: styled without # markers (pi-style - text only, colored).
		Heading: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BlockSuffix: "\n",
				Color:       strp(colorHeading),
				Bold:        boolp(true),
			},
		},
		H1: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:     strp(colorHeading),
				Bold:      boolp(true),
				Underline: boolp(true),
			},
		},
		H2: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: strp(colorHeading),
				Bold:  boolp(true),
			},
		},
		H3: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:  strp(colorHeading),
				Italic: boolp(true),
			},
		},
		H4: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: strp(colorHeading),
			},
		},
		H5: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: strp(colorHeading),
			},
		},
		H6: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: strp(colorHeading),
			},
		},
		Emph: ansi.StylePrimitive{
			Italic: boolp(true),
			Color:  strp(colorText),
		},
		Strong: ansi.StylePrimitive{
			Bold:  boolp(true),
			Color: strp(colorText),
		},
		Strikethrough: ansi.StylePrimitive{
			CrossedOut: boolp(true),
		},
		HorizontalRule: ansi.StylePrimitive{
			Color:  strp(colorBorderHr),
			Format: "\n--------\n",
		},
		Item: ansi.StylePrimitive{
			BlockPrefix: "• ",
			Color:       strp(colorBullet),
		},
		Enumeration: ansi.StylePrimitive{
			Color: strp(colorBullet),
		},
		Task: ansi.StyleTask{
			StylePrimitive: ansi.StylePrimitive{Color: strp(colorText)},
			Ticked:         "[x] ",
			Unticked:       "[ ] ",
		},
		Link: ansi.StylePrimitive{
			Color:     strp(colorLink),
			Underline: boolp(true),
		},
		LinkText: ansi.StylePrimitive{
			Color: strp(colorLink),
			Bold:  boolp(true),
		},
		Image: ansi.StylePrimitive{
			Color:     strp(colorLink),
			Underline: boolp(true),
		},
		ImageText: ansi.StylePrimitive{
			Color:  strp(colorMuted),
			Format: "Image: {{.text}} ->",
		},
		Code: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix:          "`",
				Suffix:          "`",
				Color:           strp(colorCode),
				BackgroundColor: strp("#1A1A2E"),
			},
		},
		CodeBlock: ansi.StyleCodeBlock{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{
					Color: strp(colorCodeBlock),
				},
				Margin: margin,
			},
			Chroma: &ansi.Chroma{
				Text:    ansi.StylePrimitive{Color: strp(colorText)},
				Comment: ansi.StylePrimitive{Color: strp(colorSyntaxComment), Italic: boolp(true)},
				CommentPreproc: ansi.StylePrimitive{
					Color: strp(colorSyntaxOp),
				},
				Keyword:      ansi.StylePrimitive{Color: strp(colorSyntaxKeyword), Bold: boolp(true)},
				KeywordType:  ansi.StylePrimitive{Color: strp(colorSyntaxType)},
				Operator:     ansi.StylePrimitive{Color: strp(colorSyntaxOp)},
				Punctuation:  ansi.StylePrimitive{Color: strp(colorText)},
				NameBuiltin:  ansi.StylePrimitive{Color: strp(colorSyntaxFunction)},
				NameFunction: ansi.StylePrimitive{Color: strp(colorSyntaxFunction)},
				NameClass:    ansi.StylePrimitive{Color: strp(colorSyntaxType), Bold: boolp(true)},
				NameDecorator: ansi.StylePrimitive{
					Color: strp(colorSyntaxKeyword),
				},
				LiteralString:       ansi.StylePrimitive{Color: strp(colorSyntaxString)},
				LiteralStringEscape: ansi.StylePrimitive{Color: strp(colorCode)},
				LiteralNumber:       ansi.StylePrimitive{Color: strp(colorSyntaxNumber)},
				GenericDeleted:      ansi.StylePrimitive{Color: strp("#FF5F5F")},
				GenericInserted:     ansi.StylePrimitive{Color: strp(colorCodeBlock)},
				GenericSubheading:   ansi.StylePrimitive{Color: strp(colorHeading)},
			},
		},
		Table: ansi.StyleTable{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{Color: strp(colorText)},
			},
			CenterSeparator: strp("|"),
			ColumnSeparator: strp("|"),
			RowSeparator:    strp("-"),
		},
	}
}

// strp returns a pointer to s (used for glamour StylePrimitive string fields).
func strp(s string) *string { return &s }

// boolp returns a pointer to b (used for glamour StylePrimitive bool fields).
func boolp(b bool) *bool { return &b }

// terminalWidth returns the current terminal width, capped to a readable max.
func terminalWidth() int {
	w, _, err := term.GetSize(1)
	if err != nil || w <= 0 {
		return defaultTermWidth
	}
	if w > maxTermWidth {
		return maxTermWidth
	}
	return w
}

// renderThinkingBlock renders thinking block content in gray italic pi style.
// Shows a "* thinking" header followed by the markdown-rendered thinking text.
// Uses a muted gray style config so thinking content visually recedes from the main response.
func renderThinkingBlock(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	label := styleThinkingLabel.Render("* thinking")
	rendered := renderThinkingMarkdown(content)
	return label + "\n" + rendered + "\n"
}

// renderThinkingMarkdown renders markdown with a muted gray palette for thinking blocks.
// Uses gray tones throughout to avoid teal-on-teal contrast issues with the main style.
func renderThinkingMarkdown(text string) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(thinkingStyleConfig()),
		glamour.WithWordWrap(terminalWidth()),
	)
	if err != nil {
		return text
	}
	out, err := r.Render(text)
	if err != nil {
		return text
	}
	return trimTrailingSpaces(out)
}

// thinkingStyleConfig returns a glamour StyleConfig using muted grays for thinking blocks.
// All colors are gray-toned to ensure readability and visual separation from the main response.
func thinkingStyleConfig() ansi.StyleConfig {
	margin := func() *uint { u := uint(1); return &u }()
	const (
		textGray    = "#AAAAAA" // primary thinking text
		dimGray     = "#777777" // secondary / muted
		accentGray  = "#CCCCCC" // bold / emphasized
		codeGray    = "#BBBBBB" // inline code
		headingGray = "#CCCCCC" // headings
	)
	return ansi.StyleConfig{
		Document: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BlockPrefix: "\n",
				BlockSuffix: "\n",
				Color:       strp(textGray),
			},
			Margin: margin,
		},
		BlockQuote: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:  strp(dimGray),
				Italic: boolp(true),
			},
			Indent:      func() *uint { u := uint(1); return &u }(),
			IndentToken: strp("| "),
		},
		Paragraph: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: strp(textGray)},
		},
		List: ansi.StyleList{LevelIndent: defaultListLevelIndent},
		Heading: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BlockSuffix: "\n",
				Color:       strp(headingGray),
				Bold:        boolp(true),
			},
		},
		H1: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:     strp(accentGray),
				Bold:      boolp(true),
				Underline: boolp(true),
			},
		},
		H2: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: strp(accentGray), Bold: boolp(true)},
		},
		H3: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: strp(headingGray), Italic: boolp(true)},
		},
		H4: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: strp(headingGray)}},
		H5: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: strp(headingGray)}},
		H6: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: strp(dimGray)}},
		Emph: ansi.StylePrimitive{
			Italic: boolp(true),
			Color:  strp(textGray),
		},
		Strong: ansi.StylePrimitive{
			Bold:  boolp(true),
			Color: strp(accentGray),
		},
		Strikethrough: ansi.StylePrimitive{CrossedOut: boolp(true)},
		HorizontalRule: ansi.StylePrimitive{
			Color:  strp(dimGray),
			Format: "\n--------\n",
		},
		Item: ansi.StylePrimitive{
			BlockPrefix: "• ",
			Color:       strp(textGray),
		},
		Enumeration: ansi.StylePrimitive{Color: strp(textGray)},
		Task: ansi.StyleTask{
			StylePrimitive: ansi.StylePrimitive{Color: strp(textGray)},
			Ticked:         "[x] ",
			Unticked:       "[ ] ",
		},
		Link: ansi.StylePrimitive{
			Color:     strp(dimGray),
			Underline: boolp(true),
		},
		LinkText:  ansi.StylePrimitive{Color: strp(textGray), Bold: boolp(true)},
		Image:     ansi.StylePrimitive{Color: strp(dimGray), Underline: boolp(true)},
		ImageText: ansi.StylePrimitive{Color: strp(dimGray), Format: "Image: {{.text}} ->"},
		Code: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix: "`",
				Suffix: "`",
				Color:  strp(codeGray),
			},
		},
		CodeBlock: ansi.StyleCodeBlock{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{Color: strp(codeGray)},
				Margin:         margin,
			},
		},
		Table: ansi.StyleTable{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{Color: strp(textGray)},
			},
			CenterSeparator: strp("|"),
			ColumnSeparator: strp("|"),
			RowSeparator:    strp("-"),
		},
	}
}

// trimTrailingSpaces removes trailing whitespace from each line of s.
// Glamour's word-wrap can pad short lines to the wrap width; this strips that padding.
func trimTrailingSpaces(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return strings.Join(lines, "\n")
}

// renderMarkdown renders markdown text with the pi-inspired glamour theme.
// Falls back to plain text if rendering fails.
func renderMarkdown(text string) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(replStyleConfig()),
		glamour.WithWordWrap(terminalWidth()),
	)
	if err != nil {
		return text
	}
	out, err := r.Render(text)
	if err != nil {
		return text
	}
	return trimTrailingSpaces(out)
}

// renderREPLOutput renders a full LLM response for terminal display.
// Thinking blocks (<thinking>...</thinking>) are extracted and shown in gray
// italic above the main response. The remaining content is rendered as markdown.
func renderREPLOutput(text string) string {
	if text == "" {
		return ""
	}

	var sb strings.Builder

	matches := thinkingRe.FindAllStringSubmatchIndex(text, -1)
	if len(matches) > 0 {
		var body strings.Builder
		last := 0
		for _, loc := range matches {
			body.WriteString(text[last:loc[0]])
			sb.WriteString(renderThinkingBlock(text[loc[2]:loc[3]]))
			last = loc[1]
		}
		body.WriteString(text[last:])
		mainText := strings.TrimSpace(body.String())
		if mainText != "" {
			sb.WriteString(renderMarkdown(mainText))
		}
	} else {
		sb.WriteString(renderMarkdown(text))
	}

	result := strings.TrimRight(sb.String(), "\n")
	return result + "\n"
}

// renderToolCall returns a styled tool call display line.
// Format: [toolName -> args] with dim brackets and yellow tool name.
func renderToolCall(name, args string) string {
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color(colorMuted))
	tool := lipgloss.NewStyle().Foreground(lipgloss.Color(colorHeading))
	if args == "" {
		return dim.Render("[") + tool.Render(name) + dim.Render("]")
	}
	return dim.Render("[") + tool.Render(name) + dim.Render(" -> ") + tool.Render(args) + dim.Render("]")
}

// liveThinkingWriter streams reasoning tokens to stdout in real-time, then on
// Flush() erases the raw output and replaces it with a markdown-rendered version.
// This gives immediate token-by-token feedback during generation while still
// delivering properly formatted markdown when each round completes.
type liveThinkingWriter struct {
	buf        strings.Builder // accumulates full content for markdown render
	screenRows int             // screen rows consumed by raw output (for erase)
	started    bool
}

// screenRowsForText counts how many terminal rows the text occupies, accounting
// for word wrap at the terminal width. Used to erase the correct number of rows.
func screenRowsForText(text string) int {
	width := terminalWidth()
	rows := 1
	col := 0
	for _, ch := range text {
		if ch == '\n' {
			rows++
			col = 0
		} else {
			col++
			if col >= width {
				rows++
				col = 0
			}
		}
	}
	return rows
}

// Write streams the raw chunk to stdout immediately for real-time feedback and
// also buffers it for the markdown re-render on Flush().
func (w *liveThinkingWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	_, _ = w.buf.Write(p)
	if !w.started {
		hdr := styleThinkingLabel.Render("* thinking")
		fmt.Fprintf(os.Stdout, "\n%s\n  ", hdr)
		w.screenRows = 2 // header line + indent start
		w.started = true
	}
	// Stream raw text, indenting newlines for visual structure.
	text := strings.ReplaceAll(strings.TrimRight(string(p), "\n"), "\n", "\n  ")
	fmt.Fprint(os.Stdout, text)
	w.screenRows += screenRowsForText(text)
	return len(p), nil
}

// Flush erases the raw streamed output and replaces it with the markdown-rendered
// thinking block, then resets the writer for the next round.
func (w *liveThinkingWriter) Flush() {
	content := w.buf.String()
	w.buf.Reset()
	if !w.started {
		return
	}
	w.started = false
	// Erase raw output: move cursor up screenRows lines then clear to end of screen.
	if w.screenRows > 0 {
		fmt.Fprintf(os.Stdout, "\033[%dA\033[J", w.screenRows)
	}
	w.screenRows = 0
	if out := renderThinkingBlock(content); out != "" {
		fmt.Fprint(os.Stdout, out)
	}
}

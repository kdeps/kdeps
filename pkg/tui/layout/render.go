package layout

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Render draws the frame: each box with a rounded border, sized to its rect.
// main, feed, status and prompt are the box contents; text beyond a box is cut.
func Render(f Frame, header, main, feed, status, prompt string) string {
	top := box(f.Main, main)
	if f.Feed.W > 0 {
		top = lipgloss.JoinHorizontal(lipgloss.Top, top, box(f.Feed, feed))
	}
	head := lipgloss.NewStyle().Width(max(f.Header.W, 0)).MaxHeight(max(f.Header.H, 0)).Render(header)
	frame := lipgloss.JoinVertical(lipgloss.Left, head, top, box(f.Status, status), box(f.Prompt, prompt))
	return clampRows(frame, f.Prompt.Y+f.Prompt.H, f.Prompt.X+f.Prompt.W)
}

// clampRows cuts a frame to rows lines of at most width cells each. A line the
// terminal would wrap (a glyph it draws wider than it measures) would push every
// box below it down, so no row is allowed past the frame's size.
func clampRows(frame string, rows, width int) string {
	lines := strings.Split(frame, "\n")
	if len(lines) > rows {
		lines = lines[:rows]
	}
	for i, line := range lines {
		lines[i] = ansi.Truncate(line, width, "")
	}
	return strings.Join(lines, "\n")
}

// BoxPadX is the number of blank columns between a box border and its text,
// on each side.
const BoxPadX = 1

// InnerWidth is the columns of text a box of width w holds: the border and
// the padding on both sides are taken out.
func InnerWidth(w int) int { return max(w-2-2*BoxPadX, 0) }

// box draws content inside a rounded border of exactly r.W x r.H cells.
// box draws content inside a rounded border of exactly r.W x r.H cells.
func box(r Rect, content string) string {
	// Lip Gloss v2 counts the border inside Width and Height.
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		PaddingLeft(BoxPadX).
		PaddingRight(BoxPadX).
		Width(max(r.W, 0)).
		MaxWidth(r.W).
		Height(max(r.H, 0)).
		MaxHeight(r.H)
	return style.Render(strings.TrimRight(content, "\n"))
}

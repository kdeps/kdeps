package layout

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Render draws the frame: each box between a top and a bottom rule, sized to
// its rect. Boxes have no side borders.
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

// BoxPadX is the number of blank columns between a box edge and its text,
// on each side.
const BoxPadX = 1

// boxBorderRows is the rows a box's top and bottom border take.
const boxBorderRows = 2

// InnerWidth is the columns of text a box of width w holds: the padding on
// both sides is taken out.
func InnerWidth(w int) int { return max(w-2*BoxPadX, 0) }

// tabWidth is the spaces a tab becomes, as Lip Gloss would render it.
const tabWidth = 4

// box draws content between a top and a bottom rule, exactly r.W x r.H cells. Text
// is cut to the inside of the box first: a line Lip Gloss wrapped would add a
// row, and MaxHeight would then cut the bottom border instead of the text.
func box(r Rect, content string) string {
	content = fitInside(content, InnerWidth(r.W), max(r.H-boxBorderRows, 0))
	// Lip Gloss v2 counts the border inside Width and Height.
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder(), true, false).
		PaddingLeft(BoxPadX).
		PaddingRight(BoxPadX).
		Width(max(r.W, 0)).
		MaxWidth(r.W).
		Height(max(r.H, 0)).
		MaxHeight(r.H)
	return style.Render(strings.TrimRight(content, "\n"))
}

// fitInside keeps at most rows lines of content, each cut to width cells. A
// line whose text is cut ends in an ellipsis. Tabs become spaces first: a tab
// measures 0 cells but renders as tabWidth, so it would overflow the box.
func fitInside(content string, width, rows int) string {
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	if len(lines) > rows {
		lines = lines[:rows]
	}
	for i, line := range lines {
		line = strings.ReplaceAll(line, "\t", strings.Repeat(" ", tabWidth))
		// Only cut text gets an ellipsis: padding past the edge is just dropped.
		tail := ""
		if ansi.StringWidth(strings.TrimRight(ansi.Strip(line), " ")) > width {
			tail = "…"
		}
		lines[i] = ansi.Truncate(line, width, tail)
	}
	return strings.Join(lines, "\n")
}

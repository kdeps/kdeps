package layout

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// Render draws the frame: each box with a rounded border, sized to its rect.
// main, feed, status and prompt are the box contents; text beyond a box is cut.
func Render(f Frame, main, feed, status, prompt string) string {
	top := box(f.Main, main)
	if f.Feed.W > 0 {
		top = lipgloss.JoinHorizontal(lipgloss.Top, top, box(f.Feed, feed))
	}
	return lipgloss.JoinVertical(lipgloss.Left, top, box(f.Status, status), box(f.Prompt, prompt))
}

// box draws content inside a rounded border of exactly r.W x r.H cells.
func box(r Rect, content string) string {
	// Lip Gloss v2 counts the border inside Width and Height.
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Width(max(r.W, 0)).
		MaxWidth(r.W).
		Height(max(r.H, 0)).
		MaxHeight(r.H)
	return style.Render(strings.TrimRight(content, "\n"))
}

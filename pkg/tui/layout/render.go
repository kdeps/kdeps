package layout

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
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

// borderCells is the width and height a rounded border takes (one per side).
const borderCells = 2

// box draws content inside a rounded border of exactly r.W x r.H cells.
func box(r Rect, content string) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Width(max(r.W-borderCells, 0)).
		MaxWidth(r.W).
		Height(max(r.H-borderCells, 0)).
		MaxHeight(r.H)
	return style.Render(strings.TrimRight(content, "\n"))
}

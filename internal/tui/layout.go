package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/motoryang/velo-ssh/internal/term"
)

func (m Model) contentWidth() int {
	if m.width <= 0 {
		return 100
	}
	if m.width < 80 {
		return 80
	}
	return m.width
}

func borderedBlock(title string, width int, body []string) string {
	if width < 40 {
		width = 40
	}
	inner := width - 2
	lines := []string{topBorder(width, title)}
	for _, line := range body {
		lines = append(lines, "|"+padVisual(term.Truncate(line, inner), inner)+"|")
	}
	lines = append(lines, "+"+strings.Repeat("-", inner)+"+")
	return strings.Join(lines, "\n") + "\n"
}

func topBorder(width int, title string) string {
	inner := width - 2
	if title == "" {
		return "+" + strings.Repeat("-", inner) + "+"
	}
	label := " " + term.Truncate(title, inner-2) + " "
	labelWidth := lipgloss.Width(label)
	if labelWidth >= inner {
		return "+" + term.Truncate(label, inner) + "+"
	}
	left := (inner - labelWidth) / 2
	right := inner - labelWidth - left
	return "+" + strings.Repeat("-", left) + label + strings.Repeat("-", right) + "+"
}

func blankLine(width int) string {
	if width <= 0 {
		return ""
	}
	return strings.Repeat(" ", width)
}

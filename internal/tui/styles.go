package tui

import "github.com/charmbracelet/lipgloss"

type styles struct {
	title    lipgloss.Style
	selected lipgloss.Style
	muted    lipgloss.Style
	error    lipgloss.Style
}

func newStyles(ascii bool) styles {
	return styles{
		title:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")),
		selected: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42")),
		muted:    lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
		error:    lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
	}
}

package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/motoryang/velo-ssh/internal/term"
)

const settingsPanelWidth = 88

func (m Model) viewSettings() string {
	lines := m.settingsPanelLines()
	return centerBlock(lines, m.width, m.height)
}

func (m Model) settingsPanelLines() []string {
	innerWidth := settingsPanelWidth - 2
	lines := []string{
		"+" + strings.Repeat("-", innerWidth) + "+",
		"|" + centerPlain(m.tr(textSettingsTitle), innerWidth) + "|",
		"|" + centerPlain(m.tr(textSettingsGuide), innerWidth) + "|",
		"|" + strings.Repeat(" ", innerWidth) + "|",
	}
	for i, field := range m.settingsForm.fields {
		prefix := "  "
		if i == m.settingsForm.index {
			prefix = "> "
		}
		value := field.View()
		if _, ok := settingsFieldOptions[i]; ok {
			value = optionDisplay(i, field.Value())
		}
		row := prefix + padVisual(settingsLabel(i, m.config.Settings.Language), 26) + " " + value
		if i == m.settingsForm.index {
			row = m.styles.selected.Render(row)
		}
		lines = append(lines, "|"+padVisual(row, innerWidth)+"|")
	}
	lines = append(lines, "|"+strings.Repeat(" ", innerWidth)+"|")
	lines = append(lines, "|"+centerVisual(m.settingsButtons(), innerWidth)+"|")
	lines = append(lines, "+"+strings.Repeat("-", innerWidth)+"+")
	return lines
}

func (m Model) settingsButtons() string {
	ok := "[ " + m.tr(textSettingsOK) + " ]"
	cancel := "[ " + m.tr(textSettingsCancel) + " ]"
	if m.settingsForm.okFocused() {
		ok = m.styles.selected.Render(ok)
	}
	if m.settingsForm.cancelFocused() {
		cancel = m.styles.selected.Render(cancel)
	}
	return ok + "   " + cancel
}

func optionDisplay(field int, value string) string {
	return fmt.Sprintf("< %-10s >", optionLabel(field, value))
}

func centerBlock(lines []string, width, height int) string {
	if len(lines) == 0 {
		return ""
	}
	if width <= 0 {
		width = settingsPanelWidth
	}
	var out strings.Builder
	top := 0
	if height > len(lines) {
		top = (height - len(lines)) / 2
	}
	out.WriteString(strings.Repeat("\n", top))
	for _, line := range lines {
		lineWidth := lipgloss.Width(line)
		left := 0
		if width > lineWidth {
			left = (width - lineWidth) / 2
		}
		out.WriteString(strings.Repeat(" ", left))
		out.WriteString(line)
		out.WriteByte('\n')
	}
	return out.String()
}

func centerPlain(s string, width int) string {
	w := term.Width(s)
	if w >= width {
		return s
	}
	left := (width - w) / 2
	right := width - w - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}

func centerVisual(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	left := (width - w) / 2
	right := width - w - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}

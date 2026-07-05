package tui

import (
	"fmt"
	"strings"

	"github.com/motoryang/velo-ssh/internal/app"
)

func (m Model) withFooter(body, help string) string {
	body = strings.TrimRight(body, "\n")
	if help == "" {
		return body + "\n"
	}
	footer := m.footerBlock(help)
	if m.height <= 0 {
		return body + "\n\n" + footer + "\n"
	}
	lines := 0
	if body != "" {
		lines = strings.Count(body, "\n") + 1
	}
	footerLines := strings.Count(footer, "\n") + 1
	padding := m.height - lines - footerLines
	if padding < 1 {
		padding = 1
	}
	return body + strings.Repeat("\n", padding) + footer + "\n"
}

func (m Model) footerBlock(help string) string {
	lines := splitHelpLines(help)
	width := m.width
	if width <= 0 {
		width = 80
	}
	border := strings.Repeat("-", width)
	if len(lines) == 1 {
		return fmt.Sprintf("%s\n%s", border, lines[0])
	}
	return fmt.Sprintf("%s\n%s\n%s", border, lines[0], lines[1])
}

func splitHelpLines(help string) []string {
	parts := strings.Split(help, " | ")
	if len(parts) <= 1 {
		return []string{help}
	}
	mid := (len(parts) + 1) / 2
	return []string{
		strings.Join(parts[:mid], " | "),
		strings.Join(parts[mid:], " | "),
	}
}

func (m Model) helpText() string {
	switch m.state {
	case app.StateServerList:
		if m.searching {
			return "[Enter] Apply Filter | [Esc] Clear Filter"
		}
		return "[j/k] Move | [/] Filter | [Enter] Connect | [f] Files | [S] Settings | [a/e/d] Add/Edit/Delete | [q] Quit"
	case app.StateServerForm:
		return "[Tab/Down] Next | [Shift+Tab/Up] Previous | [Enter] Next/Save | [Esc] Cancel"
	case app.StateSettingsCenter:
		return ""
	case app.StateFileManager:
		if m.renaming {
			return "[Enter] Save Rename | [Esc] Cancel Rename"
		}
		if m.config.Settings.DefaultViewMode == "single" {
			return "[b] Show Local | [q] SSH Panel | [Enter] Open | [Space] Select | [r] Rename | [m] Toggle Time | [d] Download | [R] Refresh | [t] Tasks | [Esc] Back"
		}
		return "[Tab] Pane | [b] Hide Local | [q] SSH Panel | [Enter] Open | [Space] Select | [a] All | [c] Clear | [r] Rename | [u] Upload | [m] Toggle Time | [d] Download | [R] Refresh | [t] Tasks | [Esc] Back"
	case app.StateTaskCenter:
		return "[j/k] Move | [p] Pause | [r] Resume | [x] Cancel Task | [R] Refresh | [t]/[q]/[Esc] Back"
	case app.StateConfirmModal:
		if m.modalKind == modalHostKey {
			return "[Enter]/[y] Trust Host Key | [Esc]/[n] Cancel"
		}
		if m.modalKind == modalOverwrite {
			return "[Enter]/[y] Overwrite | [Esc]/[n] Cancel"
		}
		return "[Enter]/[y] Confirm | [Esc]/[n] Cancel"
	case app.StateShell:
		return "[Enter]/[o] Open Shell | [Esc] Server List"
	case app.StateHelp:
		return "[Esc]/[q] Back"
	default:
		return ""
	}
}

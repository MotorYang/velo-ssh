package tui

import (
	"fmt"
	"strings"

	"github.com/motoryang/velo-ssh/internal/app"
	"github.com/motoryang/velo-ssh/internal/term"
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
	return border + "\n" + strings.Join(lines, "\n")
}

func splitHelpLines(help string) []string {
	parts := strings.Split(help, " | ")
	if len(parts) <= 1 {
		return []string{help}
	}
	lines := make([]string, 0, 3)
	current := ""
	maxWidth := 96
	for _, part := range parts {
		candidate := part
		if current != "" {
			candidate = current + " | " + part
		}
		if term.Width(candidate) > maxWidth && current != "" {
			lines = append(lines, current)
			current = part
			continue
		}
		current = candidate
	}
	if current != "" {
		lines = append(lines, current)
	}
	if len(lines) == 0 {
		return []string{help}
	}
	return lines
}

func (m Model) helpText() string {
	switch m.state {
	case app.StateServerList:
		if m.searching {
			return m.tr(textFooterServerSearch)
		}
		return m.tr(textFooterServerList)
	case app.StateServerForm:
		return m.tr(textFooterServerForm)
	case app.StateSettingsCenter:
		return ""
	case app.StateFileManager:
		if m.fileSearching {
			return m.tr(textFooterFileSearch)
		}
		if m.renaming {
			return m.tr(textFooterRename)
		}
		if m.creatingDir {
			return m.tr(textFooterCreateDir)
		}
		if m.config.Settings.DefaultViewMode == "single" {
			return m.tr(textFooterFileSingle)
		}
		return m.tr(textFooterFileSplit)
	case app.StateTaskCenter:
		if m.taskDraftMode {
			return m.tr(textFooterStateTaskCenter)
		}
		return m.tr(textFooterTaskCenter)
	case app.StateConfirmModal:
		if m.modalKind == modalUpdateAvailable {
			return m.tr(textUpdateAction) + " | " + m.tr(textFooterCancel) + " | " + m.tr(textSkipUpdateAction)
		}
		if m.modalKind == modalUpdateInstalling {
			return m.tr(textUpdateCancelAction)
		}
		if m.modalKind == modalHostKey {
			return m.tr(textTrustAndRetry)
		}
		if m.modalKind == modalOverwrite {
			return m.tr(textOverwriteAction)
		}
		if m.modalKind == modalFileDelete {
			return m.tr(textDeleteAction)
		}
		if m.modalKind == modalTaskCancel {
			return m.tr(textCancelTaskAction) + " | " + m.tr(textKeepTaskAction)
		}
		if m.modalKind == modalServerFormDiscard {
			return m.tr(textFooterSettingsDiscard)
		}
		return m.tr(textFooterConfirm)
	case app.StateShell:
		return m.tr(textFooterStateShell)
	case app.StateHelp:
		return m.tr(textFooterBack)
	default:
		return ""
	}
}

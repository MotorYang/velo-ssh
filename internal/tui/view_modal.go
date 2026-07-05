package tui

import (
	"strings"

	"github.com/motoryang/velo-ssh/internal/term"
)

const (
	modalDelete            = "delete"
	modalHostKey           = "host_key"
	modalOverwrite         = "overwrite"
	modalFileDelete        = "file_delete"
	modalTaskCancel        = "task_cancel"
	modalServerFormDiscard = "server_form_discard"

	hostKeyActionShell       = "shell"
	hostKeyActionFileManager = "file_manager"
	hostKeyActionReconnect   = "reconnect"
)

const modalPanelWidth = 88

func (m Model) viewModal(message string) string {
	innerWidth := modalPanelWidth - 2
	lines := []string{
		"+" + strings.Repeat("-", innerWidth) + "+",
		"|" + centerVisual(m.styles.title.Render("Confirm"), innerWidth) + "|",
		"|" + strings.Repeat(" ", innerWidth) + "|",
	}
	for _, line := range strings.Split(message, "\n") {
		line = term.Truncate(line, innerWidth)
		lines = append(lines, "|"+padVisual(line, innerWidth)+"|")
	}
	lines = append(lines, "+"+strings.Repeat("-", innerWidth)+"+")
	height := m.height
	if height > 4 {
		height -= 4
	}
	return centerBlock(lines, m.width, height)
}

func (m Model) viewDeleteConfirm() string {
	return m.viewModal("Delete server " + m.deleteName + " (" + m.deleteID + ")?\n\nThis removes it from ~/.config/vssh/config.json.")
}

func (m Model) viewHostKeyConfirm() string {
	if m.hostKeyErr == nil {
		return m.viewModal("Missing host key confirmation context.")
	}
	srv := m.pendingHostKeyServer
	return m.viewModal(
		"Trust SSH host key for " + srv.Name + " (" + srv.Host + ")?\n\n" +
			"Target: " + m.hostKeyErr.Host + "\n" +
			"Fingerprint: " + m.hostKeyErr.Fingerprint + "\n" +
			"Known hosts: " + m.hostKeyErr.KnownHostsPath + "\n\n" +
			"Accept only if this fingerprint matches the server you expect.\n\n" +
			"[Enter]/[y] Trust and retry | [Esc]/[n] Cancel",
	)
}

func (m Model) viewOverwriteConfirm() string {
	targets := m.pendingOverwrite
	if len(targets) > 6 {
		targets = targets[:6]
	}
	message := "Overwrite existing target file(s)?\n\n"
	for _, target := range targets {
		message += "- " + target + "\n"
	}
	if len(m.pendingOverwrite) > len(targets) {
		message += "- ...\n"
	}
	message += "\nExisting targets are only replaced after the atomic transfer succeeds.\n\n"
	message += "[Enter]/[y] Overwrite | [Esc]/[n] Cancel"
	return m.viewModal(message)
}

func (m Model) viewFileDeleteConfirm() string {
	items := m.pendingFileDelete
	if len(items) > 6 {
		items = items[:6]
	}
	scope := "local"
	if m.pendingDeleteRemote {
		scope = "remote"
	}
	message := "Delete selected " + scope + " path(s)?\n\n"
	for _, item := range items {
		message += "- " + item.Path + "\n"
	}
	if len(m.pendingFileDelete) > len(items) {
		message += "- ...\n"
	}
	message += "\nThis operation cannot be undone by VeloSSH.\n\n"
	message += "[Enter]/[y] Delete | [Esc]/[n] Cancel"
	return m.viewModal(message)
}

func (m Model) viewTaskCancelConfirm() string {
	message := "Cancel and remove transfer task?\n\n"
	message += "Task: " + m.pendingTaskCancelID + "\n"
	if m.pendingTaskCancelName != "" {
		message += "Path: " + m.pendingTaskCancelName + "\n"
	}
	message += "\nThe running transfer is canceled, temporary files are removed when possible, and the task record is removed from the list.\n\n"
	message += "[Enter]/[y] Cancel Task | [Esc]/[n] Keep Task"
	return m.viewModal(message)
}

func (m Model) viewServerFormDiscardConfirm() string {
	return m.viewModal(
		"Discard unsaved server changes?\n\n" +
			"You have edited fields in this server form. Leaving now will lose those changes.\n\n" +
			"[Enter]/[y] Discard Changes | [Esc]/[n] Keep Editing",
	)
}

func (m Model) viewConfirmModal() string {
	if m.modalKind == modalHostKey {
		return m.viewHostKeyConfirm()
	}
	if m.modalKind == modalOverwrite {
		return m.viewOverwriteConfirm()
	}
	if m.modalKind == modalFileDelete {
		return m.viewFileDeleteConfirm()
	}
	if m.modalKind == modalTaskCancel {
		return m.viewTaskCancelConfirm()
	}
	if m.modalKind == modalServerFormDiscard {
		return m.viewServerFormDiscardConfirm()
	}
	return m.viewDeleteConfirm()
}

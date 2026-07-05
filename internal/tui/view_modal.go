package tui

import (
	"strings"

	"github.com/motoryang/velo-ssh/internal/term"
	"github.com/motoryang/velo-ssh/internal/version"
)

const (
	modalDelete            = "delete"
	modalHostKey           = "host_key"
	modalOverwrite         = "overwrite"
	modalFileDelete        = "file_delete"
	modalTaskCancel        = "task_cancel"
	modalServerFormDiscard = "server_form_discard"
	modalUpdateAvailable   = "update_available"

	hostKeyActionShell       = "shell"
	hostKeyActionFileManager = "file_manager"
	hostKeyActionReconnect   = "reconnect"
)

const modalPanelWidth = 88

func (m Model) viewModal(message string) string {
	innerWidth := modalPanelWidth - 2
	lines := []string{
		"+" + strings.Repeat("-", innerWidth) + "+",
		"|" + centerVisual(m.styles.title.Render(m.tr(textConfirmTitle)), innerWidth) + "|",
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
	return m.viewModal(m.tr(textDeleteServerPrompt) + " " + m.deleteName + " (" + m.deleteID + ")?\n\n" + m.tr(textDeleteServerBody))
}

func (m Model) viewHostKeyConfirm() string {
	if m.hostKeyErr == nil {
		return m.viewModal(m.tr(textMissingHostKeyContext))
	}
	srv := m.pendingHostKeyServer
	return m.viewModal(
		m.tr(textTrustHostKeyPrompt) + " " + srv.Name + " (" + srv.Host + ")?\n\n" +
			m.tr(textTarget) + ": " + m.hostKeyErr.Host + "\n" +
			"Fingerprint: " + m.hostKeyErr.Fingerprint + "\n" +
			m.tr(textKnownHosts) + ": " + m.hostKeyErr.KnownHostsPath + "\n\n" +
			m.tr(textHostKeyWarning) + "\n\n" +
			m.tr(textTrustAndRetry),
	)
}

func (m Model) viewOverwriteConfirm() string {
	targets := m.pendingOverwrite
	if len(targets) > 6 {
		targets = targets[:6]
	}
	message := m.tr(textOverwritePrompt) + "\n\n"
	for _, target := range targets {
		message += "- " + target + "\n"
	}
	if len(m.pendingOverwrite) > len(targets) {
		message += "- ...\n"
	}
	message += "\n" + m.tr(textOverwriteBody) + "\n\n"
	message += m.tr(textOverwriteAction)
	return m.viewModal(message)
}

func (m Model) viewFileDeleteConfirm() string {
	items := m.pendingFileDelete
	if len(items) > 6 {
		items = items[:6]
	}
	message := m.tr(textDeletePathsPrompt) + "\n\n"
	for _, item := range items {
		message += "- " + item.Path + "\n"
	}
	if len(m.pendingFileDelete) > len(items) {
		message += "- ...\n"
	}
	message += "\n" + m.tr(textDeletePathsBody) + "\n\n"
	message += m.tr(textDeleteAction)
	return m.viewModal(message)
}

func (m Model) viewTaskCancelConfirm() string {
	message := m.tr(textCancelTaskPrompt) + "\n\n"
	message += m.tr(textTask) + ": " + m.pendingTaskCancelID + "\n"
	if m.pendingTaskCancelName != "" {
		message += m.tr(textPath) + ": " + m.pendingTaskCancelName + "\n"
	}
	message += "\n" + m.tr(textCancelTaskBody) + "\n\n"
	message += m.tr(textCancelTaskAction) + " | " + m.tr(textKeepTaskAction)
	return m.viewModal(message)
}

func (m Model) viewServerFormDiscardConfirm() string {
	return m.viewModal(
		m.tr(textDiscardServerPrompt) + "\n\n" +
			m.tr(textDiscardServerBody) + "\n\n" +
			m.tr(textDiscardAction) + " | " + m.tr(textKeepEditingAction),
	)
}

func (m Model) viewUpdateAvailableConfirm() string {
	rel := m.pendingUpdate
	return m.viewModal(
		m.tr(textUpdateAvailablePrompt) + "\n\n" +
			m.tr(textUpdateCurrent) + ": " + version.String() + "\n" +
			m.tr(textUpdateLatest) + ": " + rel.Version + "\n\n" +
			m.tr(textUpdateBody) + "\n" +
			rel.URL + "\n\n" +
			m.tr(textUpdateAction) + " | " + m.tr(textFooterCancel) + " | " + m.tr(textSkipUpdateAction),
	)
}

func (m Model) viewConfirmModal() string {
	if m.modalKind == modalUpdateAvailable {
		return m.viewUpdateAvailableConfirm()
	}
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

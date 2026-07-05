package tui

const (
	modalDelete     = "delete"
	modalHostKey    = "host_key"
	modalOverwrite  = "overwrite"
	modalFileDelete = "file_delete"

	hostKeyActionShell       = "shell"
	hostKeyActionFileManager = "file_manager"
	hostKeyActionReconnect   = "reconnect"
)

func (m Model) viewModal(message string) string {
	return m.styles.title.Render("Confirm") + "\n\n" + message
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
	return m.viewDeleteConfirm()
}

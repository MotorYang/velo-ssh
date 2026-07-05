package tui

import "strings"

func (m Model) viewShell() string {
	var b strings.Builder
	b.WriteString(m.styles.title.Render("SSH Shell"))
	b.WriteString("\n\n")
	b.WriteString("Remote shell is connected.\n")
	b.WriteString("\n")
	b.WriteString("Local escape commands inside the shell:\n")
	b.WriteString("  :vssh files       switch to file manager\n")
	b.WriteString("  :vssh tasks       open task center\n")
	b.WriteString("  :vssh settings    open settings\n")
	b.WriteString("  :vssh back        return to server list\n")
	b.WriteString("  :vssh reconnect   reconnect current SSH session\n")
	b.WriteString("  :vssh quit        disconnect current session\n")
	b.WriteString("  :vssh send <text> send text to remote\n")
	return b.String()
}

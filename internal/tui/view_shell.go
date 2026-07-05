package tui

import (
	"fmt"
	"strings"

	"github.com/motoryang/velo-ssh/internal/term"
)

func (m Model) viewShell() string {
	width := m.width
	if width <= 0 {
		width = 88
	}
	if width < 60 {
		width = 60
	}
	title := m.shellFrameTitle()
	innerWidth := width - 2
	lines := []string{
		shellTopBorder(width, title),
		"|" + padVisual("Remote shell is connected.", innerWidth) + "|",
		"|" + strings.Repeat(" ", innerWidth) + "|",
		"|" + padVisual("Local escape commands:", innerWidth) + "|",
		"|" + padVisual("  :vssh files       switch to file manager", innerWidth) + "|",
		"|" + padVisual("  :vssh tasks       open task center", innerWidth) + "|",
		"|" + padVisual("  :vssh settings    open settings", innerWidth) + "|",
		"|" + padVisual("  :vssh back        return to server list", innerWidth) + "|",
		"|" + padVisual("  :vssh reconnect   reconnect current SSH session", innerWidth) + "|",
		"|" + padVisual("  :vssh quit        disconnect current session", innerWidth) + "|",
		"|" + padVisual("  :vssh send <text> send text to remote", innerWidth) + "|",
		"+" + strings.Repeat("-", innerWidth) + "+",
	}
	return strings.Join(lines, "\n") + "\n"
}

func (m Model) shellFrameTitle() string {
	name := strings.TrimSpace(m.activeServer.Name)
	if name == "" {
		name = strings.TrimSpace(m.activeServer.ID)
	}
	if name == "" {
		return "SSH"
	}
	return "SSH " + name
}

func shellTopBorder(width int, title string) string {
	if width < 4 {
		width = 4
	}
	innerWidth := width - 2
	title = " " + term.Truncate(title, innerWidth-2) + " "
	titleWidth := term.Width(title)
	if titleWidth >= innerWidth {
		return "+" + term.Truncate(title, innerWidth) + "+"
	}
	left := (innerWidth - titleWidth) / 2
	right := innerWidth - titleWidth - left
	return fmt.Sprintf("+%s%s%s+", strings.Repeat("-", left), title, strings.Repeat("-", right))
}

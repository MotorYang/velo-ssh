package tui

import (
	"fmt"

	"github.com/motoryang/velo-ssh/internal/term"
)

func (m Model) viewServerList() string {
	width := m.contentWidth()
	inner := width - 2
	body := []string{}
	if m.searching {
		body = append(body, "Search: "+m.searchInput.View())
	} else if m.filter != "" {
		body = append(body, "Filter: "+m.filter)
	} else {
		body = append(body, "Filter: -")
	}
	body = append(body, blankLine(inner))
	servers := m.filteredServers()
	if len(servers) == 0 {
		body = append(body, "No servers configured or matched.")
	} else {
		for i, srv := range servers {
			prefix := "  "
			if i == m.cursor {
				prefix = "> "
			}
			line := fmt.Sprintf("%s[%s] %s %s@%s:%d", prefix, srv.Env, srv.Name, srv.User, srv.Host, srv.Port)
			line = term.Truncate(line, inner)
			if i == m.cursor {
				line = m.styles.selected.Render(line)
			}
			body = append(body, line)
			if srv.Desc != "" {
				body = append(body, term.Truncate("    "+srv.Desc, inner))
			}
		}
	}
	return borderedBlock("VeloSSH Manager", width, body)
}

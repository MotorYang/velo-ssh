package tui

import (
	"fmt"
	"strings"
)

func (m Model) viewServerList() string {
	var b strings.Builder
	fmt.Fprintln(&b, m.styles.title.Render("VeloSSH Manager"))
	if m.searching {
		fmt.Fprintf(&b, "Search: %s\n", m.searchInput.View())
	} else if m.filter != "" {
		fmt.Fprintf(&b, "Filter: %s\n", m.filter)
	}
	fmt.Fprintln(&b)
	servers := m.filteredServers()
	if len(servers) == 0 {
		fmt.Fprintln(&b, "No servers configured or matched.")
	} else {
		for i, srv := range servers {
			prefix := "  "
			if i == m.cursor {
				prefix = "> "
			}
			line := fmt.Sprintf("%s[%s] %s %s@%s:%d", prefix, srv.Env, srv.Name, srv.User, srv.Host, srv.Port)
			if i == m.cursor {
				line = m.styles.selected.Render(line)
			}
			fmt.Fprintln(&b, line)
			if srv.Desc != "" {
				fmt.Fprintf(&b, "    %s\n", srv.Desc)
			}
		}
	}
	fmt.Fprintln(&b)
	return b.String()
}

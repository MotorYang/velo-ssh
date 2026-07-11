package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/motoryang/velo-ssh/internal/config"
	"github.com/motoryang/velo-ssh/internal/term"
	"github.com/motoryang/velo-ssh/internal/version"
)

func (m Model) viewServerList() string {
	width := m.contentWidth()
	inner := width - 2
	body := []string{}
	body = append(body, m.tr(textVersion)+": "+version.String())
	if m.searching {
		body = append(body, m.tr(textSearch)+": "+m.searchInput.View())
	} else if m.filter != "" {
		body = append(body, m.tr(textFilter)+": "+m.filter)
	} else {
		body = append(body, m.tr(textFilter)+": -")
	}
	body = append(body, blankLine(inner))
	servers := m.filteredServers()
	if len(servers) == 0 {
		body = append(body, m.tr(textNoServers))
	} else {
		groups, names := groupedServersByTag(servers)
		for groupIndex, name := range names {
			if groupIndex > 0 {
				body = append(body, blankLine(inner))
			}
			body = append(body, term.Truncate("["+name+"]", inner))
			for _, item := range groups[name] {
				srv := item.server
				prefix := "  "
				if item.index == m.cursor {
					prefix = "> "
				}
				line := fmt.Sprintf("%s%s [%s] [%s] %s@%s:%d", prefix, srv.Name, defaultServerEnv(srv.Env), m.serverHealthLabel(srv.ID), srv.User, srv.Host, srv.Port)
				line = term.Truncate(line, inner)
				if item.index == m.cursor {
					line = m.styles.selected.Render(line)
				}
				body = append(body, line)
				if srv.Desc != "" {
					body = append(body, term.Truncate("    "+srv.Desc, inner))
				}
			}
		}
	}
	return borderedBlock(m.tr(textManagerTitle), width, body)
}

func (m Model) serverHealthLabel(id string) string {
	health, ok := m.serverHealth[id]
	if !ok || !health.Checked {
		return "?"
	}
	if !health.Online {
		return "down"
	}
	if health.Latency <= 0 {
		return "up"
	}
	return fmt.Sprintf("up %dms", health.Latency.Milliseconds())
}

func defaultServerEnv(env string) string {
	env = strings.TrimSpace(env)
	if env == "" {
		return "default"
	}
	return env
}

type groupedServerItem struct {
	index  int
	server config.Server
}

func groupedServersByTag(servers []config.Server) (map[string][]groupedServerItem, []string) {
	groups := map[string][]groupedServerItem{}
	for i, srv := range servers {
		tags := serverTags(srv)
		for _, tag := range tags {
			groups[tag] = append(groups[tag], groupedServerItem{index: i, server: srv})
		}
	}
	names := make([]string, 0, len(groups))
	for name := range groups {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		if names[i] == "default" {
			return true
		}
		if names[j] == "default" {
			return false
		}
		return names[i] < names[j]
	})
	return groups, names
}

func visibleServerCursors(servers []config.Server) []int {
	groups, names := groupedServersByTag(servers)
	cursors := make([]int, 0, len(servers))
	for _, name := range names {
		for _, item := range groups[name] {
			cursors = append(cursors, item.index)
		}
	}
	return cursors
}

func previousVisibleServerCursor(servers []config.Server, cursor int) int {
	visible := visibleServerCursors(servers)
	if len(visible) == 0 {
		return 0
	}
	pos := visibleServerCursorPosition(visible, cursor)
	if pos <= 0 {
		return visible[0]
	}
	return visible[pos-1]
}

func nextVisibleServerCursor(servers []config.Server, cursor int) int {
	visible := visibleServerCursors(servers)
	if len(visible) == 0 {
		return 0
	}
	pos := visibleServerCursorPosition(visible, cursor)
	if pos >= len(visible)-1 {
		return visible[len(visible)-1]
	}
	return visible[pos+1]
}

func visibleServerCursorPosition(visible []int, cursor int) int {
	for i, value := range visible {
		if value == cursor {
			return i
		}
	}
	return 0
}

func serverTags(srv config.Server) []string {
	tags := []string{}
	seen := map[string]bool{}
	for _, tag := range srv.Tags {
		tag = strings.TrimSpace(tag)
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		tags = append(tags, tag)
	}
	if len(tags) > 0 {
		return tags
	}
	return []string{"default"}
}

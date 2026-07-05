package tui

import (
	"fmt"
	"strings"
)

func (m Model) viewServerForm() string {
	var b strings.Builder
	title := "Add Server"
	if m.form.mode == "edit" {
		title = "Edit Server"
	}
	fmt.Fprintln(&b, m.styles.title.Render(title))
	fmt.Fprintln(&b)
	for i, field := range m.form.fields {
		prefix := "  "
		if i == m.form.index {
			prefix = "> "
		}
		fmt.Fprintf(&b, "%s%-28s %s\n", prefix, serverFormLabels[i]+":", field.View())
	}
	return b.String()
}

package tui

import (
	"fmt"
	"strings"
)

func (m Model) viewServerForm() string {
	var b strings.Builder
	title := m.tr(textAddServerTitle)
	if m.form.mode == "edit" {
		title = m.tr(textEditServerTitle)
	}
	if m.form.mode == "clone" {
		title = m.tr(textCloneServerTitle)
	}
	fmt.Fprintln(&b, m.styles.title.Render(title))
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, m.tr(textServerFormGuide))
	fmt.Fprintln(&b)
	for _, i := range m.form.visibleFields() {
		field := m.form.fields[i]
		prefix := "  "
		if i == m.form.index {
			prefix = "> "
		}
		value := field.View()
		if i == serverFieldAuthType {
			value = authTypeSelector(m.form.authType())
		}
		fmt.Fprintf(&b, "%s%s %s\n", prefix, padVisual(serverFormLabel(i, m.config.Settings.Language)+":", 28), value)
	}
	return b.String()
}

func authTypeSelector(current string) string {
	options := []string{"agent", "key", "password"}
	var parts []string
	for _, option := range options {
		if option == current {
			parts = append(parts, "["+option+"]")
		} else {
			parts = append(parts, " "+option+" ")
		}
	}
	return strings.Join(parts, "  ")
}

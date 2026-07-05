package sshnet

import "strings"

type EscapeResult struct {
	Local   bool
	Command string
	Arg     string
	Help    bool
	Unknown bool
}

func ParseEscapeLine(line string) EscapeResult {
	if line != ":vssh" && !strings.HasPrefix(line, ":vssh ") {
		return EscapeResult{}
	}
	if line == ":vssh" {
		return EscapeResult{Local: true, Command: "help", Help: true}
	}
	rest := strings.TrimSpace(strings.TrimPrefix(line, ":vssh "))
	if rest == "" {
		return EscapeResult{Local: true, Command: "help", Help: true}
	}
	cmd, arg, _ := strings.Cut(rest, " ")
	arg = strings.TrimPrefix(arg, " ")
	switch cmd {
	case "files", "tasks", "settings", "back", "reconnect", "quit", "help":
		return EscapeResult{Local: true, Command: cmd, Arg: arg, Help: cmd == "help"}
	case "send":
		return EscapeResult{Local: true, Command: cmd, Arg: arg}
	default:
		return EscapeResult{Local: true, Command: cmd, Unknown: true, Help: true}
	}
}

func EscapeHelp() string {
	return `VeloSSH local commands:
  :vssh files       switch to file manager
  :vssh tasks       open task center
  :vssh settings    open settings
  :vssh back        return to server list
  :vssh reconnect   reconnect current SSH session
  :vssh quit        disconnect current session
  :vssh help        show this help
  :vssh send <text> send text to remote`
}

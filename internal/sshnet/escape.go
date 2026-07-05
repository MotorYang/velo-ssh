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
	return EscapeHelpWithLanguage("")
}

func EscapeHelpWithLanguage(language string) string {
	if language == "zh-CN" {
		return `VeloSSH 本地命令：
  :vssh files       切换到文件管理器
  :vssh tasks       打开任务中心
  :vssh settings    打开设置
  :vssh back        返回服务器列表
  :vssh reconnect   重连当前 SSH 会话
  :vssh quit        断开当前会话
  :vssh help        显示本帮助
  :vssh send <text> 强制发送文本到远端`
	}
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

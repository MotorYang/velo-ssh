package sshnet

import (
	"strings"
	"testing"
)

func TestParseEscapeLineIgnoresOrdinaryColon(t *testing.T) {
	tests := []string{":", ":wq", "echo :vssh files", " :vssh files", ":vsshx files"}
	for _, tt := range tests {
		if got := ParseEscapeLine(tt); got.Local {
			t.Fatalf("ParseEscapeLine(%q) marked ordinary input as local: %+v", tt, got)
		}
	}
}

func TestParseEscapeLineCommands(t *testing.T) {
	tests := map[string]string{
		":vssh files":     "files",
		":vssh tasks":     "tasks",
		":vssh settings":  "settings",
		":vssh back":      "back",
		":vssh reconnect": "reconnect",
		":vssh quit":      "quit",
		":vssh help":      "help",
	}
	for input, want := range tests {
		got := ParseEscapeLine(input)
		if !got.Local || got.Command != want {
			t.Fatalf("ParseEscapeLine(%q) = %+v, want local command %q", input, got, want)
		}
	}
}

func TestParseEscapeLineSend(t *testing.T) {
	got := ParseEscapeLine(":vssh send :vssh files")
	if !got.Local || got.Command != "send" || got.Arg != ":vssh files" {
		t.Fatalf("unexpected send parse: %+v", got)
	}
}

func TestParseEscapeLineUnknownShowsHelp(t *testing.T) {
	got := ParseEscapeLine(":vssh nope")
	if !got.Local || !got.Unknown || !got.Help {
		t.Fatalf("unexpected unknown parse: %+v", got)
	}
}

func TestEscapeHelpWithLanguageChinese(t *testing.T) {
	got := EscapeHelpWithLanguage("zh-CN")
	for _, want := range []string{"VeloSSH 本地命令", ":vssh files", "切换到文件管理器", "强制发送文本到远端"} {
		if !strings.Contains(got, want) {
			t.Fatalf("chinese help missing %q in %q", want, got)
		}
	}
	if strings.Contains(got, "switch to file manager") {
		t.Fatalf("chinese help should not contain English command description: %q", got)
	}
}

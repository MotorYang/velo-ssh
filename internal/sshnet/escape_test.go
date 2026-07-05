package sshnet

import "testing"

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

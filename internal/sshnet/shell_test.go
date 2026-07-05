package sshnet

import (
	"bytes"
	"strings"
	"testing"
)

func TestShellInputDoesNotSendLocalFilesCommand(t *testing.T) {
	var remote bytes.Buffer
	var errOut bytes.Buffer
	var got EscapeResult
	localExit, err := runShellInput(strings.NewReader(":vssh files\n"), &remote, &errOut, func(res EscapeResult) {
		got = res
	})
	if err != nil {
		t.Fatal(err)
	}
	if !localExit {
		t.Fatal("expected local files command to exit shell")
	}
	if remote.String() != "" {
		t.Fatalf("remote got %q, want empty", remote.String())
	}
	if got.Command != "files" {
		t.Fatalf("local command = %q, want files", got.Command)
	}
	if !strings.Contains(errOut.String(), ":vssh files") {
		t.Fatalf("captured command was not echoed locally: %q", errOut.String())
	}
}

func TestShellInputSendForcesRemoteText(t *testing.T) {
	var remote bytes.Buffer
	localExit, err := runShellInput(strings.NewReader(":vssh send :vssh files\n"), &remote, &bytes.Buffer{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if localExit {
		t.Fatal("send should not exit shell")
	}
	if got := remote.String(); got != ":vssh files\n" {
		t.Fatalf("remote got %q, want forced text", got)
	}
}

func TestShellInputPassesOrdinaryColonCommands(t *testing.T) {
	var remote bytes.Buffer
	localExit, err := runShellInput(strings.NewReader(":wq\n:vsshx files\n"), &remote, &bytes.Buffer{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if localExit {
		t.Fatal("ordinary colon commands should not exit shell")
	}
	want := ":wq\n:vsshx files\n"
	if got := remote.String(); got != want {
		t.Fatalf("remote got %q, want %q", got, want)
	}
}

func TestShellInputInterceptsVSSHWhenLocalLineStateDrifts(t *testing.T) {
	var remote bytes.Buffer
	var got EscapeResult
	localExit, err := runShellInput(strings.NewReader("stale-local-state:vssh files\n"), &remote, &bytes.Buffer{}, func(res EscapeResult) {
		got = res
	})
	if err != nil {
		t.Fatal(err)
	}
	if !localExit {
		t.Fatal("expected :vssh command to be intercepted even when local state drifted")
	}
	if got.Command != "files" {
		t.Fatalf("local command = %q, want files", got.Command)
	}
	if remote.String() != "stale-local-state" {
		t.Fatalf("remote got %q, want prefix before local command", remote.String())
	}
}

func TestShellInputPassesArrowEscapeSequences(t *testing.T) {
	var remote bytes.Buffer
	localExit, err := runShellInput(strings.NewReader("\x1b[A\x1b[B\x1b[C\x1b[D"), &remote, &bytes.Buffer{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if localExit {
		t.Fatal("arrow escape sequences should not exit shell")
	}
	want := "\x1b[A\x1b[B\x1b[C\x1b[D"
	if got := remote.String(); got != want {
		t.Fatalf("remote got %q, want %q", got, want)
	}
}

func TestShellInputLocalCommandAfterArrowSequence(t *testing.T) {
	var remote bytes.Buffer
	var got EscapeResult
	localExit, err := runShellInput(strings.NewReader("\x1b[A:vssh files\n"), &remote, &bytes.Buffer{}, func(res EscapeResult) {
		got = res
	})
	if err != nil {
		t.Fatal(err)
	}
	if !localExit {
		t.Fatal("expected local command after arrow sequence to exit shell")
	}
	if got.Command != "files" {
		t.Fatalf("local command = %q, want files", got.Command)
	}
	if remote.String() != "\x1b[A" {
		t.Fatalf("remote got %q, want only arrow sequence", remote.String())
	}
}

func TestShellInputLocalCommandAfterLineResetControl(t *testing.T) {
	var remote bytes.Buffer
	var got EscapeResult
	localExit, err := runShellInput(strings.NewReader("partial\x03:vssh files\n"), &remote, &bytes.Buffer{}, func(res EscapeResult) {
		got = res
	})
	if err != nil {
		t.Fatal(err)
	}
	if !localExit {
		t.Fatal("expected local command after ctrl-c to exit shell")
	}
	if got.Command != "files" {
		t.Fatalf("local command = %q, want files", got.Command)
	}
	if remote.String() != "partial\x03" {
		t.Fatalf("remote got %q, want only reset line prefix", remote.String())
	}
}

func TestShellInputLocalCommandWithBracketedPasteMarkers(t *testing.T) {
	var remote bytes.Buffer
	var got EscapeResult
	localExit, err := runShellInput(strings.NewReader("\x1b[200~:vssh files\x1b[201~\n"), &remote, &bytes.Buffer{}, func(res EscapeResult) {
		got = res
	})
	if err != nil {
		t.Fatal(err)
	}
	if !localExit {
		t.Fatal("expected pasted local command to exit shell")
	}
	if got.Command != "files" {
		t.Fatalf("local command = %q, want files", got.Command)
	}
	if remote.String() != "" {
		t.Fatalf("remote got %q, want empty", remote.String())
	}
}

func TestShellInputUnknownShowsHelpWithoutSendingRemote(t *testing.T) {
	var remote bytes.Buffer
	var errOut bytes.Buffer
	localExit, err := runShellInput(strings.NewReader(":vssh nope\n"), &remote, &errOut, nil)
	if err != nil {
		t.Fatal(err)
	}
	if localExit {
		t.Fatal("unknown local command should not exit shell")
	}
	if remote.String() != "" {
		t.Fatalf("remote got %q, want empty", remote.String())
	}
	if !strings.Contains(errOut.String(), "VeloSSH local commands") {
		t.Fatalf("missing help output: %q", errOut.String())
	}
	if hasBareLF(errOut.String()) || !strings.Contains(errOut.String(), "\r\n  :vssh") {
		t.Fatalf("help output should use CRLF line endings in raw mode: %q", errOut.String())
	}
}

func TestShellInputHelpUsesConfiguredLanguage(t *testing.T) {
	var remote bytes.Buffer
	var errOut bytes.Buffer
	localExit, err := runShellInputWithHelp(strings.NewReader(":vssh help\n"), &remote, &errOut, nil, EscapeHelpWithLanguage("zh-CN"))
	if err != nil {
		t.Fatal(err)
	}
	if localExit {
		t.Fatal("help should not exit shell")
	}
	if remote.String() != "" {
		t.Fatalf("remote got %q, want empty", remote.String())
	}
	if !strings.Contains(errOut.String(), "VeloSSH 本地命令") || !strings.Contains(errOut.String(), "切换到文件管理器") {
		t.Fatalf("missing chinese help output: %q", errOut.String())
	}
}

func TestShellInputExitLineQuitsAfterSendingRemote(t *testing.T) {
	tests := []string{"exit\n", "exit 0\n"}
	for _, input := range tests {
		var remote bytes.Buffer
		var got EscapeResult
		localExit, err := runShellInput(strings.NewReader(input), &remote, &bytes.Buffer{}, func(res EscapeResult) {
			got = res
		})
		if err != nil {
			t.Fatal(err)
		}
		if !localExit {
			t.Fatalf("input %q should exit shell", input)
		}
		if got.Command != "quit" {
			t.Fatalf("input %q command = %q, want quit", input, got.Command)
		}
		if remote.String() != strings.TrimSuffix(input, "\n")+"\n" && remote.String() != strings.TrimSuffix(input, "\n")+"\r" {
			t.Fatalf("remote got %q, want exit line sent", remote.String())
		}
	}
}

func TestShellInputNonExitLineDoesNotQuit(t *testing.T) {
	var remote bytes.Buffer
	localExit, err := runShellInput(strings.NewReader("echo exit\n"), &remote, &bytes.Buffer{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if localExit {
		t.Fatal("echo exit should not exit shell")
	}
	if remote.String() != "echo exit\n" {
		t.Fatalf("remote got %q", remote.String())
	}
}

func TestShellInputLocalCommandWithCRLFDoesNotLeakLF(t *testing.T) {
	var remote bytes.Buffer
	localExit, err := runShellInput(strings.NewReader(":vssh files\r\n"), &remote, &bytes.Buffer{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !localExit {
		t.Fatal("expected local command to exit")
	}
	if remote.String() != "" {
		t.Fatalf("remote got %q, want empty", remote.String())
	}
}

func TestShellInputCaptureSupportsBackspaceEcho(t *testing.T) {
	var remote bytes.Buffer
	var errOut bytes.Buffer
	localExit, err := runShellInput(strings.NewReader(":vsshh\x7f files\n"), &remote, &errOut, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !localExit {
		t.Fatal("expected corrected local command to exit")
	}
	if remote.String() != "" {
		t.Fatalf("remote got %q, want empty", remote.String())
	}
	if !strings.Contains(errOut.String(), "\b \b") {
		t.Fatalf("backspace was not echoed: %q", errOut.String())
	}
}

func TestShellDetachBuffersAndReattachFlushesOutput(t *testing.T) {
	var first bytes.Buffer
	var second bytes.Buffer
	shell := &Shell{}
	shell.attach(&first, &first)
	shell.writeOutput([]byte("attached"), false)
	shell.detach()
	shell.writeOutput([]byte("buffered"), false)
	if first.String() != "attached" {
		t.Fatalf("first output = %q", first.String())
	}
	shell.attach(&second, &second)
	if second.String() != "buffered" {
		t.Fatalf("reattached output = %q, want buffered", second.String())
	}
}

func hasBareLF(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' && (i == 0 || s[i-1] != '\r') {
			return true
		}
	}
	return false
}

package tui

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/motoryang/velo-ssh/internal/sshnet"
)

func TestActionErrorFormatsUserFacingContext(t *testing.T) {
	err := actionError("delete remote path", "/var/www/app", os.ErrPermission)
	got := err.Error()
	for _, want := range []string{"delete remote path failed", "target=/var/www/app", "permission denied", "Recovery:"} {
		if !strings.Contains(got, want) {
			t.Fatalf("error %q missing %q", got, want)
		}
	}
}

func TestClassifyErrorCases(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "permission", err: os.ErrPermission, want: "permission denied"},
		{name: "exists", err: os.ErrExist, want: "target already exists"},
		{name: "missing", err: os.ErrNotExist, want: "path does not exist"},
		{name: "disk", err: errors.New("write: no space left on device"), want: "disk is full"},
		{name: "auth", err: errors.New("ssh: handshake failed: unable to authenticate"), want: "ssh authentication failed"},
		{name: "keyring", err: errors.New("read password from keyring: secret not found"), want: "keyring is unavailable or missing the secret"},
		{name: "timeout", err: context.DeadlineExceeded, want: "connection timed out"},
		{name: "canceled", err: errors.New("transfer canceled"), want: "task was canceled"},
		{name: "stale", err: errors.New("stale connection: ssh keepalive failed"), want: "ssh connection is stale"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason, recovery := classifyError(tt.err)
			if reason != tt.want {
				t.Fatalf("reason = %q, want %q", reason, tt.want)
			}
			if recovery == "" {
				t.Fatal("recovery should not be empty")
			}
		})
	}
}

func TestDisplayErrorFormatsHostKeyMismatch(t *testing.T) {
	err := &sshnet.ChangedHostKeyError{
		Host:           "example.com:22",
		Fingerprint:    "SHA256:test",
		KnownHostsPath: "/tmp/known_hosts",
		Err:            errors.New("knownhosts mismatch"),
	}
	got := displayError(err)
	for _, want := range []string{"Action failed", "target=example.com:22", "ssh host key changed", "/tmp/known_hosts"} {
		if !strings.Contains(got, want) {
			t.Fatalf("display error %q missing %q", got, want)
		}
	}
}

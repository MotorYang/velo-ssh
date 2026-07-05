package tui

import (
	"errors"
	"os"
	"strings"
	"testing"
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

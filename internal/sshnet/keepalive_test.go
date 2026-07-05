package sshnet

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/motoryang/velo-ssh/internal/config"
)

func TestKeepAliveStopsAfterClose(t *testing.T) {
	client := NewClient(config.Settings{KeepAliveSeconds: 60}, nil)
	client.startKeepAlive(context.Background())
	client.mu.Lock()
	done := client.keepaliveDone
	client.mu.Unlock()
	if done == nil {
		t.Fatal("expected keepalive done channel")
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("keepalive goroutine did not stop after close")
	}
}

func TestOpenSFTPRejectsStaleConnection(t *testing.T) {
	client := NewClient(config.DefaultSettings(), nil)
	client.stale = true
	if _, err := client.OpenSFTP(context.Background()); err == nil || !strings.Contains(err.Error(), "stale connection") {
		t.Fatalf("OpenSFTP error = %v, want stale connection", err)
	}
}

func TestOpenShellRejectsStaleConnection(t *testing.T) {
	client := NewClient(config.DefaultSettings(), nil)
	client.stale = true
	if _, err := client.OpenShell(context.Background(), PtySize{}); err == nil || !strings.Contains(err.Error(), "stale connection") {
		t.Fatalf("OpenShell error = %v, want stale connection", err)
	}
}

package sshnet

import (
	"context"
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

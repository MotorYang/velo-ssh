package sshnet

import (
	"context"
	"time"
)

func (c *Client) startKeepAlive(ctx context.Context) {
	seconds := c.settings.KeepAliveSeconds
	if seconds <= 0 {
		seconds = 20
	}
	c.keepaliveCancel()
	kaCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	c.mu.Lock()
	c.cancelKeepalive = cancel
	c.keepaliveDone = done
	c.mu.Unlock()
	go func() {
		defer close(done)
		ticker := time.NewTicker(time.Duration(seconds) * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-kaCtx.Done():
				return
			case <-ticker.C:
				c.mu.Lock()
				client := c.ssh
				c.mu.Unlock()
				if client == nil {
					continue
				}
				_, _, err := client.SendRequest("keepalive@vssh", true, nil)
				if err != nil {
					c.mu.Lock()
					c.stale = true
					c.mu.Unlock()
				}
			}
		}
	}()
}

func (c *Client) keepaliveCancel() {
	c.mu.Lock()
	cancel := c.cancelKeepalive
	done := c.keepaliveDone
	c.cancelKeepalive = nil
	c.keepaliveDone = nil
	c.mu.Unlock()
	if cancel != nil {
		cancel()
		if done != nil {
			select {
			case <-done:
			case <-time.After(2 * time.Second):
			}
		}
	}
}

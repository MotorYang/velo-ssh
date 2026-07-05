package tui

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/motoryang/velo-ssh/internal/sshnet"
)

func actionError(action, target string, err error) error {
	if err == nil {
		return nil
	}
	reason, recovery := classifyError(err)
	return fmt.Errorf("%s failed: target=%s, reason=%s. Recovery: %s", action, target, reason, recovery)
}

func displayError(err error) string {
	if err == nil {
		return ""
	}
	reason, recovery := classifyError(err)
	return fmt.Sprintf("Action failed: target=%s, reason=%s. Recovery: %s", errorTarget(err), reason, recovery)
}

func classifyError(err error) (string, string) {
	text := strings.ToLower(err.Error())
	var changed *sshnet.ChangedHostKeyError
	if errors.As(err, &changed) {
		return "ssh host key changed for " + changed.Host, "do not connect until you verify the server fingerprint and fix " + changed.KnownHostsPath
	}
	var unknown *sshnet.UnknownHostKeyError
	if errors.As(err, &unknown) {
		return "ssh host key is not trusted for " + unknown.Host, "accept the fingerprint only if it matches the server you expect"
	}
	var netErr net.Error
	switch {
	case errors.Is(err, context.DeadlineExceeded) || errors.As(err, &netErr) && netErr.Timeout() || strings.Contains(text, "timeout") || strings.Contains(text, "i/o timeout"):
		return "connection timed out", "check the host, port, network, proxy or firewall, then retry"
	case strings.Contains(text, "keyring"):
		return "keyring is unavailable or missing the secret", "open settings/server edit and re-save the password or passphrase, or check system keychain access"
	case errors.Is(err, os.ErrPermission) || strings.Contains(text, "permission denied"):
		return "permission denied", "use a path you can access, change permissions, or retry with another user"
	case errors.Is(err, os.ErrExist) || strings.Contains(text, "file exists") || strings.Contains(text, "already exists"):
		return "target already exists", "choose another name or delete/overwrite the existing target"
	case errors.Is(err, os.ErrNotExist) || strings.Contains(text, "no such file") || strings.Contains(text, "not found"):
		return "path does not exist", "refresh the file list and verify the path still exists"
	case strings.Contains(text, "no space left") || strings.Contains(text, "disk full") || strings.Contains(text, "quota exceeded"):
		return "disk is full", "free space on the target filesystem and retry"
	case strings.Contains(text, "unable to authenticate") || strings.Contains(text, "no supported methods remain") || strings.Contains(text, "ssh: handshake failed") && strings.Contains(text, "auth"):
		return "ssh authentication failed", "verify username, auth type, password, private key, passphrase, or SSH agent configuration"
	case strings.Contains(text, "transfer canceled") || strings.Contains(text, "context canceled") || strings.Contains(text, "canceled"):
		return "task was canceled", "restart the transfer if you still need it"
	case strings.Contains(text, "stale connection") || strings.Contains(text, "keepalive failed"):
		return "ssh connection is stale", "use :vssh reconnect or reconnect from the server list"
	default:
		return err.Error(), "check the target path, permissions, connection state, then retry"
	}
}

func errorTarget(err error) string {
	var changed *sshnet.ChangedHostKeyError
	if errors.As(err, &changed) {
		return changed.Host
	}
	var unknown *sshnet.UnknownHostKeyError
	if errors.As(err, &unknown) {
		return unknown.Host
	}
	text := err.Error()
	for _, prefix := range []string{"connect ", "ssh handshake "} {
		if strings.HasPrefix(text, prefix) {
			rest := strings.TrimPrefix(text, prefix)
			if idx := strings.Index(rest, ":"); idx > 0 {
				return rest[:idx]
			}
		}
	}
	return "current operation"
}

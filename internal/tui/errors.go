package tui

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

func actionError(action, target string, err error) error {
	if err == nil {
		return nil
	}
	reason, recovery := classifyError(err)
	return fmt.Errorf("%s failed: target=%s, reason=%s. Recovery: %s", action, target, reason, recovery)
}

func classifyError(err error) (string, string) {
	text := strings.ToLower(err.Error())
	switch {
	case errors.Is(err, os.ErrPermission) || strings.Contains(text, "permission denied"):
		return "permission denied", "use a path you can access, change permissions, or retry with another user"
	case errors.Is(err, os.ErrExist) || strings.Contains(text, "file exists") || strings.Contains(text, "already exists"):
		return "target already exists", "choose another name or delete/overwrite the existing target"
	case errors.Is(err, os.ErrNotExist) || strings.Contains(text, "no such file") || strings.Contains(text, "not found"):
		return "path does not exist", "refresh the file list and verify the path still exists"
	case strings.Contains(text, "no space left") || strings.Contains(text, "disk full") || strings.Contains(text, "quota exceeded"):
		return "disk is full", "free space on the target filesystem and retry"
	default:
		return err.Error(), "check the target path, permissions, connection state, then retry"
	}
}

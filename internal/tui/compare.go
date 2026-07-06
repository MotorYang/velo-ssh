package tui

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/motoryang/velo-ssh/internal/config"
	"github.com/pkg/sftp"
)

const (
	maxTextDiffBytes = 256 * 1024
	maxTextDiffLines = 120
)

func (m Model) compareFilesCmd() tea.Cmd {
	return func() tea.Msg {
		if m.config.Settings.DefaultViewMode != config.ViewSplit {
			return compareResultMsg{err: fmt.Errorf("compare failed: show the local pane first with [b]")}
		}
		if m.ssh == nil {
			return compareResultMsg{err: fmt.Errorf("compare failed: ssh client is not connected")}
		}
		localItems := selectedFileItems(filteredFileItems(m.localFiles, m.localFileFilter), m.localCursor)
		remoteItems := selectedFileItems(filteredFileItems(m.remoteFiles, m.remoteFileFilter), m.remoteCursor)
		if len(localItems) != 1 || len(remoteItems) != 1 {
			return compareResultMsg{err: fmt.Errorf("compare failed: select exactly one local file and one remote file")}
		}
		if localItems[0].Name == ".." || remoteItems[0].Name == ".." {
			return compareResultMsg{err: fmt.Errorf("compare failed: parent directory entries cannot be compared")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		client, err := m.ssh.OpenSFTP(ctx)
		if err != nil {
			return compareResultMsg{err: err}
		}
		result, err := compareLocalRemoteFiles(client, localItems[0], remoteItems[0])
		return compareResultMsg{result: result, err: err}
	}
}

func compareLocalRemoteFiles(client *sftp.Client, local fileItem, remote fileItem) (string, error) {
	if local.IsDir || remote.IsDir {
		return "", fmt.Errorf("compare failed: select one local file and one remote file, not directories")
	}
	localHash, localSize, err := hashLocalFile(local.Path)
	if err != nil {
		return "", actionError("hash local file", local.Path, err)
	}
	remoteHash, remoteSize, err := hashRemoteFile(client, remote.Path)
	if err != nil {
		return "", actionError("hash remote file", remote.Path, err)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Local: %s\n", local.Path)
	fmt.Fprintf(&b, "Remote: %s\n\n", remote.Path)
	fmt.Fprintf(&b, "Local SHA-256:  %s\n", localHash)
	fmt.Fprintf(&b, "Remote SHA-256: %s\n", remoteHash)
	fmt.Fprintf(&b, "Local size:  %s\n", humanBytes(localSize))
	fmt.Fprintf(&b, "Remote size: %s\n\n", humanBytes(remoteSize))
	if localHash == remoteHash {
		b.WriteString("Result: files are identical.")
		return b.String(), nil
	}
	b.WriteString("Result: files differ.")
	diff, ok, err := textDiff(client, local.Path, remote.Path)
	if err != nil {
		fmt.Fprintf(&b, "\n\nText diff unavailable: %v", err)
		return b.String(), nil
	}
	if ok {
		fmt.Fprintf(&b, "\n\n%s", diff)
	}
	return b.String(), nil
}

func hashLocalFile(filePath string) (string, int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", 0, err
	}
	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), info.Size(), nil
}

func hashRemoteFile(client *sftp.Client, filePath string) (string, int64, error) {
	file, err := client.Open(filePath)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", 0, err
	}
	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), info.Size(), nil
}

func textDiff(client *sftp.Client, localPath, remotePath string) (string, bool, error) {
	localData, localText, err := readSmallLocalText(localPath)
	if err != nil || !localText {
		return "", false, err
	}
	remoteData, remoteText, err := readSmallRemoteText(client, remotePath)
	if err != nil || !remoteText {
		return "", false, err
	}
	diff := unifiedLineDiff(localPath, remotePath, splitLines(localData), splitLines(remoteData), maxTextDiffLines)
	return diff, true, nil
}

func readSmallLocalText(filePath string) ([]byte, bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, false, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxTextDiffBytes+1))
	if err != nil {
		return nil, false, err
	}
	return data, looksText(data) && len(data) <= maxTextDiffBytes, nil
}

func readSmallRemoteText(client *sftp.Client, filePath string) ([]byte, bool, error) {
	file, err := client.Open(filePath)
	if err != nil {
		return nil, false, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxTextDiffBytes+1))
	if err != nil {
		return nil, false, err
	}
	return data, looksText(data) && len(data) <= maxTextDiffBytes, nil
}

func looksText(data []byte) bool {
	if len(data) == 0 {
		return true
	}
	if !utf8.Valid(data) {
		return false
	}
	for _, b := range data {
		if b == 0 {
			return false
		}
	}
	return true
}

func splitLines(data []byte) []string {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	return strings.Split(text, "\n")
}

func unifiedLineDiff(localName, remoteName string, localLines, remoteLines []string, maxLines int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "--- local:%s\n", localName)
	fmt.Fprintf(&b, "+++ remote:%s\n", remoteName)
	b.WriteString("@@\n")
	i, j, emitted := 0, 0, 0
	for (i < len(localLines) || j < len(remoteLines)) && emitted < maxLines {
		if i < len(localLines) && j < len(remoteLines) && localLines[i] == remoteLines[j] {
			i++
			j++
			continue
		}
		if i < len(localLines) {
			fmt.Fprintf(&b, "- %s\n", localLines[i])
			i++
			emitted++
		}
		if emitted >= maxLines {
			break
		}
		if j < len(remoteLines) {
			fmt.Fprintf(&b, "+ %s\n", remoteLines[j])
			j++
			emitted++
		}
	}
	if i < len(localLines) || j < len(remoteLines) {
		b.WriteString("... diff truncated ...\n")
	}
	return b.String()
}

package tui

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/motoryang/velo-ssh/internal/app"
	"github.com/motoryang/velo-ssh/internal/config"
	"github.com/motoryang/velo-ssh/internal/transfer"
	"github.com/motoryang/velo-ssh/internal/updater"
	"github.com/pkg/sftp"
)

const (
	maxTextDiffBytes = 256 * 1024
	maxTextDiffLines = 120
)

type compareProgressMsg struct {
	progress updater.Progress
	result   string
	done     bool
	err      error
}

func (m Model) startCompareFromSelection(files []fileItem, cursor int) (tea.Model, tea.Cmd) {
	if m.config.Settings.DefaultViewMode != config.ViewSplit {
		m.err = "compare failed: show the local pane first with [b]"
		return m, nil
	}
	if m.ssh == nil {
		m.err = "compare failed: ssh client is not connected"
		return m, nil
	}
	localItems := selectedFileItems(filteredFileItems(m.localFiles, m.localFileFilter), m.localCursor)
	remoteItems := selectedFileItems(filteredFileItems(m.remoteFiles, m.remoteFileFilter), m.remoteCursor)
	if len(localItems) != 1 || len(remoteItems) != 1 {
		m.err = "compare failed: select exactly one local file and one remote file"
		return m, nil
	}
	if localItems[0].Name == ".." || remoteItems[0].Name == ".." || localItems[0].IsDir || remoteItems[0].IsDir {
		m.err = "compare failed: select one local file and one remote file, not directories"
		return m, nil
	}
	m.previous = app.StateFileManager
	m.modalKind = modalCompareProgress
	m.compareProgress = updater.Progress{Stage: "downloading"}
	m.compareCancel = make(chan struct{})
	m.compareCh = make(chan compareProgressMsg, 16)
	m.state = app.StateConfirmModal
	return m, startCompareCmd(m.ssh, localItems[0], remoteItems[0], m.compareCancel, m.compareCh)
}

func startCompareCmd(sshClient interface {
	OpenSFTP(context.Context) (*sftp.Client, error)
}, local fileItem, remote fileItem, canceled <-chan struct{}, ch chan compareProgressMsg) tea.Cmd {
	return func() tea.Msg {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			client, err := sshClient.OpenSFTP(ctx)
			if err != nil {
				sendCompareDone(ch, canceled, compareProgressMsg{done: true, err: err})
				return
			}
			result, err := compareLocalRemoteFilesWithDownload(client, local, remote, canceled, func(done, total int64) {
				sendCompareMsg(ch, compareProgressMsg{progress: updater.Progress{Stage: "downloading", Downloaded: done, Total: total}})
			})
			sendCompareDone(ch, canceled, compareProgressMsg{result: result, done: true, err: err})
		}()
		return <-ch
	}
}

func sendCompareMsg(ch chan compareProgressMsg, msg compareProgressMsg) {
	select {
	case ch <- msg:
	default:
	}
}

func sendCompareDone(ch chan compareProgressMsg, canceled <-chan struct{}, msg compareProgressMsg) {
	select {
	case <-canceled:
		return
	case ch <- msg:
	}
}

func waitCompareCmd(ch chan compareProgressMsg) tea.Cmd {
	return func() tea.Msg {
		if ch == nil {
			return nil
		}
		return <-ch
	}
}

func (m *Model) clearCompare() {
	m.compareResult = ""
	m.compareProgress = updater.Progress{}
	m.compareCh = nil
	m.compareCancel = nil
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

func compareLocalRemoteFilesWithDownload(client *sftp.Client, local fileItem, remote fileItem, canceled <-chan struct{}, progress func(done, total int64)) (string, error) {
	tmpDir, err := os.MkdirTemp("", "velossh-compare-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)
	tmpPath := filepath.Join(tmpDir, filepath.Base(remote.Path))
	if err := transfer.AtomicDownload(client, remote.Path, tmpPath, "compare", progress, nil, canceled, nil); err != nil {
		return "", err
	}
	return compareLocalFiles(local.Path, tmpPath, local.Path, remote.Path)
}

func compareLocalFiles(localPath, downloadedPath, localLabel, remoteLabel string) (string, error) {
	localHash, localSize, err := hashLocalFile(localPath)
	if err != nil {
		return "", actionError("hash local file", localPath, err)
	}
	remoteHash, remoteSize, err := hashLocalFile(downloadedPath)
	if err != nil {
		return "", actionError("hash downloaded remote file", downloadedPath, err)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Local: %s\n", localLabel)
	fmt.Fprintf(&b, "Remote: %s\n\n", remoteLabel)
	fmt.Fprintf(&b, "Local SHA-256:  %s\n", localHash)
	fmt.Fprintf(&b, "Remote SHA-256: %s\n", remoteHash)
	fmt.Fprintf(&b, "Local size:  %s\n", humanBytes(localSize))
	fmt.Fprintf(&b, "Remote size: %s\n\n", humanBytes(remoteSize))
	if localHash == remoteHash {
		b.WriteString("Result: files are identical.")
		return b.String(), nil
	}
	b.WriteString("Result: files differ.")
	diff, ok, err := textDiffLocal(localPath, downloadedPath, localLabel, remoteLabel)
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

func textDiffLocal(localPath, downloadedPath, localLabel, remoteLabel string) (string, bool, error) {
	localData, localText, err := readSmallLocalText(localPath)
	if err != nil || !localText {
		return "", false, err
	}
	remoteData, remoteText, err := readSmallLocalText(downloadedPath)
	if err != nil || !remoteText {
		return "", false, err
	}
	diff := unifiedLineDiff(localLabel, remoteLabel, splitLines(localData), splitLines(remoteData), maxTextDiffLines)
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

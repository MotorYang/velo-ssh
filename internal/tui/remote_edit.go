package tui

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/motoryang/velo-ssh/internal/config"
	"github.com/motoryang/velo-ssh/internal/transfer"
)

type remoteEditPreparedMsg struct {
	draft config.Draft
	err   error
}

type remoteEditFinishedMsg struct {
	draft config.Draft
	err   error
}

func (m Model) prepareRemoteEditCmd(item fileItem) tea.Cmd {
	return func() tea.Msg {
		if m.ssh == nil {
			return remoteEditPreparedMsg{err: fmt.Errorf("remote edit failed: ssh client is not connected")}
		}
		if item.IsDir || item.Name == ".." {
			return remoteEditPreparedMsg{err: fmt.Errorf("remote edit failed: select one remote file")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		client, err := m.ssh.OpenSFTP(ctx)
		if err != nil {
			return remoteEditPreparedMsg{err: err}
		}
		info, err := client.Stat(item.Path)
		if err != nil {
			return remoteEditPreparedMsg{err: actionError("stat remote edit file", item.Path, err)}
		}
		draftDir, err := m.remoteDraftDir()
		if err != nil {
			return remoteEditPreparedMsg{err: err}
		}
		if err := os.MkdirAll(draftDir, 0o700); err != nil {
			return remoteEditPreparedMsg{err: err}
		}
		draftID := newTaskID("draft")
		localPath := filepath.Join(draftDir, draftID+"-"+filepath.Base(item.Path))
		if err := transfer.AtomicDownload(client, item.Path, localPath, draftID, nil, nil, nil, nil); err != nil {
			return remoteEditPreparedMsg{err: actionError("download remote edit draft", item.Path, err)}
		}
		sum, err := hashFileSHA256(localPath)
		if err != nil {
			return remoteEditPreparedMsg{err: err}
		}
		return remoteEditPreparedMsg{draft: config.Draft{
			ID:              draftID,
			ServerID:        m.activeServer.ID,
			RemotePath:      item.Path,
			LocalPath:       localPath,
			BaseRemoteSize:  info.Size(),
			BaseRemoteMTime: info.ModTime(),
			LocalSHA256:     sum,
			Status:          config.DraftPending,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}}
	}
}

func (m Model) openRemoteDraftEditorCmd(draft config.Draft) tea.Cmd {
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "vi"
	}
	cmd := exec.Command(editor, draft.LocalPath)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return remoteEditFinishedMsg{draft: draft, err: err}
	})
}

func (m Model) syncRemoteEditDraftCmd(draft config.Draft) tea.Cmd {
	return func() tea.Msg {
		if m.ssh == nil {
			_ = m.store.UpsertDraft(markDraftFailed(draft))
			return filePanesLoadedMsg{err: fmt.Errorf("remote edit sync failed: ssh client is not connected; draft saved for retry")}
		}
		sum, err := hashFileSHA256(draft.LocalPath)
		if err == nil {
			draft.LocalSHA256 = sum
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		client, err := m.ssh.OpenSFTP(ctx)
		if err != nil {
			_ = m.store.UpsertDraft(markDraftFailed(draft))
			return filePanesLoadedMsg{err: fmt.Errorf("remote edit sync failed: %w; draft saved for retry", err)}
		}
		if err := transfer.AtomicUpload(client, draft.LocalPath, draft.RemotePath, draft.ID, nil, nil, nil, nil); err != nil {
			_ = m.store.UpsertDraft(markDraftFailed(draft))
			return filePanesLoadedMsg{err: fmt.Errorf("remote edit sync failed: %w; draft saved for retry", err)}
		}
		draft.Status = config.DraftResolved
		_ = m.store.UpsertDraft(draft)
		_ = os.Remove(draft.LocalPath)
		remote, err := listRemoteFiles(client, m.remoteDir)
		return filePanesLoadedMsg{local: m.localFiles, remote: remote, err: err}
	}
}

func (m Model) remoteDraftDir() (string, error) {
	dir, err := config.DefaultDir()
	if err != nil {
		return "", err
	}
	serverID := m.activeServer.ID
	if serverID == "" {
		serverID = "unknown"
	}
	return filepath.Join(dir, "drafts", serverID), nil
}

func markDraftFailed(draft config.Draft) config.Draft {
	draft.Status = config.DraftFailed
	draft.UpdatedAt = time.Now()
	return draft
}

func hashFileSHA256(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

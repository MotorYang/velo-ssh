package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/motoryang/velo-ssh/internal/config"
	"github.com/motoryang/velo-ssh/internal/sshnet"
	"github.com/motoryang/velo-ssh/internal/transfer"
	"github.com/pkg/sftp"
)

func (m Model) connectFileManagerCmd(srv config.Server) tea.Cmd {
	return func() tea.Msg {
		client := sshnet.NewClient(m.config.Settings, config.OSKeyring{})
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := client.Connect(ctx, srv); err != nil {
			var unknown *sshnet.UnknownHostKeyError
			if errors.As(err, &unknown) {
				return hostKeyPromptMsg{err: unknown, server: srv, action: hostKeyActionFileManager}
			}
			return errMsg{err}
		}
		return fileManagerConnectedMsg{client: client, server: srv}
	}
}

func (m Model) refreshFilePanesCmd() tea.Cmd {
	return func() tea.Msg {
		local, err := listLocalFiles(m.localDir)
		if err != nil {
			return filePanesLoadedMsg{err: err}
		}
		remote := m.remoteFiles
		if m.ssh != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			client, err := m.ssh.OpenSFTP(ctx)
			if err != nil {
				return filePanesLoadedMsg{err: fmt.Errorf("open sftp: %w", err)}
			}
			remote, err = listRemoteFiles(client, m.remoteDir)
			if err != nil {
				return filePanesLoadedMsg{err: err}
			}
		}
		return filePanesLoadedMsg{local: local, remote: remote}
	}
}

func (m Model) renameCurrentCmd(newName string) tea.Cmd {
	newName = strings.TrimSpace(newName)
	return func() tea.Msg {
		if newName == "" || strings.Contains(newName, "/") || strings.Contains(newName, string(os.PathSeparator)) {
			return errMsg{fmt.Errorf("rename target must be a plain file name")}
		}
		files := m.currentFiles()
		cursor := m.currentFileCursor()
		if len(files) == 0 || cursor >= len(files) {
			return errMsg{fmt.Errorf("no file selected")}
		}
		item := files[cursor]
		if item.Name == ".." {
			return errMsg{fmt.Errorf("cannot rename parent directory entry")}
		}
		if m.activePane == 0 {
			target := filepath.Join(filepath.Dir(item.Path), newName)
			if err := os.Rename(item.Path, target); err != nil {
				return errMsg{actionError("rename local path", item.Path+" -> "+target, err)}
			}
			local, err := listLocalFiles(m.localDir)
			return filePanesLoadedMsg{local: local, remote: m.remoteFiles, err: err}
		}
		if m.ssh == nil {
			return errMsg{fmt.Errorf("ssh client is not connected")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		client, err := m.ssh.OpenSFTP(ctx)
		if err != nil {
			return errMsg{err}
		}
		target := path.Join(path.Dir(item.Path), newName)
		if err := client.Rename(item.Path, target); err != nil {
			return errMsg{actionError("rename remote path", item.Path+" -> "+target, err)}
		}
		remote, err := listRemoteFiles(client, m.remoteDir)
		return filePanesLoadedMsg{local: m.localFiles, remote: remote, err: err}
	}
}

func (m Model) mkdirCurrentCmd(name string) tea.Cmd {
	name = strings.TrimSpace(name)
	return func() tea.Msg {
		if name == "" || strings.Contains(name, "/") || strings.Contains(name, string(os.PathSeparator)) {
			return filePanesLoadedMsg{err: fmt.Errorf("create directory failed: target name must be a plain directory name")}
		}
		if m.config.Settings.DefaultViewMode != config.ViewSingle && m.activePane == 0 {
			target := filepath.Join(m.localDir, name)
			if err := os.Mkdir(target, 0o755); err != nil {
				return filePanesLoadedMsg{err: actionError("create local directory", target, err)}
			}
			local, err := listLocalFiles(m.localDir)
			return filePanesLoadedMsg{local: local, remote: m.remoteFiles, err: err}
		}
		if m.ssh == nil {
			return filePanesLoadedMsg{err: fmt.Errorf("create remote directory failed: ssh client is not connected")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		client, err := m.ssh.OpenSFTP(ctx)
		if err != nil {
			return filePanesLoadedMsg{err: fmt.Errorf("create remote directory failed: open sftp: %w", err)}
		}
		target := path.Join(m.remoteDir, name)
		if err := client.Mkdir(target); err != nil {
			return filePanesLoadedMsg{err: actionError("create remote directory", target, err)}
		}
		remote, err := listRemoteFiles(client, m.remoteDir)
		return filePanesLoadedMsg{local: m.localFiles, remote: remote, err: err}
	}
}

func (m Model) deleteFilesCmd(items []fileItem, remote bool) tea.Cmd {
	return func() tea.Msg {
		if len(items) == 0 {
			return filePanesLoadedMsg{err: fmt.Errorf("delete failed: no file selected")}
		}
		if !remote {
			for _, item := range items {
				if item.Name == ".." {
					continue
				}
				if err := os.RemoveAll(item.Path); err != nil {
					return filePanesLoadedMsg{err: actionError("delete local path", item.Path, err)}
				}
			}
			local, err := listLocalFiles(m.localDir)
			return filePanesLoadedMsg{local: local, remote: m.remoteFiles, err: err}
		}
		if m.ssh == nil {
			return filePanesLoadedMsg{err: fmt.Errorf("delete remote path failed: ssh client is not connected")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		client, err := m.ssh.OpenSFTP(ctx)
		if err != nil {
			return filePanesLoadedMsg{err: fmt.Errorf("delete remote path failed: open sftp: %w", err)}
		}
		for _, item := range items {
			if item.Name == ".." {
				continue
			}
			if err := removeRemoteRecursive(client, item.Path); err != nil {
				return filePanesLoadedMsg{err: actionError("delete remote path", item.Path, err)}
			}
		}
		remoteFiles, err := listRemoteFiles(client, m.remoteDir)
		return filePanesLoadedMsg{local: m.localFiles, remote: remoteFiles, err: err}
	}
}

func (m Model) startUploadCmd(force bool, items []fileItem) tea.Cmd {
	return func() tea.Msg {
		if m.ssh == nil {
			return transferStartedMsg{err: fmt.Errorf("ssh client is not connected")}
		}
		if len(items) == 0 {
			items = selectedTransferItems(m.localFiles, m.localCursor)
		}
		if len(items) == 0 {
			return transferStartedMsg{err: fmt.Errorf("no local file selected for upload")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		client, err := m.ssh.OpenSFTP(ctx)
		if err != nil {
			return transferStartedMsg{err: err}
		}
		if m.config.Settings.ConfirmOverwrite && !force {
			var existing []string
			for _, item := range items {
				if item.IsDir {
					continue
				}
				target := path.Join(m.remoteDir, item.Name)
				if _, err := client.Stat(target); err == nil {
					existing = append(existing, target)
				} else if !os.IsNotExist(err) {
					return transferStartedMsg{err: actionError("check remote upload target", target, err)}
				}
			}
			if len(existing) > 0 {
				return overwritePromptMsg{direction: transfer.Upload, items: items, targets: existing}
			}
		}
		started := 0
		for _, item := range items {
			if item.IsDir {
				continue
			}
			task := transfer.NewTask(newTaskID("up"), transfer.Upload, item.Path, path.Join(m.remoteDir, item.Name))
			task.ServerID = m.activeServer.ID
			m.tasks.Start(context.Background(), client, task)
			started++
		}
		if started == 0 {
			return transferStartedMsg{err: fmt.Errorf("folder upload is not implemented in MVP")}
		}
		return transferStartedMsg{message: fmt.Sprintf("Started %d upload task(s).", started)}
	}
}

func (m Model) startDownloadCmd(force bool, items []fileItem) tea.Cmd {
	return func() tea.Msg {
		if m.ssh == nil {
			return transferStartedMsg{err: fmt.Errorf("ssh client is not connected")}
		}
		if len(items) == 0 {
			items = selectedTransferItems(m.remoteFiles, m.remoteCursor)
		}
		if len(items) == 0 {
			return transferStartedMsg{err: fmt.Errorf("no remote file selected for download")}
		}
		if m.config.Settings.ConfirmOverwrite && !force {
			var existing []string
			for _, item := range items {
				if item.IsDir {
					continue
				}
				target := filepath.Join(m.localDir, item.Name)
				if _, err := os.Stat(target); err == nil {
					existing = append(existing, target)
				} else if !os.IsNotExist(err) {
					return transferStartedMsg{err: actionError("check local download target", target, err)}
				}
			}
			if len(existing) > 0 {
				return overwritePromptMsg{direction: transfer.Download, items: items, targets: existing}
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		client, err := m.ssh.OpenSFTP(ctx)
		if err != nil {
			return transferStartedMsg{err: err}
		}
		started := 0
		for _, item := range items {
			if item.IsDir {
				continue
			}
			task := transfer.NewTask(newTaskID("down"), transfer.Download, item.Path, filepath.Join(m.localDir, item.Name))
			task.ServerID = m.activeServer.ID
			m.tasks.Start(context.Background(), client, task)
			started++
		}
		if started == 0 {
			return transferStartedMsg{err: fmt.Errorf("folder download is not implemented in MVP")}
		}
		return transferStartedMsg{message: fmt.Sprintf("Started %d download task(s).", started)}
	}
}

func listLocalFiles(dir string) ([]fileItem, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	items := []fileItem{{Name: "..", Path: filepath.Dir(dir), Mode: os.ModeDir | 0o755, IsDir: true}}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		items = append(items, fileItem{
			Name:    entry.Name(),
			Path:    filepath.Join(dir, entry.Name()),
			Mode:    info.Mode(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
			IsDir:   entry.IsDir(),
		})
	}
	sortFileItems(items)
	return items, nil
}

type remoteReadDir interface {
	ReadDir(string) ([]os.FileInfo, error)
}

func listRemoteFiles(client remoteReadDir, dir string) ([]fileItem, error) {
	if dir == "" {
		dir = "/"
	}
	entries, err := client.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read remote dir %s: %w", dir, err)
	}
	parent := path.Dir(dir)
	if parent == "." {
		parent = "/"
	}
	items := []fileItem{{Name: "..", Path: parent, Mode: os.ModeDir | 0o755, IsDir: true}}
	for _, entry := range entries {
		items = append(items, fileItem{
			Name:    entry.Name(),
			Path:    path.Join(dir, entry.Name()),
			Mode:    entry.Mode(),
			Size:    entry.Size(),
			ModTime: entry.ModTime(),
			IsDir:   entry.IsDir(),
		})
	}
	sortFileItems(items)
	return items, nil
}

func sortFileItems(items []fileItem) {
	if len(items) <= 1 {
		return
	}
	sort.Slice(items[1:], func(i, j int) bool {
		a := items[i+1]
		b := items[j+1]
		if a.IsDir != b.IsDir {
			return a.IsDir
		}
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})
}

func selectedTransferItems(items []fileItem, cursor int) []fileItem {
	return selectedFileItems(items, cursor)
}

func selectedFileItems(items []fileItem, cursor int) []fileItem {
	var selected []fileItem
	for _, item := range items {
		if item.Selected && item.Name != ".." {
			selected = append(selected, item)
		}
	}
	if len(selected) > 0 {
		return selected
	}
	if cursor >= 0 && cursor < len(items) && items[cursor].Name != ".." {
		return []fileItem{items[cursor]}
	}
	return nil
}

func removeRemoteRecursive(client *sftp.Client, remotePath string) error {
	info, err := client.Stat(remotePath)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return client.Remove(remotePath)
	}
	entries, err := client.ReadDir(remotePath)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := removeRemoteRecursive(client, path.Join(remotePath, entry.Name())); err != nil {
			return err
		}
	}
	return client.RemoveDirectory(remotePath)
}

func newTaskID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

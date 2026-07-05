package transfer

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/pkg/sftp"
)

const bufferSize = 4 * 1024 * 1024

func TempRemotePath(targetPath, taskID string) string {
	dir, base := path.Split(targetPath)
	return path.Join(dir, fmt.Sprintf(".%s.vssh.tmp.%s", base, taskID))
}

func TempLocalPath(targetPath, taskID string) string {
	dir := filepath.Dir(targetPath)
	base := filepath.Base(targetPath)
	return filepath.Join(dir, fmt.Sprintf(".%s.vssh.tmp.%s", base, taskID))
}

func AtomicUpload(client *sftp.Client, localPath, remotePath, taskID string, progress func(done, total int64), canceled <-chan struct{}, waitIfPaused func(<-chan struct{}) bool) error {
	local, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer local.Close()
	info, err := local.Stat()
	if err != nil {
		return err
	}
	total := info.Size()
	tmpPath := TempRemotePath(remotePath, taskID)
	var mode os.FileMode
	if old, err := client.Stat(remotePath); err == nil {
		mode = old.Mode()
	} else {
		mode = info.Mode()
	}
	remote, err := client.Create(tmpPath)
	if err != nil {
		return err
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = client.Remove(tmpPath)
		}
	}()
	buf := make([]byte, bufferSize)
	var done int64
	for {
		if waitIfPaused != nil && !waitIfPaused(canceled) {
			_ = remote.Close()
			return fmt.Errorf("transfer canceled")
		}
		select {
		case <-canceled:
			_ = remote.Close()
			return fmt.Errorf("transfer canceled")
		default:
		}
		n, readErr := local.Read(buf)
		if n > 0 {
			written, writeErr := remote.Write(buf[:n])
			if writeErr != nil {
				_ = remote.Close()
				return writeErr
			}
			done += int64(written)
			if progress != nil {
				progress(done, total)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			_ = remote.Close()
			return readErr
		}
	}
	if err := remote.Close(); err != nil {
		return err
	}
	if st, err := client.Stat(tmpPath); err != nil {
		return err
	} else if st.Size() != total {
		return fmt.Errorf("uploaded size mismatch: got %d want %d", st.Size(), total)
	}
	_ = client.Chmod(tmpPath, mode)
	if err := client.Rename(tmpPath, remotePath); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func AtomicDownload(client *sftp.Client, remotePath, localPath, taskID string, progress func(done, total int64), canceled <-chan struct{}, waitIfPaused func(<-chan struct{}) bool) error {
	remote, err := client.Open(remotePath)
	if err != nil {
		return err
	}
	defer remote.Close()
	info, err := remote.Stat()
	if err != nil {
		return err
	}
	total := info.Size()
	tmpPath := TempLocalPath(localPath, taskID)
	local, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()
	buf := make([]byte, bufferSize)
	var done int64
	for {
		if waitIfPaused != nil && !waitIfPaused(canceled) {
			_ = local.Close()
			return fmt.Errorf("transfer canceled")
		}
		select {
		case <-canceled:
			_ = local.Close()
			return fmt.Errorf("transfer canceled")
		default:
		}
		n, readErr := remote.Read(buf)
		if n > 0 {
			written, writeErr := local.Write(buf[:n])
			if writeErr != nil {
				_ = local.Close()
				return writeErr
			}
			done += int64(written)
			if progress != nil {
				progress(done, total)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			_ = local.Close()
			return readErr
		}
	}
	if err := local.Sync(); err != nil {
		_ = local.Close()
		return err
	}
	if err := local.Close(); err != nil {
		return err
	}
	if st, err := os.Stat(tmpPath); err != nil {
		return err
	} else if st.Size() != total {
		return fmt.Errorf("downloaded size mismatch: got %d want %d", st.Size(), total)
	}
	_ = os.Chmod(tmpPath, info.Mode())
	_ = os.Chtimes(tmpPath, time.Now(), info.ModTime())
	if err := os.Rename(tmpPath, localPath); err != nil {
		return err
	}
	cleanup = false
	return nil
}

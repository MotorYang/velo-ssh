package transfer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pkg/sftp"
)

const (
	MultipartThreshold = 64 * 1024 * 1024
	DefaultChunkSize   = 8 * 1024 * 1024
)

type chunkManifest struct {
	Version    int            `json:"version"`
	LocalPath  string         `json:"localPath"`
	RemotePath string         `json:"remotePath"`
	Size       int64          `json:"size"`
	ModTime    time.Time      `json:"modTime"`
	ChunkSize  int64          `json:"chunkSize"`
	TempPath   string         `json:"tempPath"`
	Chunks     map[int64]bool `json:"chunks"`
	UpdatedAt  time.Time      `json:"updatedAt"`
}

type uploadChunk struct {
	index  int64
	offset int64
	size   int64
}

func AtomicMultipartUpload(client *sftp.Client, localPath, remotePath, taskID string, concurrency int, progress func(done, total int64), tempPath func(string), canceled <-chan struct{}, waitIfPaused func(<-chan struct{}) bool) error {
	if concurrency <= 0 {
		concurrency = 1
	}
	localInfo, err := os.Stat(localPath)
	if err != nil {
		return err
	}
	if localInfo.IsDir() {
		return fmt.Errorf("multipart upload only supports regular files")
	}
	total := localInfo.Size()
	chunkSize := int64(DefaultChunkSize)
	manifestPath, err := multipartManifestPath(localPath, remotePath, total, localInfo.ModTime())
	if err != nil {
		return err
	}
	manifest, fresh, err := loadChunkManifest(manifestPath, localPath, remotePath, total, localInfo.ModTime(), chunkSize)
	if err != nil {
		return err
	}
	if manifest.TempPath == "" {
		manifest.TempPath = stableMultipartTempPath(remotePath, manifestPath)
		fresh = true
	}
	tmpPath := manifest.TempPath
	if tempPath != nil {
		tempPath(tmpPath)
	}
	var mode os.FileMode
	if old, err := client.Stat(remotePath); err == nil {
		mode = old.Mode()
	} else {
		mode = localInfo.Mode()
	}
	remote, err := client.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY)
	if err != nil {
		return err
	}
	if fresh || !remoteTempExists(client, tmpPath) {
		if err := remote.Truncate(total); err != nil {
			_ = remote.Close()
			return err
		}
		manifest.Chunks = map[int64]bool{}
		manifest.UpdatedAt = time.Now()
		if err := saveChunkManifest(manifestPath, manifest); err != nil {
			_ = remote.Close()
			return err
		}
	}
	_ = remote.Close()
	chunks := buildUploadChunks(total, chunkSize)
	var done int64
	for _, chunk := range chunks {
		if manifest.Chunks[chunk.index] {
			done += chunk.size
		}
	}
	if progress != nil {
		progress(done, total)
	}
	jobs := make(chan uploadChunk)
	errCh := make(chan error, 1)
	var mu sync.Mutex
	var wg sync.WaitGroup
	for worker := 0; worker < concurrency; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for chunk := range jobs {
				if manifest.Chunks[chunk.index] {
					continue
				}
				if waitIfPaused != nil && !waitIfPaused(canceled) {
					sendMultipartErr(errCh, fmt.Errorf("transfer canceled"))
					return
				}
				select {
				case <-canceled:
					sendMultipartErr(errCh, fmt.Errorf("transfer canceled"))
					return
				default:
				}
				if err := uploadChunkAt(client, localPath, tmpPath, chunk); err != nil {
					sendMultipartErr(errCh, err)
					return
				}
				mu.Lock()
				if !manifest.Chunks[chunk.index] {
					manifest.Chunks[chunk.index] = true
					done += chunk.size
					manifest.UpdatedAt = time.Now()
					err := saveChunkManifest(manifestPath, manifest)
					if progress != nil {
						progress(done, total)
					}
					mu.Unlock()
					if err != nil {
						sendMultipartErr(errCh, err)
						return
					}
					continue
				}
				mu.Unlock()
			}
		}()
	}
	for _, chunk := range chunks {
		select {
		case err := <-errCh:
			close(jobs)
			wg.Wait()
			return err
		case jobs <- chunk:
		}
	}
	close(jobs)
	wg.Wait()
	select {
	case err := <-errCh:
		return err
	default:
	}
	if st, err := client.Stat(tmpPath); err != nil {
		return err
	} else if st.Size() != total {
		return fmt.Errorf("multipart uploaded size mismatch: got %d want %d", st.Size(), total)
	}
	_ = client.Chmod(tmpPath, mode)
	if err := finalizeRemoteWrite(client, tmpPath, remotePath); err != nil {
		return err
	}
	_ = client.Remove(tmpPath)
	_ = os.Remove(manifestPath)
	return nil
}

func uploadChunkAt(client *sftp.Client, localPath, tmpPath string, chunk uploadChunk) error {
	local, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer local.Close()
	if _, err := local.Seek(chunk.offset, io.SeekStart); err != nil {
		return err
	}
	remote, err := client.OpenFile(tmpPath, os.O_WRONLY)
	if err != nil {
		return err
	}
	defer remote.Close()
	buf := make([]byte, chunk.size)
	n, err := io.ReadFull(local, buf)
	if err != nil && err != io.ErrUnexpectedEOF {
		return err
	}
	written, err := remote.WriteAt(buf[:n], chunk.offset)
	if err != nil {
		return err
	}
	if int64(written) != chunk.size {
		return fmt.Errorf("multipart chunk short write: got %d want %d", written, chunk.size)
	}
	return nil
}

func sendMultipartErr(ch chan error, err error) {
	select {
	case ch <- err:
	default:
	}
}

func buildUploadChunks(total, chunkSize int64) []uploadChunk {
	var chunks []uploadChunk
	for offset, index := int64(0), int64(0); offset < total; offset, index = offset+chunkSize, index+1 {
		size := chunkSize
		if remaining := total - offset; remaining < size {
			size = remaining
		}
		chunks = append(chunks, uploadChunk{index: index, offset: offset, size: size})
	}
	return chunks
}

func multipartManifestPath(localPath, remotePath string, size int64, modTime time.Time) (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	idInput := fmt.Sprintf("%s|%s|%d|%d", localPath, remotePath, size, modTime.UnixNano())
	sum := sha256.Sum256([]byte(idInput))
	dir := filepath.Join(cacheDir, "vssh", "multipart")
	return filepath.Join(dir, hex.EncodeToString(sum[:])+".json"), nil
}

func loadChunkManifest(path, localPath, remotePath string, size int64, modTime time.Time, chunkSize int64) (chunkManifest, bool, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		var manifest chunkManifest
		if err := json.Unmarshal(data, &manifest); err == nil &&
			manifest.LocalPath == localPath &&
			manifest.RemotePath == remotePath &&
			manifest.Size == size &&
			manifest.ModTime.Equal(modTime) &&
			manifest.ChunkSize == chunkSize &&
			manifest.Chunks != nil {
			return manifest, false, nil
		}
	}
	if err != nil && !os.IsNotExist(err) {
		return chunkManifest{}, false, err
	}
	return chunkManifest{
		Version:    1,
		LocalPath:  localPath,
		RemotePath: remotePath,
		Size:       size,
		ModTime:    modTime,
		ChunkSize:  chunkSize,
		TempPath:   stableMultipartTempPath(remotePath, path),
		Chunks:     map[int64]bool{},
		UpdatedAt:  time.Now(),
	}, true, nil
}

func stableMultipartTempPath(remotePath, manifestPath string) string {
	sum := sha256.Sum256([]byte(manifestPath))
	suffix := hex.EncodeToString(sum[:])[:16]
	return TempRemotePath(remotePath, "multipart-"+suffix)
}

func remoteTempExists(client *sftp.Client, tmpPath string) bool {
	_, err := client.Stat(tmpPath)
	return err == nil
}

func saveChunkManifest(path string, manifest chunkManifest) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp.")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

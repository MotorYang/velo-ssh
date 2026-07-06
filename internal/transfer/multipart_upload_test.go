package transfer

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildUploadChunks(t *testing.T) {
	chunks := buildUploadChunks(10, 4)
	if len(chunks) != 3 {
		t.Fatalf("chunks = %d, want 3", len(chunks))
	}
	want := []uploadChunk{
		{index: 0, offset: 0, size: 4},
		{index: 1, offset: 4, size: 4},
		{index: 2, offset: 8, size: 2},
	}
	for i := range want {
		if chunks[i] != want[i] {
			t.Fatalf("chunk[%d] = %#v, want %#v", i, chunks[i], want[i])
		}
	}
}

func TestChunkManifestRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.json")
	modTime := time.Unix(10, 20)
	manifest := chunkManifest{
		Version:    1,
		LocalPath:  "/local/file",
		RemotePath: "/remote/file",
		Size:       12,
		ModTime:    modTime,
		ChunkSize:  4,
		TempPath:   "/remote/.file.vssh.tmp.multipart-stable",
		Chunks:     map[int64]bool{1: true},
		UpdatedAt:  time.Now(),
	}
	if err := saveChunkManifest(path, manifest); err != nil {
		t.Fatal(err)
	}
	loaded, fresh, err := loadChunkManifest(path, manifest.LocalPath, manifest.RemotePath, manifest.Size, manifest.ModTime, manifest.ChunkSize)
	if err != nil {
		t.Fatal(err)
	}
	if fresh {
		t.Fatal("existing manifest should not be fresh")
	}
	if !loaded.Chunks[1] {
		t.Fatalf("loaded chunks = %#v", loaded.Chunks)
	}
	if loaded.TempPath != manifest.TempPath {
		t.Fatalf("temp path = %q, want %q", loaded.TempPath, manifest.TempPath)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("manifest mode = %v, want 0600", got)
	}
}

func TestNewChunkManifestUsesStableTempPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.json")
	modTime := time.Unix(10, 20)
	first, fresh, err := loadChunkManifest(path, "/local/file", "/remote/file", 12, modTime, 4)
	if err != nil {
		t.Fatal(err)
	}
	if !fresh {
		t.Fatal("new manifest should be fresh")
	}
	second, _, err := loadChunkManifest(path, "/local/file", "/remote/file", 12, modTime, 4)
	if err != nil {
		t.Fatal(err)
	}
	if first.TempPath != second.TempPath {
		t.Fatalf("temp path changed: %q != %q", first.TempPath, second.TempPath)
	}
}

func TestShouldUseMultipartUploadThreshold(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "large.bin")
	file, err := os.Create(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(MultipartThreshold); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if !shouldUseMultipartUpload(filePath) {
		t.Fatal("expected multipart upload for threshold-sized file")
	}
}

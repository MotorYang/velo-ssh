package tui

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/motoryang/velo-ssh/internal/ignore"
)

func TestCreateFolderArchiveRespectsIgnore(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "folder")
	if err := os.Mkdir(source, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(source, "dist"), 0o700); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		filepath.Join(source, "keep.txt"):       "keep",
		filepath.Join(source, "skip.log"):       "skip",
		filepath.Join(source, "dist", "app.js"): "skip",
	}
	for filePath, content := range files {
		if err := os.WriteFile(filePath, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	archivePath, err := createFolderArchive(source, root, ignore.New([]string{"*.log", "folder/dist/"}))
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(archivePath)
	names := readTarNames(t, archivePath)
	want := []string{"keep.txt"}
	if got := stringsJoin(names); got != stringsJoin(want) {
		t.Fatalf("archive names = %v, want %v", names, want)
	}
}

func readTarNames(t *testing.T, archivePath string) []string {
	t.Helper()
	file, err := os.Open(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	var names []string
	for {
		header, err := tr.Next()
		if err != nil {
			break
		}
		if header.Typeflag == tar.TypeReg {
			names = append(names, header.Name)
		}
	}
	sort.Strings(names)
	return names
}

func stringsJoin(values []string) string {
	out := ""
	for i, value := range values {
		if i > 0 {
			out += ","
		}
		out += value
	}
	return out
}

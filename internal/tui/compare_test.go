package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/motoryang/velo-ssh/internal/app"
	"github.com/motoryang/velo-ssh/internal/config"
)

func TestHashLocalFileUsesSHA256(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "data.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	hash, size, err := hashLocalFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if size != 5 {
		t.Fatalf("size = %d, want 5", size)
	}
	const want = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if hash != want {
		t.Fatalf("hash = %q, want %q", hash, want)
	}
}

func TestUnifiedLineDiffShowsTextChanges(t *testing.T) {
	got := unifiedLineDiff("local.txt", "remote.txt", []string{"a", "b", "c"}, []string{"a", "B", "c"}, 20)
	for _, want := range []string{"--- local:local.txt", "+++ remote:remote.txt", "- b", "+ B"} {
		if !strings.Contains(got, want) {
			t.Fatalf("diff missing %q in %q", want, got)
		}
	}
}

func TestLooksTextRejectsBinary(t *testing.T) {
	if looksText([]byte{'a', 0, 'b'}) {
		t.Fatal("binary data was treated as text")
	}
}

func TestCompareResultCloseReturnsFileManager(t *testing.T) {
	m := NewModel(app.StateConfirmModal, config.NewStore(t.TempDir()), config.DefaultFile())
	m.modalKind = modalCompareResult
	m.compareResult = "Result: files differ."
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updated.(Model)
	if m.state != app.StateFileManager {
		t.Fatalf("state = %s, want file manager", m.state)
	}
	if m.compareResult != "" {
		t.Fatalf("compare result was not cleared")
	}
}

func TestCompareProgressCancelReturnsFileManager(t *testing.T) {
	m := NewModel(app.StateConfirmModal, config.NewStore(t.TempDir()), config.DefaultFile())
	m.modalKind = modalCompareProgress
	m.compareCancel = make(chan struct{})
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.state != app.StateFileManager {
		t.Fatalf("state = %s, want file manager", m.state)
	}
}

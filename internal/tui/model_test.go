package tui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/motoryang/velo-ssh/internal/app"
	"github.com/motoryang/velo-ssh/internal/config"
	"github.com/motoryang/velo-ssh/internal/sshnet"
	"github.com/motoryang/velo-ssh/internal/transfer"
)

func TestSmallTerminalFallback(t *testing.T) {
	m := NewModel(app.StateServerList, config.NewStore(t.TempDir()), config.DefaultFile())
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 10})
	got := updated.(Model).View()
	if !strings.Contains(got, "terminal too small") || !strings.Contains(got, "80x24") {
		t.Fatalf("expected small terminal warning, got %q", got)
	}
}

func TestServerAddEditDeleteFlow(t *testing.T) {
	store := config.NewStore(t.TempDir())
	m := NewModel(app.StateServerList, store, config.DefaultFile())
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(Model)
	if m.state != app.StateServerForm {
		t.Fatalf("state after add = %s, want server form", m.state)
	}
	setServerFormValues(&m, map[int]string{
		serverFieldID:                "prod-web",
		serverFieldName:              "Prod Web",
		serverFieldEnv:               "prod",
		serverFieldHost:              "10.0.0.10",
		serverFieldPort:              "2222",
		serverFieldUser:              "root",
		serverFieldAuthType:          config.AuthAgent,
		serverFieldDefaultRemotePath: "/var/www",
		serverFieldTags:              "web,nginx",
	})
	updated, _ = m.saveServerForm()
	m = updated.(Model)
	if len(m.config.Servers) != 1 {
		t.Fatalf("servers after add = %d, want 1", len(m.config.Servers))
	}
	if got := m.config.Servers[0].Host; got != "10.0.0.10" {
		t.Fatalf("host = %q", got)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = updated.(Model)
	if m.state != app.StateServerForm {
		t.Fatalf("state after edit = %s, want server form", m.state)
	}
	m.form.fields[serverFieldHost].SetValue("10.0.0.11")
	updated, _ = m.saveServerForm()
	m = updated.(Model)
	if got := m.config.Servers[0].Host; got != "10.0.0.11" {
		t.Fatalf("edited host = %q", got)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = updated.(Model)
	if m.state != app.StateConfirmModal {
		t.Fatalf("state after delete = %s, want confirm modal", m.state)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if len(m.config.Servers) != 0 {
		t.Fatalf("servers after delete = %d, want 0", len(m.config.Servers))
	}
}

func TestServerFilter(t *testing.T) {
	cfg := config.DefaultFile()
	cfg.Servers = []config.Server{
		{ID: "prod-web", Name: "Prod Web", Env: "prod", Host: "10.0.0.1", Port: 22, User: "root", AuthType: config.AuthAgent},
		{ID: "dev-db", Name: "Dev DB", Env: "dev", Host: "10.0.0.2", Port: 22, User: "postgres", AuthType: config.AuthAgent},
	}
	m := NewModel(app.StateServerList, config.NewStore(t.TempDir()), cfg)
	m.filter = "postgres"
	got := m.filteredServers()
	if len(got) != 1 || got[0].ID != "dev-db" {
		t.Fatalf("filtered servers = %+v", got)
	}
}

func TestServerListUsesBorderedPanel(t *testing.T) {
	cfg := config.DefaultFile()
	cfg.Servers = []config.Server{{ID: "dev", Name: "Dev", Env: "dev", Host: "127.0.0.1", Port: 22, User: "root", AuthType: config.AuthAgent}}
	m := NewModel(app.StateServerList, config.NewStore(t.TempDir()), cfg)
	m.width = 90
	got := m.viewServerList()
	if !strings.HasPrefix(got, "+") || !strings.Contains(got, "VeloSSH Manager") || !strings.Contains(got, "|") {
		t.Fatalf("server list should render bordered panel: %q", got)
	}
}

func TestPasswordServerStoresSecretRef(t *testing.T) {
	store := config.NewStore(t.TempDir())
	m := NewModel(app.StateServerList, store, config.DefaultFile())
	secrets := fakeSecrets{values: map[string]string{}}
	m.secrets = secrets
	m.form = newServerForm("add", config.Server{Port: 22, AuthType: config.AuthPassword})
	setServerFormValues(&m, map[int]string{
		serverFieldID:             "prod-db",
		serverFieldName:           "Prod DB",
		serverFieldEnv:            "prod",
		serverFieldHost:           "10.0.0.20",
		serverFieldPort:           "22",
		serverFieldUser:           "root",
		serverFieldAuthType:       config.AuthPassword,
		serverFieldPasswordSecret: "secret-password",
	})
	updated, _ := m.saveServerForm()
	m = updated.(Model)
	if len(m.config.Servers) != 1 {
		t.Fatalf("servers = %d, want 1", len(m.config.Servers))
	}
	srv := m.config.Servers[0]
	if srv.PasswordRef != config.PasswordRef("prod-db") {
		t.Fatalf("passwordRef = %q", srv.PasswordRef)
	}
	if got := secrets.values[srv.PasswordRef]; got != "secret-password" {
		t.Fatalf("stored password = %q", got)
	}
}

func TestShellFinishedTransitions(t *testing.T) {
	tests := []struct {
		command string
		want    app.AppState
	}{
		{command: "files", want: app.StateFileManager},
		{command: "tasks", want: app.StateTaskCenter},
		{command: "settings", want: app.StateSettingsCenter},
		{command: "back", want: app.StateServerList},
		{command: "quit", want: app.StateServerList},
	}
	for _, tt := range tests {
		m := NewModel(app.StateShell, config.NewStore(t.TempDir()), config.DefaultFile())
		updated, _ := m.Update(shellFinishedMsg{action: sshnet.EscapeResult{Local: true, Command: tt.command}})
		got := updated.(Model).state
		if got != tt.want {
			t.Fatalf("command %s state = %s, want %s", tt.command, got, tt.want)
		}
	}
}

func TestHostKeyPromptAcceptWritesKnownHostsAndRetries(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".ssh", "known_hosts")
	m := NewModel(app.StateServerList, config.NewStore(t.TempDir()), config.DefaultFile())
	srv := config.Server{ID: "dev", Name: "Dev", Host: "example.com", Port: 22, User: "root", AuthType: config.AuthAgent}
	updated, _ := m.Update(hostKeyPromptMsg{
		err: &sshnet.UnknownHostKeyError{
			Host:           "example.com:22",
			Fingerprint:    "SHA256:test",
			KnownHostsLine: "example.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB",
			KnownHostsPath: path,
		},
		server: srv,
		action: hostKeyActionShell,
	})
	m = updated.(Model)
	if m.state != app.StateConfirmModal || m.modalKind != modalHostKey {
		t.Fatalf("state=%s modal=%s, want host key confirm", m.state, m.modalKind)
	}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("expected retry command after accepting host key")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("known_hosts was not written: %v", err)
	}
	if m.modalKind != "" || m.hostKeyErr != nil {
		t.Fatalf("host key prompt was not cleared: modal=%s err=%v", m.modalKind, m.hostKeyErr)
	}
}

func TestHostKeyPromptRejectCancels(t *testing.T) {
	m := NewModel(app.StateServerList, config.NewStore(t.TempDir()), config.DefaultFile())
	updated, _ := m.Update(hostKeyPromptMsg{
		err:    &sshnet.UnknownHostKeyError{Host: "example.com:22", Fingerprint: "SHA256:test"},
		server: config.Server{ID: "dev", Name: "Dev", Host: "example.com", Port: 22},
		action: hostKeyActionFileManager,
	})
	m = updated.(Model)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = updated.(Model)
	if cmd != nil {
		t.Fatal("rejecting host key should not retry")
	}
	if m.state != app.StateServerList {
		t.Fatalf("state = %s, want server list", m.state)
	}
	if !strings.Contains(m.status, "not trusted") {
		t.Fatalf("status = %q, want not trusted message", m.status)
	}
}

func TestChangedHostKeyDoesNotOpenConfirmModal(t *testing.T) {
	m := NewModel(app.StateServerList, config.NewStore(t.TempDir()), config.DefaultFile())
	changed := &sshnet.ChangedHostKeyError{Host: "example.com:22", Fingerprint: "SHA256:test", KnownHostsPath: "/tmp/known_hosts"}
	updated, _ := m.Update(errMsg{err: changed})
	m = updated.(Model)
	if m.state == app.StateConfirmModal {
		t.Fatal("changed host key must not open confirmation modal")
	}
	if !strings.Contains(m.err, "host key changed") {
		t.Fatalf("err = %q, want changed host key message", m.err)
	}
}

func TestShellViewUsesBorderedFrameWithServerName(t *testing.T) {
	m := NewModel(app.StateShell, config.NewStore(t.TempDir()), config.DefaultFile())
	m.width = 90
	m.activeServer = config.Server{Name: "云南服务器", ID: "yn"}
	got := m.viewShell()
	if !strings.Contains(got, "+---") || !strings.Contains(got, "SSH 云南服务器") || !strings.Contains(got, "|Remote shell is connected.") {
		t.Fatalf("shell view should render bordered server frame: %q", got)
	}
}

func TestShellTopBorderContainsTitle(t *testing.T) {
	got := shellTopBorder(40, "SSH prod")
	if !strings.HasPrefix(got, "+") || !strings.HasSuffix(got, "+") || !strings.Contains(got, "SSH prod") {
		t.Fatalf("top border = %q", got)
	}
}

func TestConfirmModalIsBorderedAndCentered(t *testing.T) {
	m := NewModel(app.StateConfirmModal, config.NewStore(t.TempDir()), config.DefaultFile())
	m.width = 120
	m.height = 32
	m.modalKind = modalOverwrite
	m.pendingOverwrite = []string{"/tmp/existing.txt"}
	got := m.viewConfirmModal()
	if !strings.Contains(got, "+---") || !strings.Contains(got, "|") || !strings.Contains(got, "Confirm") {
		t.Fatalf("confirm modal should render bordered panel: %q", got)
	}
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	firstPanelLine := ""
	for _, line := range lines {
		if strings.Contains(line, "+---") {
			firstPanelLine = line
			break
		}
	}
	if !strings.HasPrefix(firstPanelLine, " ") {
		t.Fatalf("confirm modal should be horizontally centered, first panel line=%q", firstPanelLine)
	}
}

func TestFileManagerSelection(t *testing.T) {
	cfg := config.DefaultFile()
	cfg.Settings.DefaultViewMode = config.ViewSplit
	m := NewModel(app.StateFileManager, config.NewStore(t.TempDir()), cfg)
	m.activePane = 0
	if len(m.localFiles) == 0 {
		t.Fatal("expected local files to include parent directory")
	}
	m.localCursor = 1
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = updated.(Model)
	if !m.localFiles[1].Selected {
		t.Fatal("expected current local file to be selected")
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = updated.(Model)
	if m.localFiles[1].Selected {
		t.Fatal("expected clear selection to deselect local file")
	}
}

func TestFileManagerSearchFiltersCurrentPane(t *testing.T) {
	cfg := config.DefaultFile()
	cfg.Settings.DefaultViewMode = config.ViewSplit
	m := NewModel(app.StateFileManager, config.NewStore(t.TempDir()), cfg)
	m.activePane = 0
	m.localFiles = []fileItem{
		{Name: "..", Path: "/tmp", IsDir: true},
		{Name: "alpha.log", Path: "/tmp/alpha.log"},
		{Name: "beta.txt", Path: "/tmp/beta.txt"},
	}
	m.remoteFiles = []fileItem{
		{Name: "..", Path: "/", IsDir: true},
		{Name: "remote-alpha", Path: "/remote-alpha"},
		{Name: "remote-beta", Path: "/remote-beta"},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = updated.(Model)
	if !m.fileSearching {
		t.Fatal("expected file search mode")
	}
	for _, r := range "beta" {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(Model)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	files := m.currentFiles()
	if len(files) != 2 || files[1].Name != "beta.txt" {
		t.Fatalf("filtered files = %+v, want parent plus beta.txt", files)
	}
	if m.remoteFileFilter != "" {
		t.Fatalf("remote filter = %q, want empty", m.remoteFileFilter)
	}
}

func TestFileManagerSearchSelectionOnlyAffectsVisibleItems(t *testing.T) {
	cfg := config.DefaultFile()
	cfg.Settings.DefaultViewMode = config.ViewSplit
	m := NewModel(app.StateFileManager, config.NewStore(t.TempDir()), cfg)
	m.activePane = 0
	m.localFiles = []fileItem{
		{Name: "..", Path: "/tmp", IsDir: true},
		{Name: "alpha.log", Path: "/tmp/alpha.log"},
		{Name: "beta.txt", Path: "/tmp/beta.txt"},
	}
	m.localFileFilter = "beta"
	m.localCursor = 1
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = updated.(Model)
	if m.localFiles[1].Selected {
		t.Fatal("hidden alpha.log should not be selected")
	}
	if !m.localFiles[2].Selected {
		t.Fatal("visible beta.txt should be selected")
	}
}

func TestFileManagerEscDoesNotLeavePanel(t *testing.T) {
	m := NewModel(app.StateFileManager, config.NewStore(t.TempDir()), config.DefaultFile())
	m.previous = app.StateServerList
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.state != app.StateFileManager {
		t.Fatalf("state after esc = %s, want file manager", m.state)
	}
	if strings.Contains(m.helpText(), "[Esc] Back") {
		t.Fatalf("file manager help should not include Esc Back: %q", m.helpText())
	}
}

func TestFileManagerQLeavesPanel(t *testing.T) {
	m := NewModel(app.StateFileManager, config.NewStore(t.TempDir()), config.DefaultFile())
	m.previous = app.StateServerList
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updated.(Model)
	if m.state != app.StateServerList {
		t.Fatalf("state after q = %s, want server list", m.state)
	}
}

func TestListLocalFilesSortsDirsFirst(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "z.txt"), []byte("z"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "adir"), 0o700); err != nil {
		t.Fatal(err)
	}
	items, err := listLocalFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) < 3 {
		t.Fatalf("items = %+v", items)
	}
	if items[0].Name != ".." || items[1].Name != "adir" || !items[1].IsDir {
		t.Fatalf("unexpected sort order: %+v", items)
	}
	if !strings.HasPrefix(formatMode(items[1]), "d") {
		t.Fatalf("directory mode = %q", formatMode(items[1]))
	}
}

func TestListRemoteFiles(t *testing.T) {
	client := fakeRemoteDir{entries: []os.FileInfo{
		fakeFileInfo{name: "b.txt", size: 2},
		fakeFileInfo{name: "adir", dir: true},
	}}
	items, err := listRemoteFiles(client, "/var/www")
	if err != nil {
		t.Fatal(err)
	}
	if items[0].Name != ".." || items[0].Path != "/var" {
		t.Fatalf("parent item = %+v", items[0])
	}
	if items[1].Name != "adir" || !items[1].IsDir {
		t.Fatalf("remote sort order = %+v", items)
	}
}

func TestSelectedTransferItemsFallsBackToCursor(t *testing.T) {
	items := []fileItem{
		{Name: "..", IsDir: true},
		{Name: "a.txt", Path: "/tmp/a.txt"},
	}
	got := selectedTransferItems(items, 1)
	if len(got) != 1 || got[0].Name != "a.txt" {
		t.Fatalf("selected fallback = %+v", got)
	}
}

func TestFileRowModeAndHumanSize(t *testing.T) {
	modTime := time.Date(2026, 7, 5, 12, 30, 0, 0, time.Local)
	file := fileItem{Name: "app.log", Mode: 0o644, Size: 1536, ModTime: modTime}
	if got := formatMode(file); got != "-rw-r--r--" {
		t.Fatalf("mode = %q", got)
	}
	if got := humanSize(file); got != "1.5KB" {
		t.Fatalf("size = %q", got)
	}
	if got := formatModTime(file); got != "2026-07-05 12:30" {
		t.Fatalf("mtime = %q", got)
	}
	dir := fileItem{Name: "logs", Mode: os.ModeDir | 0o755, IsDir: true}
	if got := formatMode(dir); got != "drwxr-xr-x" {
		t.Fatalf("dir mode = %q", got)
	}
	if got := humanSize(dir); got != "-" {
		t.Fatalf("dir size = %q", got)
	}
}

func TestFileRowModTimeToggle(t *testing.T) {
	modTime := time.Date(2026, 7, 5, 12, 30, 0, 0, time.Local)
	m := NewModel(app.StateFileManager, config.NewStore(t.TempDir()), config.DefaultFile())
	row := m.renderFileRow(0, fileItem{Name: "app.log", Mode: 0o644, Size: 1, ModTime: modTime}, true, 0, 80)
	if strings.Contains(row, "2026-07-05") {
		t.Fatalf("mod time should be hidden by default: %q", row)
	}
	m.showFileTime = true
	row = m.renderFileRow(0, fileItem{Name: "app.log", Mode: 0o644, Size: 1, ModTime: modTime}, true, 0, 80)
	if !strings.Contains(row, "2026-07-05") {
		t.Fatalf("mod time should be visible after toggle: %q", row)
	}
}

func TestFileRowOmitsTypeColumnAndFitsPane(t *testing.T) {
	m := NewModel(app.StateFileManager, config.NewStore(t.TempDir()), config.DefaultFile())
	row := m.renderFileRow(0, fileItem{
		Name:    "very-long-file-name-with-many-segments-and-中文.log",
		Mode:    0o644,
		Size:    4096,
		ModTime: time.Date(2026, 7, 5, 12, 30, 0, 0, time.Local),
	}, true, 0, 54)
	if strings.Contains(row, " file ") || strings.Contains(row, " dir ") {
		t.Fatalf("row should not render a type column: %q", row)
	}
	if got := visibleWidth(row); got != 54 {
		t.Fatalf("row width = %d, want 54: %q", got, row)
	}
	m.showFileTime = true
	row = m.renderFileRow(0, fileItem{
		Name:    "very-long-file-name-with-many-segments-and-中文.log",
		Mode:    0o644,
		Size:    4096,
		ModTime: time.Date(2026, 7, 5, 12, 30, 0, 0, time.Local),
	}, true, 0, 72)
	if got := visibleWidth(row); got != 72 {
		t.Fatalf("row with time width = %d, want 72: %q", got, row)
	}
}

func TestVisibleFileRangeTracksCursor(t *testing.T) {
	start, end := visibleFileRange(100, 50, 10)
	if start != 45 || end != 55 {
		t.Fatalf("range = %d,%d", start, end)
	}
	start, end = visibleFileRange(100, 2, 10)
	if start != 0 || end != 10 {
		t.Fatalf("range near top = %d,%d", start, end)
	}
	start, end = visibleFileRange(100, 99, 10)
	if start != 90 || end != 100 {
		t.Fatalf("range near bottom = %d,%d", start, end)
	}
}

func TestFileManagerViewDoesNotRenderAllRows(t *testing.T) {
	cfg := config.DefaultFile()
	cfg.Settings.DefaultViewMode = config.ViewSplit
	m := NewModel(app.StateFileManager, config.NewStore(t.TempDir()), cfg)
	m.height = 20
	m.localCursor = 50
	m.remoteCursor = 2
	m.localFiles = make([]fileItem, 100)
	m.remoteFiles = make([]fileItem, 100)
	for i := 0; i < 100; i++ {
		m.localFiles[i] = fileItem{Name: "local", Mode: 0o644, Size: int64(i)}
		m.remoteFiles[i] = fileItem{Name: "remote", Mode: 0o644, Size: int64(i)}
	}
	got := m.viewFileManager()
	if strings.Count(got, "\n") > 20 {
		t.Fatalf("file manager rendered too many lines: %d", strings.Count(got, "\n"))
	}
	if !strings.Contains(got, "LOCAL rows 46-55/100") || !strings.Contains(got, "REMOTE rows 1-10/100") {
		t.Fatalf("missing viewport marker: %q", got)
	}
	if !strings.Contains(got, "Sel") || !strings.Contains(got, "Mode") || !strings.Contains(got, "Name") {
		t.Fatalf("missing file table header: %q", got)
	}
}

func TestFileManagerPaneScrollsIndependently(t *testing.T) {
	cfg := config.DefaultFile()
	cfg.Settings.DefaultViewMode = config.ViewSplit
	m := NewModel(app.StateFileManager, config.NewStore(t.TempDir()), cfg)
	m.height = 20
	m.localCursor = 80
	m.remoteCursor = 3
	m.localFiles = make([]fileItem, 100)
	m.remoteFiles = make([]fileItem, 100)
	for i := 0; i < 100; i++ {
		m.localFiles[i] = fileItem{Name: "local", Mode: 0o644}
		m.remoteFiles[i] = fileItem{Name: "remote", Mode: 0o644}
	}
	got := m.viewFileManager()
	if !strings.Contains(got, "LOCAL rows 76-85/100") {
		t.Fatalf("local viewport did not follow local cursor: %q", got)
	}
	if !strings.Contains(got, "REMOTE rows 1-10/100") {
		t.Fatalf("remote viewport should not follow local cursor: %q", got)
	}
}

func TestFilePaneWidthRespondsToResize(t *testing.T) {
	m := NewModel(app.StateFileManager, config.NewStore(t.TempDir()), config.DefaultFile())
	m.width = 100
	narrow := m.filePaneWidth(false)
	m.width = 160
	wide := m.filePaneWidth(false)
	if wide <= narrow {
		t.Fatalf("width did not grow after resize: narrow=%d wide=%d", narrow, wide)
	}
}

func TestPadVisualHandlesStyledRows(t *testing.T) {
	m := NewModel(app.StateFileManager, config.NewStore(t.TempDir()), config.DefaultFile())
	width := m.filePaneWidth(false)
	row := m.renderFileRow(0, fileItem{Name: "..", Mode: os.ModeDir | 0o755, IsDir: true}, true, 0, width)
	padded := padVisual(row, width)
	if got := visibleWidth(padded); got != width {
		t.Fatalf("visible width = %d, want %d", got, width)
	}
}

func TestFileManagerToggleSinglePaneAndQuitToShell(t *testing.T) {
	cfg := config.DefaultFile()
	cfg.Settings.DefaultViewMode = config.ViewSplit
	m := NewModel(app.StateFileManager, config.NewStore(t.TempDir()), cfg)
	m.ssh = &sshnet.Client{}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	m = updated.(Model)
	if m.config.Settings.DefaultViewMode != config.ViewSingle || m.activePane != 1 {
		t.Fatalf("view=%s activePane=%d", m.config.Settings.DefaultViewMode, m.activePane)
	}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updated.(Model)
	if m.state != app.StateShell {
		t.Fatalf("state = %s, want shell", m.state)
	}
	if cmd == nil {
		t.Fatal("expected q from file manager to resume interactive shell")
	}
}

func TestFooterIsLastLine(t *testing.T) {
	m := NewModel(app.StateServerList, config.NewStore(t.TempDir()), config.DefaultFile())
	m.height = 30
	m.width = 100
	got := strings.TrimRight(m.View(), "\n")
	lines := strings.Split(got, "\n")
	if !strings.Contains(lines[len(lines)-2], "[j/k] Move") {
		t.Fatalf("first footer line = %q", lines[len(lines)-2])
	}
	if !strings.Contains(lines[len(lines)-1], "[q] Quit") {
		t.Fatalf("last line = %q", lines[len(lines)-1])
	}
	if lines[len(lines)-3] != strings.Repeat("-", 100) {
		t.Fatalf("footer border = %q", lines[len(lines)-3])
	}
}

func TestFooterSplitsLongHelpIntoMultipleLines(t *testing.T) {
	long := "[a] Alpha | [b] Beta | [c] Gamma | [d] Delta | [e] Epsilon | [f] Phi | [g] Gamma | [h] Eta | [i] Iota | [j] Jota | [k] Kappa"
	lines := splitHelpLines(long)
	if len(lines) < 2 {
		t.Fatalf("expected long help to split, got %v", lines)
	}
	for _, line := range lines {
		if visibleWidth(line) > 110 {
			t.Fatalf("footer line too wide: %d %q", visibleWidth(line), line)
		}
	}
}

func TestTaskCenterUsesBorderedPanel(t *testing.T) {
	m := NewModel(app.StateTaskCenter, config.NewStore(t.TempDir()), config.DefaultFile())
	m.width = 100
	m.tasks.Add(transfer.NewTask("task-a", transfer.Upload, "/local/a", "/remote/a"))
	got := m.viewTaskCenter()
	if !strings.HasPrefix(got, "+") || !strings.Contains(got, "Task Center") || !strings.Contains(got, "upload") {
		t.Fatalf("task center should render bordered panel: %q", got)
	}
}

func TestFileManagerMKeyTogglesTime(t *testing.T) {
	m := NewModel(app.StateFileManager, config.NewStore(t.TempDir()), config.DefaultFile())
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m = updated.(Model)
	if !m.showFileTime {
		t.Fatal("expected m to toggle file time display")
	}
}

func TestTaskCenterMoveAndCancelWithConfirm(t *testing.T) {
	m := NewModel(app.StateTaskCenter, config.NewStore(t.TempDir()), config.DefaultFile())
	first := transfer.NewTask("task-a", transfer.Upload, "/local/a", "/remote/a")
	second := transfer.NewTask("task-b", transfer.Download, "/remote/b", "/local/b")
	second.StartedAt = first.StartedAt.Add(time.Second)
	m.tasks.Add(first)
	m.tasks.Add(second)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)
	if m.taskCursor != 1 {
		t.Fatalf("taskCursor = %d, want 1", m.taskCursor)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = updated.(Model)
	if m.state != app.StateConfirmModal || m.modalKind != modalTaskCancel {
		t.Fatalf("state=%s modal=%s, want task cancel confirm", m.state, m.modalKind)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	tasks := m.taskSnapshots()
	if len(tasks) != 1 || tasks[0].ID != "task-a" {
		t.Fatalf("tasks after cancel = %+v, want only task-a", tasks)
	}
	if !strings.Contains(m.viewTaskCenter(), ">") {
		t.Fatalf("task center should render selected row: %q", m.viewTaskCenter())
	}
}

func TestTaskCenterCancelPreservesBackTarget(t *testing.T) {
	m := NewModel(app.StateTaskCenter, config.NewStore(t.TempDir()), config.DefaultFile())
	m.previous = app.StateFileManager
	m.tasks.Add(transfer.NewTask("task-a", transfer.Upload, "/local/a", "/remote/a"))

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = updated.(Model)
	if m.state != app.StateConfirmModal {
		t.Fatalf("state after x = %s, want confirm modal", m.state)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.state != app.StateTaskCenter {
		t.Fatalf("state after confirm = %s, want task center", m.state)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updated.(Model)
	if m.state != app.StateFileManager {
		t.Fatalf("state after q = %s, want file manager", m.state)
	}
}

func TestTaskCenterPauseAndResume(t *testing.T) {
	m := NewModel(app.StateTaskCenter, config.NewStore(t.TempDir()), config.DefaultFile())
	task := transfer.NewTask("task-a", transfer.Upload, "/local/a", "/remote/a")
	m.tasks.Add(task)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	m = updated.(Model)
	tasks := m.taskSnapshots()
	if tasks[0].Status != transfer.TaskPaused {
		t.Fatalf("task status after pause = %s, want paused", tasks[0].Status)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = updated.(Model)
	tasks = m.taskSnapshots()
	if tasks[0].Status != transfer.TaskQueued {
		t.Fatalf("task status after resume = %s, want queued", tasks[0].Status)
	}
}

func TestShutdownCancelsActiveTransferTasks(t *testing.T) {
	m := NewModel(app.StateServerList, config.NewStore(t.TempDir()), config.DefaultFile())
	task := transfer.NewTask("task-a", transfer.Upload, "/local/a", "/remote/a")
	m.tasks.Add(task)
	m.shutdown()
	if got := task.Snapshot().Status; got != transfer.TaskCanceled {
		t.Fatalf("task status = %s, want canceled", got)
	}
}

func TestOverwritePromptConfirmStartsTransferCommand(t *testing.T) {
	m := NewModel(app.StateConfirmModal, config.NewStore(t.TempDir()), config.DefaultFile())
	m.previous = app.StateFileManager
	m.modalKind = modalOverwrite
	m.pendingTransferDir = transfer.Download
	m.pendingTransferItems = []fileItem{{Name: "remote.txt", Path: "/tmp/remote.txt"}}
	m.pendingOverwrite = []string{filepath.Join(m.localDir, "remote.txt")}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.state != app.StateFileManager {
		t.Fatalf("state = %s, want file manager", m.state)
	}
	if cmd == nil {
		t.Fatal("expected confirmed overwrite to continue transfer command")
	}
	if m.pendingTransferItems != nil || m.pendingOverwrite != nil {
		t.Fatalf("pending overwrite state was not cleared")
	}
}

func TestFileManagerOpensTaskCenterWithRefreshTick(t *testing.T) {
	m := NewModel(app.StateFileManager, config.NewStore(t.TempDir()), config.DefaultFile())
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	m = updated.(Model)
	if m.state != app.StateTaskCenter {
		t.Fatalf("state = %s, want task center", m.state)
	}
	if cmd == nil {
		t.Fatal("expected task center refresh tick command")
	}
}

func TestFileManagerEnterLocalDirectory(t *testing.T) {
	dir := t.TempDir()
	child := filepath.Join(dir, "child")
	if err := os.Mkdir(child, 0o700); err != nil {
		t.Fatal(err)
	}
	cfg := config.DefaultFile()
	cfg.Settings.DefaultViewMode = config.ViewSplit
	m := NewModel(app.StateFileManager, config.NewStore(t.TempDir()), cfg)
	m.activePane = 0
	m.localDir = dir
	local, err := listLocalFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	m.localFiles = local
	for i, item := range m.localFiles {
		if item.Name == "child" {
			m.localCursor = i
			break
		}
	}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.localDir != child {
		t.Fatalf("localDir = %q, want %q", m.localDir, child)
	}
	if cmd == nil {
		t.Fatal("expected refresh command")
	}
}

func TestFileManagerCreateLocalDirectory(t *testing.T) {
	dir := t.TempDir()
	cfg := config.DefaultFile()
	cfg.Settings.DefaultViewMode = config.ViewSplit
	m := NewModel(app.StateFileManager, config.NewStore(t.TempDir()), cfg)
	m.activePane = 0
	m.localDir = dir
	local, err := listLocalFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	m.localFiles = local

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = updated.(Model)
	if !m.creatingDir {
		t.Fatal("expected create directory input mode")
	}
	m.mkdirInput.SetValue("new-dir")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("expected mkdir command")
	}
	msg := cmd()
	updated, _ = m.Update(msg)
	m = updated.(Model)
	if _, err := os.Stat(filepath.Join(dir, "new-dir")); err != nil {
		t.Fatal(err)
	}
	if len(m.localFiles) == 0 {
		t.Fatal("expected local files to refresh")
	}
}

func TestFileManagerDeleteLocalDirectoryWithConfirm(t *testing.T) {
	dir := t.TempDir()
	child := filepath.Join(dir, "delete-me")
	if err := os.Mkdir(child, 0o700); err != nil {
		t.Fatal(err)
	}
	cfg := config.DefaultFile()
	cfg.Settings.DefaultViewMode = config.ViewSplit
	m := NewModel(app.StateFileManager, config.NewStore(t.TempDir()), cfg)
	m.activePane = 0
	m.localDir = dir
	local, err := listLocalFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	m.localFiles = local
	for i, item := range m.localFiles {
		if item.Name == "delete-me" {
			m.localCursor = i
			break
		}
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = updated.(Model)
	if m.state != app.StateConfirmModal || m.modalKind != modalFileDelete {
		t.Fatalf("state=%s modal=%s, want file delete confirm", m.state, m.modalKind)
	}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("expected delete command")
	}
	msg := cmd()
	updated, _ = m.Update(msg)
	m = updated.(Model)
	if _, err := os.Stat(child); !os.IsNotExist(err) {
		t.Fatalf("deleted path still exists or unexpected stat error: %v", err)
	}
}

func TestFileManagerCopyPasteLocalDirectory(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	if err := os.Mkdir(source, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "file.txt"), []byte("content"), 0o600); err != nil {
		t.Fatal(err)
	}
	targetDir := filepath.Join(dir, "target")
	if err := os.Mkdir(targetDir, 0o700); err != nil {
		t.Fatal(err)
	}
	cfg := config.DefaultFile()
	cfg.Settings.DefaultViewMode = config.ViewSplit
	m := NewModel(app.StateFileManager, config.NewStore(t.TempDir()), cfg)
	m.activePane = 0
	m.localDir = dir
	local, err := listLocalFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	m.localFiles = local
	for i, item := range m.localFiles {
		if item.Name == "source" {
			m.localCursor = i
			break
		}
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = updated.(Model)
	m.localDir = targetDir
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("expected paste command")
	}
	msg := cmd()
	updated, _ = m.Update(msg)
	m = updated.(Model)
	if got, err := os.ReadFile(filepath.Join(targetDir, "source", "file.txt")); err != nil || string(got) != "content" {
		t.Fatalf("copied file content=%q err=%v", got, err)
	}
}

func TestFileManagerMoveLocalFile(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "move.txt")
	if err := os.WriteFile(source, []byte("content"), 0o600); err != nil {
		t.Fatal(err)
	}
	targetDir := filepath.Join(dir, "target")
	if err := os.Mkdir(targetDir, 0o700); err != nil {
		t.Fatal(err)
	}
	cfg := config.DefaultFile()
	cfg.Settings.DefaultViewMode = config.ViewSplit
	m := NewModel(app.StateFileManager, config.NewStore(t.TempDir()), cfg)
	m.activePane = 0
	m.localDir = dir
	local, err := listLocalFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	m.localFiles = local
	for i, item := range m.localFiles {
		if item.Name == "move.txt" {
			m.localCursor = i
			break
		}
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = updated.(Model)
	m.localDir = targetDir
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'M'}})
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("expected move command")
	}
	msg := cmd()
	updated, _ = m.Update(msg)
	m = updated.(Model)
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatalf("source still exists or unexpected stat error: %v", err)
	}
	if got, err := os.ReadFile(filepath.Join(targetDir, "move.txt")); err != nil || string(got) != "content" {
		t.Fatalf("moved file content=%q err=%v", got, err)
	}
}

func TestFileManagerRejectsCrossPanePaste(t *testing.T) {
	cfg := config.DefaultFile()
	cfg.Settings.DefaultViewMode = config.ViewSplit
	m := NewModel(app.StateFileManager, config.NewStore(t.TempDir()), cfg)
	m.clipboardFiles = []fileItem{{Name: "local.txt", Path: "/tmp/local.txt"}}
	m.clipboardRemote = false
	m.activePane = 1
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("expected paste command")
	}
	msg := cmd()
	updated, _ = m.Update(msg)
	m = updated.(Model)
	if !strings.Contains(m.err, "cross-pane copy") {
		t.Fatalf("err = %q, want cross-pane rejection", m.err)
	}
}

func TestCollectLocalUploadPlansForNestedFolder(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "nested")
	if err := os.Mkdir(nested, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "root.txt"), []byte("root"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "child.txt"), []byte("child"), 0o600); err != nil {
		t.Fatal(err)
	}
	var madeDirs []string
	plans, err := collectLocalUploadPlans(dir, "/remote/base", func(remotePath string, _ os.FileMode) error {
		madeDirs = append(madeDirs, remotePath)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	gotTargets := []string{plans[0].target, plans[1].target}
	sort.Strings(gotTargets)
	wantTargets := []string{"/remote/base/nested/child.txt", "/remote/base/root.txt"}
	if strings.Join(gotTargets, ",") != strings.Join(wantTargets, ",") {
		t.Fatalf("targets = %v, want %v", gotTargets, wantTargets)
	}
	if len(madeDirs) != 1 || madeDirs[0] != "/remote/base/nested" {
		t.Fatalf("made dirs = %v, want nested dir", madeDirs)
	}
}

func TestSettingsSave(t *testing.T) {
	store := config.NewStore(t.TempDir())
	m := NewModel(app.StateSettingsCenter, store, config.DefaultFile())
	m.settingsForm.fields[settingsFieldDefaultViewMode].SetValue(config.ViewSplit)
	m.settingsForm.blurCurrent()
	m.settingsForm.index = m.settingsForm.okIndex()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.state != app.StateServerList {
		t.Fatalf("state = %s, want server list", m.state)
	}
	saved, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if saved.Settings.DefaultViewMode != config.ViewSplit {
		t.Fatalf("saved view mode = %q, want split", saved.Settings.DefaultViewMode)
	}
}

func TestSettingsFormEditsAllFields(t *testing.T) {
	store := config.NewStore(t.TempDir())
	m := NewModel(app.StateSettingsCenter, store, config.DefaultFile())
	m.settingsForm.fields[settingsFieldDefaultViewMode].SetValue(config.ViewSplit)
	m.settingsForm.fields[settingsFieldASCIIFallback].SetValue(config.ASCIIFallbackDisabled)
	m.settingsForm.fields[settingsFieldFallbackRemotePath].SetValue("/var/tmp")
	m.settingsForm.fields[settingsFieldDraftTTLDays].SetValue("14")
	m.settingsForm.fields[settingsFieldTransferConcurrency].SetValue("8")
	m.settingsForm.fields[settingsFieldKeepAliveSeconds].SetValue("45")
	m.settingsForm.fields[settingsFieldTheme].SetValue("compact")
	m.settingsForm.fields[settingsFieldConfirmOverwrite].SetValue("false")
	m.settingsForm.fields[settingsFieldKnownHostsPolicy].SetValue(config.HostKeyStrict)
	m.settingsForm.blurCurrent()
	m.settingsForm.index = m.settingsForm.okIndex()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.err != "" {
		t.Fatalf("unexpected error: %s", m.err)
	}
	saved, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if saved.Settings.DefaultViewMode != config.ViewSplit ||
		saved.Settings.ASCIIFallback != config.ASCIIFallbackDisabled ||
		saved.Settings.FallbackRemotePath != "/var/tmp" ||
		saved.Settings.DraftTTLDays != 14 ||
		saved.Settings.TransferConcurrency != 8 ||
		saved.Settings.KeepAliveSeconds != 45 ||
		saved.Settings.Theme != "compact" ||
		saved.Settings.ConfirmOverwrite ||
		saved.Settings.KnownHostsPolicy != config.HostKeyStrict {
		t.Fatalf("saved settings = %+v", saved.Settings)
	}
}

func TestSettingsFormRejectsInvalidValues(t *testing.T) {
	store := config.NewStore(t.TempDir())
	m := NewModel(app.StateSettingsCenter, store, config.DefaultFile())
	m.settingsForm.fields[settingsFieldTransferConcurrency].SetValue("0")
	m.settingsForm.blurCurrent()
	m.settingsForm.index = m.settingsForm.okIndex()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if !strings.Contains(m.err, "transferConcurrency") {
		t.Fatalf("error = %q, want transferConcurrency validation", m.err)
	}
}

func TestSettingsOptionFieldsIgnoreLettersAndCycle(t *testing.T) {
	m := NewModel(app.StateSettingsCenter, config.NewStore(t.TempDir()), config.DefaultFile())
	m.settingsForm.blurCurrent()
	m.settingsForm.index = settingsFieldASCIIFallback
	m.settingsForm.fields[settingsFieldASCIIFallback].SetValue(config.ASCIIFallbackAuto)
	m.settingsForm.focusCurrent()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(Model)
	if got := m.settingsForm.fields[settingsFieldASCIIFallback].Value(); got != config.ASCIIFallbackAuto {
		t.Fatalf("ascii field after typing a = %q, want unchanged option", got)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = updated.(Model)
	if got := m.settingsForm.fields[settingsFieldASCIIFallback].Value(); got != config.ASCIIFallbackAlways {
		t.Fatalf("ascii field after right = %q, want next option", got)
	}
}

func TestSettingsViewIsCenteredPanelWithButtons(t *testing.T) {
	m := NewModel(app.StateSettingsCenter, config.NewStore(t.TempDir()), config.DefaultFile())
	m.width = 120
	m.height = 40
	got := m.View()
	if !strings.Contains(got, "+---") || !strings.Contains(got, "[ OK ]") || !strings.Contains(got, "[ Cancel ]") {
		t.Fatalf("settings view missing bordered panel/buttons: %q", got)
	}
	if !strings.Contains(got, "Left/Right or Space changes options") || !strings.Contains(got, "< ask") {
		t.Fatalf("settings view missing guide or option display: %q", got)
	}
	if strings.Contains(got, "Toggle ASCII") || strings.Contains(got, "[s]/[Enter] Save") {
		t.Fatalf("settings view should not render shortcut footer: %q", got)
	}
}

func TestSettingsKnownHostsPolicyCyclesToAsk(t *testing.T) {
	m := NewModel(app.StateSettingsCenter, config.NewStore(t.TempDir()), config.DefaultFile())
	m.settingsForm.blurCurrent()
	m.settingsForm.index = settingsFieldKnownHostsPolicy
	m.settingsForm.fields[settingsFieldKnownHostsPolicy].SetValue(config.HostKeyStrict)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = updated.(Model)
	if got := m.settingsForm.fields[settingsFieldKnownHostsPolicy].Value(); got != config.HostKeyAsk {
		t.Fatalf("knownHostsPolicy after right = %q, want ask", got)
	}
}

func TestSettingsInputFieldsStillAcceptLetters(t *testing.T) {
	m := NewModel(app.StateSettingsCenter, config.NewStore(t.TempDir()), config.DefaultFile())
	m.settingsForm.blurCurrent()
	m.settingsForm.index = settingsFieldTheme
	m.settingsForm.fields[settingsFieldTheme].SetValue("")
	m.settingsForm.focusCurrent()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(Model)
	if got := m.settingsForm.fields[settingsFieldTheme].Value(); got != "a" {
		t.Fatalf("theme field after typing a = %q, want literal input", got)
	}
}

func setServerFormValues(m *Model, values map[int]string) {
	for idx, value := range values {
		m.form.fields[idx].SetValue(value)
	}
}

type fakeSecrets struct {
	values map[string]string
}

type fakeRemoteDir struct {
	entries []os.FileInfo
}

func (f fakeRemoteDir) ReadDir(string) ([]os.FileInfo, error) {
	return f.entries, nil
}

type fakeFileInfo struct {
	name string
	size int64
	dir  bool
}

func (f fakeFileInfo) Name() string { return f.name }
func (f fakeFileInfo) Size() int64  { return f.size }
func (f fakeFileInfo) Mode() os.FileMode {
	if f.dir {
		return os.ModeDir | 0o755
	}
	return 0o644
}
func (f fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeFileInfo) IsDir() bool        { return f.dir }
func (f fakeFileInfo) Sys() any           { return nil }

func (f fakeSecrets) Get(ref string) (string, error) {
	return f.values[ref], nil
}

func (f fakeSecrets) Set(ref, value string) error {
	f.values[ref] = value
	return nil
}

func (f fakeSecrets) Delete(ref string) error {
	delete(f.values, ref)
	return nil
}

func visibleWidth(s string) int {
	return lipgloss.Width(s)
}

package tui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/motoryang/velo-ssh/internal/app"
	"github.com/motoryang/velo-ssh/internal/config"
	"github.com/motoryang/velo-ssh/internal/sshnet"
	"github.com/motoryang/velo-ssh/internal/term"
	"github.com/motoryang/velo-ssh/internal/transfer"
	"github.com/motoryang/velo-ssh/internal/updater"
	"github.com/motoryang/velo-ssh/internal/version"
)

type fileItem struct {
	Name     string
	Path     string
	Mode     os.FileMode
	Size     int64
	ModTime  time.Time
	IsDir    bool
	Selected bool
}

type serverHealth struct {
	Checked bool
	Online  bool
	Latency time.Duration
	Error   string
}

type Model struct {
	state                 app.AppState
	previous              app.AppState
	store                 *config.Store
	secrets               config.SecretStore
	config                config.File
	styles                styles
	ascii                 bool
	width                 int
	height                int
	cursor                int
	filter                string
	status                string
	err                   string
	serverHealth          map[string]serverHealth
	activePane            int
	localCursor           int
	remoteCursor          int
	showFileTime          bool
	localDir              string
	remoteDir             string
	localFiles            []fileItem
	remoteFiles           []fileItem
	tasks                 *transfer.Manager
	taskCursor            int
	taskDraftMode         bool
	draftCursor           int
	drafts                []config.Draft
	completedTasks        map[string]bool
	ssh                   *sshnet.Client
	activeServer          config.Server
	searching             bool
	searchInput           textinput.Model
	fileSearching         bool
	fileSearchInput       textinput.Model
	localFileFilter       string
	remoteFileFilter      string
	renaming              bool
	renameInput           textinput.Model
	creatingDir           bool
	mkdirInput            textinput.Model
	form                  serverForm
	settingsForm          settingsForm
	modalKind             string
	deleteID              string
	deleteName            string
	hostKeyErr            *sshnet.UnknownHostKeyError
	pendingHostKeyAction  string
	pendingHostKeyServer  config.Server
	pendingTransferDir    transfer.Direction
	pendingTransferItems  []fileItem
	pendingOverwrite      []string
	pendingFileDelete     []fileItem
	pendingDeleteRemote   bool
	pendingTaskCancelID   string
	pendingTaskCancelName string
	pendingTaskReturn     app.AppState
	pendingUpdate         updater.Release
	updateProgress        updater.Progress
	updateInstallCh       chan updateInstallMsg
	updateInstallCancel   context.CancelFunc
	compareResult         string
	compareProgress       updater.Progress
	compareCancel         chan struct{}
	compareCh             chan compareProgressMsg
	clipboardFiles        []fileItem
	clipboardRemote       bool
}

type serverForm struct {
	mode             string
	originalID       string
	fields           []textinput.Model
	initialValues    []string
	index            int
	remotePathManual bool
}

var serverFormLabels = []string{
	"ID",
	"Name",
	"Environment",
	"Host",
	"Port",
	"User",
	"Auth Type",
	"Key Path",
	"Password Ref",
	"Password",
	"Passphrase Ref",
	"Passphrase",
	"Description",
	"Default Remote Path",
	"Tags (comma separated)",
}

func NewModel(start app.AppState, store *config.Store, cfg config.File) Model {
	ascii := term.ShouldUseASCII(cfg.Settings.ASCIIFallback)
	cwd, _ := os.Getwd()
	m := Model{
		state:          start,
		previous:       app.StateServerList,
		store:          store,
		secrets:        config.NewSecretStore(store.SecretsPath()),
		config:         cfg,
		styles:         newStyles(ascii),
		ascii:          ascii,
		localDir:       cwd,
		remoteDir:      cfg.Settings.FallbackRemotePath,
		tasks:          transfer.NewManager(),
		completedTasks: map[string]bool{},
		serverHealth:   map[string]serverHealth{},
		activePane:     1,
	}
	m.tasks.SetConcurrency(cfg.Settings.TransferConcurrency)
	_, _ = store.PruneExpiredDrafts(cfg.Settings.DraftTTLDays)
	if drafts, err := store.LoadDrafts(); err == nil {
		m.drafts = drafts.Drafts
	}
	m.searchInput = textinput.New()
	m.searchInput.Placeholder = m.tr(textSearchServerPlaceholder)
	m.searchInput.CharLimit = 120
	m.fileSearchInput = textinput.New()
	m.fileSearchInput.Placeholder = m.tr(textSearchFilePlaceholder)
	m.fileSearchInput.CharLimit = 120
	m.renameInput = textinput.New()
	m.renameInput.CharLimit = 256
	m.mkdirInput = textinput.New()
	m.mkdirInput.CharLimit = 256
	m.settingsForm = newSettingsForm(cfg.Settings)
	m.refreshLocalFiles()
	return m
}

func (m Model) Init() tea.Cmd {
	if m.state == app.StateServerList && !m.config.Settings.DisableUpdateCheck {
		return tea.Batch(textinput.Blink, m.checkUpdateCmd(), m.checkServerHealthCmd())
	}
	if m.state == app.StateServerList {
		return tea.Batch(textinput.Blink, m.checkServerHealthCmd())
	}
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.ssh != nil {
			_ = m.ssh.WindowChange(msg.Height, msg.Width)
		}
	case errMsg:
		m.err = displayError(msg.err)
	case serverHealthMsg:
		if m.serverHealth == nil {
			m.serverHealth = map[string]serverHealth{}
		}
		m.serverHealth[msg.id] = msg.health
	case hostKeyPromptMsg:
		m.previous = m.state
		m.modalKind = modalHostKey
		m.hostKeyErr = msg.err
		m.pendingHostKeyAction = msg.action
		m.pendingHostKeyServer = msg.server
		m.state = app.StateConfirmModal
	case updateAvailableMsg:
		if m.state == app.StateServerList && !m.config.Settings.DisableUpdateCheck && msg.release.Version != m.config.Settings.SkippedUpdateVersion {
			m.previous = m.state
			m.modalKind = modalUpdateAvailable
			m.pendingUpdate = msg.release
			m.state = app.StateConfirmModal
		}
	case updateInstallMsg:
		if msg.done {
			versionInstalled := m.pendingUpdate.Version
			m.clearUpdatePrompt()
			m.updateInstallCh = nil
			m.state = m.previous
			if msg.err != nil {
				m.err = fmt.Sprintf(m.tr(textUpdateInstallFailed), msg.err)
				return m, nil
			}
			m.status = fmt.Sprintf(m.tr(textUpdateInstalled), versionInstalled)
			return m, nil
		}
		m.updateProgress = msg.progress
		m.modalKind = modalUpdateInstalling
		m.state = app.StateConfirmModal
		return m, waitUpdateInstallCmd(m.updateInstallCh)
	case shellConnectedMsg:
		m.ssh = msg.client
		m.activeServer = msg.server
		if msg.server.DefaultRemotePath != "" {
			m.remoteDir = msg.server.DefaultRemotePath
		}
		m.previous = m.state
		m.state = app.StateShell
		m.status = "SSH shell connected. Starting remote shell..."
		return m, m.runShellCmd()
	case fileManagerConnectedMsg:
		m.ssh = msg.client
		m.activeServer = msg.server
		if msg.server.DefaultRemotePath != "" {
			m.remoteDir = msg.server.DefaultRemotePath
		}
		m.previous = app.StateServerList
		m.state = app.StateFileManager
		return m, m.refreshFilePanesCmd()
	case filePanesLoadedMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.localFiles = msg.local
		m.remoteFiles = msg.remote
		m.localCursor = clampCursor(m.localCursor, len(filteredFileItems(m.localFiles, m.localFileFilter)))
		m.remoteCursor = clampCursor(m.remoteCursor, len(filteredFileItems(m.remoteFiles, m.remoteFileFilter)))
	case compareResultMsg:
		if msg.err != nil {
			m.err = displayError(msg.err)
			m.clearCompare()
			m.state = app.StateFileManager
			return m, nil
		}
		m.modalKind = modalCompareResult
		m.compareResult = msg.result
		m.state = app.StateConfirmModal
	case compareProgressMsg:
		if msg.done {
			if msg.err != nil {
				m.err = displayError(msg.err)
				m.clearCompare()
				m.state = app.StateFileManager
				return m, nil
			}
			m.modalKind = modalCompareResult
			m.compareResult = msg.result
			m.state = app.StateConfirmModal
			return m, nil
		}
		m.compareProgress = msg.progress
		m.modalKind = modalCompareProgress
		m.state = app.StateConfirmModal
		return m, waitCompareCmd(m.compareCh)
	case remoteEditPreparedMsg:
		if msg.err != nil {
			m.err = displayError(msg.err)
			return m, nil
		}
		m.status = fmt.Sprintf("Editing remote draft %s.", msg.draft.RemotePath)
		return m, m.openRemoteDraftEditorCmd(msg.draft)
	case remoteEditFinishedMsg:
		if msg.err != nil {
			failed := markDraftFailed(msg.draft)
			if err := m.store.UpsertDraft(failed); err != nil {
				m.err = err.Error()
				return m, nil
			}
			m.err = fmt.Sprintf("editor failed: %v; draft saved for retry", msg.err)
			if f, err := m.store.LoadDrafts(); err == nil {
				m.drafts = f.Drafts
			}
			return m, nil
		}
		return m, m.syncRemoteEditDraftCmd(msg.draft)
	case localFileOpenFinishedMsg:
		if msg.err != nil {
			m.err = fmt.Sprintf("open local file failed: %v", msg.err)
			return m, nil
		}
		m.status = fmt.Sprintf("Opened local file %s.", msg.path)
		return m, nil
	case transferStartedMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.status = msg.message
		return m, taskCenterTickCmd()
	case draftRetryStartedMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			_ = m.store.UpdateDraftStatus(msg.draftID, config.DraftFailed)
			if f, err := m.store.LoadDrafts(); err == nil {
				m.drafts = f.Drafts
			}
			return m, nil
		}
		m.status = msg.message
		return m, taskCenterTickCmd()
	case overwritePromptMsg:
		m.previous = m.state
		m.modalKind = modalOverwrite
		m.pendingTransferDir = msg.direction
		m.pendingTransferItems = msg.items
		m.pendingOverwrite = msg.targets
		m.state = app.StateConfirmModal
	case shellFinishedMsg:
		m.status = ""
		if msg.err != nil {
			m.err = msg.err.Error()
		}
		switch msg.action.Command {
		case "files":
			m.previous = app.StateShell
			m.state = app.StateFileManager
			return m, m.refreshFilePanesCmd()
		case "tasks":
			m.previous = app.StateShell
			m.state = app.StateTaskCenter
			return m, taskCenterTickCmd()
		case "settings":
			m.previous = app.StateShell
			m.openSettingsCenter()
		case "back":
			m.state = app.StateServerList
		case "reconnect":
			if m.ssh != nil {
				return m, m.reconnectCmd()
			}
			m.state = app.StateServerList
		case "quit":
			m.shutdown()
			m.state = app.StateServerList
		default:
			m.state = app.StateServerList
		}
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || (m.state == app.StateServerList && msg.String() == "q") {
			m.shutdown()
			return m, tea.Quit
		}
		m.err = ""
		m.status = ""
		if m.width > 0 && (m.width < 80 || m.height < 24) {
			return m, nil
		}
		return m.handleKey(msg)
	case taskTickMsg:
		cmd, active := m.handleTaskTick()
		if m.state == app.StateTaskCenter || (m.state == app.StateFileManager && active) {
			return m, tea.Batch(cmd, taskCenterTickCmd())
		}
		return m, cmd
	}
	return m, nil
}

func (m Model) View() string {
	if m.width > 0 && (m.width < 80 || m.height < 24) {
		return "Action failed: target=terminal, reason=terminal too small. Recovery: resize to at least 80x24.\n"
	}
	var body string
	switch m.state {
	case app.StateSettingsCenter:
		body = m.viewSettings()
	case app.StateServerForm:
		body = m.viewServerForm()
	case app.StateFileManager:
		body = m.viewFileManager()
	case app.StateTaskCenter:
		body = m.viewTaskCenter()
	case app.StateConfirmModal:
		body = m.viewConfirmModal()
	case app.StateShell:
		body = m.viewShell()
	case app.StateHelp:
		body = sshnet.EscapeHelpWithLanguage(m.config.Settings.Language)
	default:
		body = m.viewServerList()
	}
	if m.err != "" {
		body += "\n" + m.styles.error.Render(m.tr(textErrorPrefix)+": "+m.err)
	}
	if m.status != "" {
		body += "\n" + m.styles.muted.Render(m.status)
	}
	return m.withFooter(body, m.helpText())
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case app.StateServerList:
		return m.handleServerListKey(msg)
	case app.StateServerForm:
		return m.handleServerFormKey(msg)
	case app.StateSettingsCenter:
		return m.handleSettingsKey(msg)
	case app.StateFileManager:
		return m.handleFileManagerKey(msg)
	case app.StateTaskCenter:
		return m.handleTaskCenterKey(msg)
	case app.StateHelp:
		if msg.String() == keyEsc || msg.String() == "q" {
			m.state = m.previous
		}
	case app.StateConfirmModal:
		return m.handleConfirmKey(msg)
	case app.StateShell:
		switch msg.String() {
		case "o", keyEnter:
			if m.ssh == nil {
				m.err = "ssh client is not connected"
				m.state = app.StateServerList
				return m, nil
			}
			return m, m.runShellCmd()
		case keyEsc:
			m.state = app.StateServerList
		}
	}
	return m, nil
}

func (m Model) handleTaskCenterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.taskDraftMode {
		return m.handleDraftCenterKey(msg)
	}
	tasks := m.taskSnapshots()
	m.taskCursor = clampCursor(m.taskCursor, len(tasks))
	switch msg.String() {
	case "t", keyEsc, "q":
		m.state = m.previous
	case "D":
		m.taskDraftMode = true
		if drafts, err := m.store.LoadDrafts(); err == nil {
			m.drafts = drafts.Drafts
		}
		m.draftCursor = clampCursor(m.draftCursor, len(m.retryableDrafts()))
	case keyUp, "k":
		if m.taskCursor > 0 {
			m.taskCursor--
		}
	case keyDown, "j":
		if m.taskCursor < len(tasks)-1 {
			m.taskCursor++
		}
	case "x":
		if len(tasks) == 0 {
			m.err = "no task selected"
			return m, nil
		}
		task := tasks[m.taskCursor]
		returnState := m.previous
		m.previous = m.state
		m.modalKind = modalTaskCancel
		m.pendingTaskCancelID = task.ID
		m.pendingTaskCancelName = fmt.Sprintf("%s %s -> %s", task.Direction, task.SourcePath, task.TargetPath)
		m.pendingTaskReturn = returnState
		m.state = app.StateConfirmModal
	case "p":
		if len(tasks) == 0 {
			m.err = "no task selected"
			return m, nil
		}
		if err := m.tasks.Pause(tasks[m.taskCursor].ID); err != nil {
			m.err = err.Error()
			return m, nil
		}
		m.status = fmt.Sprintf("Paused task %s.", tasks[m.taskCursor].ID)
	case "r":
		if len(tasks) == 0 {
			m.err = "no task selected"
			return m, nil
		}
		if err := m.tasks.Resume(tasks[m.taskCursor].ID); err != nil {
			m.err = err.Error()
			return m, nil
		}
		m.status = fmt.Sprintf("Resumed task %s.", tasks[m.taskCursor].ID)
	case "R":
		m.status = "Task center refreshed."
	}
	return m, nil
}

func (m Model) handleDraftCenterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	drafts := m.retryableDrafts()
	m.draftCursor = clampCursor(m.draftCursor, len(drafts))
	switch msg.String() {
	case "t", keyEsc, "q":
		m.taskDraftMode = false
	case "D":
		m.taskDraftMode = false
	case keyUp, "k":
		if m.draftCursor > 0 {
			m.draftCursor--
		}
	case keyDown, "j":
		if m.draftCursor < len(drafts)-1 {
			m.draftCursor++
		}
	case "x":
		if len(drafts) == 0 {
			m.err = "no draft selected"
			return m, nil
		}
		if err := m.store.UpdateDraftStatus(drafts[m.draftCursor].ID, config.DraftResolved); err != nil {
			m.err = err.Error()
			return m, nil
		}
		m.status = fmt.Sprintf("Resolved draft %s.", drafts[m.draftCursor].ID)
		if f, err := m.store.LoadDrafts(); err == nil {
			m.drafts = f.Drafts
		}
	case "r":
		if len(drafts) == 0 {
			m.err = "no draft selected"
			return m, nil
		}
		draft := drafts[m.draftCursor]
		if m.ssh == nil || m.activeServer.ID == "" || m.activeServer.ID != draft.ServerID {
			m.err = "retry draft failed: connect to the draft server before retrying"
			return m, nil
		}
		if err := m.store.UpdateDraftStatus(draft.ID, config.DraftSyncing); err != nil {
			m.err = err.Error()
			return m, nil
		}
		if f, err := m.store.LoadDrafts(); err == nil {
			m.drafts = f.Drafts
		}
		task := transfer.NewTask(newTaskID("draft"), transfer.Upload, draft.LocalPath, draft.RemotePath)
		task.ServerID = draft.ServerID
		m.status = fmt.Sprintf("Retrying draft %s.", draft.ID)
		return m, m.startDraftRetryCmd(task, draft.ID)
	case "R":
		if f, err := m.store.LoadDrafts(); err == nil {
			m.drafts = f.Drafts
			m.status = "Draft retry center refreshed."
		}
	}
	return m, nil
}

func (m *Model) handleTaskTick() (tea.Cmd, bool) {
	if m.completedTasks == nil {
		m.completedTasks = map[string]bool{}
	}
	active := false
	refresh := false
	for _, task := range m.taskSnapshots() {
		switch task.Status {
		case transfer.TaskQueued, transfer.TaskRunning, transfer.TaskPaused:
			active = true
		case transfer.TaskSucceeded:
			if !m.completedTasks[task.ID] {
				m.completedTasks[task.ID] = true
				refresh = true
				m.status = fmt.Sprintf("Transfer task %s completed.", task.ID)
			}
		case transfer.TaskFailed, transfer.TaskCanceled:
			m.completedTasks[task.ID] = true
		}
	}
	if refresh && (m.state == app.StateFileManager || m.previous == app.StateFileManager) {
		return m.refreshFilePanesCmd(), active
	}
	return nil, active
}

func (m Model) handleServerListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.searching {
		switch msg.String() {
		case keyEsc:
			m.searching = false
			m.searchInput.Blur()
			m.filter = ""
			m.searchInput.SetValue("")
			m.cursor = 0
			return m, nil
		case keyEnter:
			m.searching = false
			m.searchInput.Blur()
			m.filter = m.searchInput.Value()
			m.cursor = 0
			return m, nil
		default:
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			m.filter = m.searchInput.Value()
			m.cursor = clampCursor(m.cursor, len(m.filteredServers()))
			return m, cmd
		}
	}
	servers := m.filteredServers()
	switch msg.String() {
	case "k", keyUp:
		m.cursor = previousVisibleServerCursor(servers, m.cursor)
	case "j", keyDown:
		m.cursor = nextVisibleServerCursor(servers, m.cursor)
	case "/":
		m.searching = true
		m.searchInput.Focus()
		m.searchInput.SetValue(m.filter)
		return m, textinput.Blink
	case "S":
		m.previous = m.state
		m.openSettingsCenter()
	case "f":
		if len(servers) == 0 {
			m.err = "file manager requires at least one configured server"
			return m, nil
		}
		if m.ssh != nil && m.activeServer.ID == servers[m.cursor].ID {
			m.previous = m.state
			m.state = app.StateFileManager
			return m, m.refreshFilePanesCmd()
		}
		return m, m.connectFileManagerCmd(servers[m.cursor])
	case "a":
		m.previous = m.state
		m.form = newServerForm("add", config.Server{
			Port:              22,
			AuthType:          config.AuthAgent,
			DefaultRemotePath: m.config.Settings.FallbackRemotePath,
			Env:               "dev",
		})
		m.state = app.StateServerForm
		return m, textinput.Blink
	case "e":
		if len(servers) > 0 {
			m.previous = m.state
			m.form = newServerForm("edit", servers[m.cursor])
			m.state = app.StateServerForm
			return m, textinput.Blink
		}
	case "c":
		if len(servers) > 0 {
			clone := servers[m.cursor]
			clone.ID = ""
			clone.Name = clone.Name + " Copy"
			m.previous = m.state
			m.form = newServerForm("clone", clone)
			m.form.originalID = servers[m.cursor].ID
			m.state = app.StateServerForm
			return m, textinput.Blink
		}
	case "d":
		if len(servers) > 0 {
			m.previous = m.state
			m.modalKind = modalDelete
			m.deleteID = servers[m.cursor].ID
			m.deleteName = servers[m.cursor].Name
			m.state = app.StateConfirmModal
		}
	case keyEnter:
		if len(servers) == 0 {
			m.err = "no server selected"
			return m, nil
		}
		return m, m.connectShellCmd(servers[m.cursor])
	}
	return m, nil
}

func (m Model) handleServerFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEsc:
		if m.form.dirty() {
			m.modalKind = modalServerFormDiscard
			m.state = app.StateConfirmModal
			return m, nil
		}
		m.state = m.previous
		return m, nil
	case "tab", keyDown:
		m.form.focusNext()
		return m, nil
	case "shift+tab", keyUp:
		m.form.focusPrev()
		return m, nil
	case "left":
		if m.form.index == serverFieldAuthType {
			m.form.cycleAuthType(-1)
			return m, nil
		}
	case "right", " ":
		if m.form.index == serverFieldAuthType {
			m.form.cycleAuthType(1)
			return m, nil
		}
	case keyEnter:
		if m.form.index != m.form.lastVisibleIndex() {
			m.form.focusNext()
			return m, nil
		}
		return m.saveServerForm()
	}
	if m.form.index == serverFieldAuthType {
		return m, nil
	}
	var cmd tea.Cmd
	oldUser := m.form.fields[serverFieldUser].Value()
	m.form.fields[m.form.index], cmd = m.form.fields[m.form.index].Update(msg)
	if m.form.index == serverFieldUser {
		if m.form.fields[serverFieldDefaultRemotePath].Value() == defaultRemotePathForUser(oldUser) {
			m.form.remotePathManual = false
		}
		m.form.setUser(m.form.fields[serverFieldUser].Value())
	}
	if m.form.index == serverFieldDefaultRemotePath {
		m.form.remotePathManual = true
	}
	return m, cmd
}

func (m Model) saveServerForm() (tea.Model, tea.Cmd) {
	formValue, err := m.form.server()
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	srv := formValue.Server
	original, hasOriginal := findServerByID(m.config.Servers, m.form.originalID)
	if m.form.mode == "edit" {
		srv.ID = m.form.originalID
	}
	if srv.ID == "" {
		srv.ID = uniqueServerID(m.config.Servers, srv)
	}
	if existing, ok := findServerByID(m.config.Servers, srv.ID); ok {
		if m.form.mode == "add" || existing.ID != m.form.originalID {
			m.err = fmt.Sprintf("server id %q already exists", srv.ID)
			return m, nil
		}
	}
	if m.form.mode == "edit" && hasOriginal {
		srv.CreatedAt = original.CreatedAt
	}
	switch srv.AuthType {
	case config.AuthPassword:
		srv.KeyPath = ""
		oldRef := srv.PasswordRef
		if m.form.mode == "edit" && oldRef != "" {
			srv.PasswordRef = oldRef
		} else {
			srv.PasswordRef = config.PasswordRef(srv.ID)
		}
		if formValue.Password != "" {
			if err := m.secrets.Set(srv.PasswordRef, formValue.Password); err != nil {
				m.err = fmt.Sprintf("store password in secret store: %v", err)
				return m, nil
			}
		} else if m.form.mode == "clone" && oldRef != "" {
			if err := m.copySecret(oldRef, srv.PasswordRef, "password"); err != nil {
				m.err = err.Error()
				return m, nil
			}
		}
		srv.PassphraseRef = ""
	case config.AuthKey:
		srv.PasswordRef = ""
		oldRef := srv.PassphraseRef
		if formValue.Passphrase != "" {
			if m.form.mode == "edit" && oldRef != "" {
				srv.PassphraseRef = oldRef
			} else {
				srv.PassphraseRef = config.PassphraseRef(srv.ID)
			}
			if err := m.secrets.Set(srv.PassphraseRef, formValue.Passphrase); err != nil {
				m.err = fmt.Sprintf("store passphrase in secret store: %v", err)
				return m, nil
			}
		} else if m.form.mode == "clone" && oldRef != "" {
			srv.PassphraseRef = config.PassphraseRef(srv.ID)
			if err := m.copySecret(oldRef, srv.PassphraseRef, "passphrase"); err != nil {
				m.err = err.Error()
				return m, nil
			}
		}
	case config.AuthAgent:
		srv.KeyPath = ""
		srv.PasswordRef = ""
		srv.PassphraseRef = ""
	}
	if m.form.mode == "edit" && m.form.originalID != srv.ID {
		if err := m.store.DeleteServer(m.form.originalID); err != nil {
			m.err = err.Error()
			return m, nil
		}
	}
	if err := m.store.UpsertServer(srv); err != nil {
		m.err = err.Error()
		return m, nil
	}
	cfg, err := m.store.Load()
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.config = cfg
	m.state = app.StateServerList
	m.filter = ""
	m.searchInput.SetValue("")
	m.cursor = indexServerByID(m.config.Servers, srv.ID)
	if m.form.mode == "edit" {
		m.status = m.tr(textServerUpdated)
	} else {
		m.status = m.tr(textServerAdded)
	}
	return m, nil
}

func (m Model) copySecret(fromRef, toRef, label string) error {
	if fromRef == "" || toRef == "" || fromRef == toRef {
		return nil
	}
	value, err := m.secrets.Get(fromRef)
	if err != nil {
		return fmt.Errorf("copy %s in secret store: %v", label, err)
	}
	if err := m.secrets.Set(toRef, value); err != nil {
		return fmt.Errorf("store cloned %s in secret store: %v", label, err)
	}
	return nil
}

func (m Model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.modalKind == modalHostKey {
		return m.handleHostKeyConfirmKey(msg)
	}
	if m.modalKind == modalOverwrite {
		return m.handleOverwriteConfirmKey(msg)
	}
	if m.modalKind == modalFileDelete {
		return m.handleFileDeleteConfirmKey(msg)
	}
	if m.modalKind == modalTaskCancel {
		return m.handleTaskCancelConfirmKey(msg)
	}
	if m.modalKind == modalServerFormDiscard {
		return m.handleServerFormDiscardConfirmKey(msg)
	}
	if m.modalKind == modalUpdateAvailable {
		return m.handleUpdateAvailableConfirmKey(msg)
	}
	if m.modalKind == modalUpdateInstalling {
		return m.handleUpdateInstallingConfirmKey(msg)
	}
	if m.modalKind == modalCompareResult {
		switch msg.String() {
		case keyEsc, keyEnter, "q", "Q", "n", "N":
			m.modalKind = ""
			m.clearCompare()
			m.state = app.StateFileManager
		}
		return m, nil
	}
	if m.modalKind == modalCompareProgress {
		switch msg.String() {
		case keyEsc, "q", "Q", "n", "N":
			if m.compareCancel != nil {
				close(m.compareCancel)
				m.compareCancel = nil
			}
			m.status = "Compare canceled."
			m.clearCompare()
			m.state = app.StateFileManager
		}
		return m, nil
	}
	switch msg.String() {
	case keyEsc, "n", "N":
		m.state = m.previous
	case keyEnter, "y", "Y":
		if err := m.store.DeleteServer(m.deleteID); err != nil {
			m.err = err.Error()
			m.state = m.previous
			return m, nil
		}
		cfg, err := m.store.Load()
		if err != nil {
			m.err = err.Error()
			m.state = m.previous
			return m, nil
		}
		m.config = cfg
		m.cursor = clampCursor(m.cursor, len(m.filteredServers()))
		m.status = fmt.Sprintf(m.tr(textServerDeleted), m.deleteName)
		m.deleteID = ""
		m.deleteName = ""
		m.state = m.previous
	}
	return m, nil
}

func (m Model) handleServerFormDiscardConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEsc, "n", "N":
		m.modalKind = ""
		m.state = app.StateServerForm
	case keyEnter, "y", "Y":
		m.modalKind = ""
		m.state = m.previous
		m.status = m.tr(textServerDiscarded)
	}
	return m, nil
}

func (m Model) handleUpdateAvailableConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEsc, "n", "N":
		m.status = m.tr(textUpdateCanceled)
		m.clearUpdatePrompt()
		m.state = m.previous
	case "s", "S":
		versionToSkip := m.pendingUpdate.Version
		m.config.Settings.SkippedUpdateVersion = versionToSkip
		if err := m.store.Save(m.config); err != nil {
			m.err = err.Error()
			return m, nil
		}
		m.status = fmt.Sprintf(m.tr(textUpdateSkipped), versionToSkip)
		m.clearUpdatePrompt()
		m.state = m.previous
	case keyEnter, "u", "U":
		m.modalKind = modalUpdateInstalling
		m.updateProgress = updater.Progress{Stage: "selecting"}
		m.updateInstallCh = make(chan updateInstallMsg, 16)
		m.state = app.StateConfirmModal

		ctx, cancel := context.WithCancel(context.Background())
		m.updateInstallCancel = cancel
		return m, startUpdateInstallCmd(ctx, m.pendingUpdate, m.updateInstallCh)
	}
	return m, nil
}

func (m Model) handleUpdateInstallingConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEsc, "n", "N":
		m.status = m.tr(textUpdateCanceled)
		m.clearUpdatePrompt()
		m.state = m.previous
	}
	return m, nil
}

func (m *Model) clearUpdatePrompt() {
	m.modalKind = ""
	m.pendingUpdate = updater.Release{}
	m.updateProgress = updater.Progress{}
	if m.updateInstallCancel != nil {
		m.updateInstallCancel()
		m.updateInstallCancel = nil
	}
}

func (m Model) handleOverwriteConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEsc, "n", "N":
		m.status = m.tr(textTransferOverwriteCancel)
		m.clearOverwritePrompt()
		m.state = m.previous
	case keyEnter, "y", "Y":
		direction := m.pendingTransferDir
		items := append([]fileItem(nil), m.pendingTransferItems...)
		m.clearOverwritePrompt()
		m.state = app.StateFileManager
		if direction == transfer.Upload {
			return m, m.startUploadCmd(true, items)
		}
		return m, m.startDownloadCmd(true, items)
	}
	return m, nil
}

func (m *Model) clearOverwritePrompt() {
	m.modalKind = ""
	m.pendingTransferDir = ""
	m.pendingTransferItems = nil
	m.pendingOverwrite = nil
}

func (m Model) handleFileDeleteConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEsc, "n", "N":
		m.status = m.tr(textDeleteCanceled)
		m.clearFileDeletePrompt()
		m.state = m.previous
	case keyEnter, "y", "Y":
		items := append([]fileItem(nil), m.pendingFileDelete...)
		remote := m.pendingDeleteRemote
		m.clearFileDeletePrompt()
		m.state = app.StateFileManager
		return m, m.deleteFilesCmd(items, remote)
	}
	return m, nil
}

func (m *Model) clearFileDeletePrompt() {
	m.modalKind = ""
	m.pendingFileDelete = nil
	m.pendingDeleteRemote = false
}

func (m Model) handleTaskCancelConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEsc, "n", "N":
		m.status = "Task cancel aborted."
		returnState := m.pendingTaskReturn
		m.clearTaskCancelPrompt()
		m.state = app.StateTaskCenter
		m.previous = returnState
	case keyEnter, "y", "Y":
		id := m.pendingTaskCancelID
		returnState := m.pendingTaskReturn
		m.clearTaskCancelPrompt()
		m.state = app.StateTaskCenter
		m.previous = returnState
		if err := m.tasks.CancelAndRemove(id); err != nil {
			m.err = err.Error()
			return m, nil
		}
		m.taskCursor = clampCursor(m.taskCursor, len(m.taskSnapshots()))
		m.status = fmt.Sprintf("Canceled and removed task %s.", id)
	}
	return m, nil
}

func (m *Model) clearTaskCancelPrompt() {
	m.modalKind = ""
	m.pendingTaskCancelID = ""
	m.pendingTaskCancelName = ""
	m.pendingTaskReturn = app.StateServerList
}

func (m Model) handleHostKeyConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEsc, "n", "N":
		m.status = "Host key was not trusted. Connection canceled."
		m.clearHostKeyPrompt()
		m.state = m.previous
	case keyEnter, "y", "Y":
		if err := sshnet.AcceptHostKey(m.hostKeyErr); err != nil {
			m.err = fmt.Sprintf("accept host key: %v", err)
			return m, nil
		}
		action := m.pendingHostKeyAction
		server := m.pendingHostKeyServer
		m.clearHostKeyPrompt()
		m.status = "Host key accepted. Retrying connection."
		switch action {
		case hostKeyActionShell:
			return m, m.connectShellCmd(server)
		case hostKeyActionFileManager:
			return m, m.connectFileManagerCmd(server)
		case hostKeyActionReconnect:
			return m, m.reconnectCmd()
		default:
			m.state = m.previous
		}
	}
	return m, nil
}

func (m *Model) clearHostKeyPrompt() {
	m.modalKind = ""
	m.hostKeyErr = nil
	m.pendingHostKeyAction = ""
	m.pendingHostKeyServer = config.Server{}
}

func (m Model) handleSettingsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab", keyDown:
		m.settingsForm.focusNext()
		return m, nil
	case "shift+tab", keyUp:
		m.settingsForm.focusPrev()
		return m, nil
	case keyEnter:
		if m.settingsForm.okFocused() {
			return m.saveSettingsForm()
		}
		if m.settingsForm.cancelFocused() {
			m.state = m.previous
			return m, nil
		}
		if m.settingsForm.optionFocused() {
			m.settingsForm.cycleOption(1)
		}
	case keyLeft:
		m.settingsForm.cycleOption(-1)
	case keyRight, " ":
		m.settingsForm.cycleOption(1)
	default:
		if m.settingsForm.inputFocused() {
			var cmd tea.Cmd
			m.settingsForm.fields[m.settingsForm.index], cmd = m.settingsForm.fields[m.settingsForm.index].Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m Model) saveSettingsForm() (tea.Model, tea.Cmd) {
	settings, err := m.settingsForm.settings()
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	settings.SkippedUpdateVersion = m.config.Settings.SkippedUpdateVersion
	m.config.Settings = settings
	if err := m.store.Save(m.config); err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.tasks.SetConcurrency(settings.TransferConcurrency)
	m.ascii = term.ShouldUseASCII(m.config.Settings.ASCIIFallback)
	m.styles = newStyles(m.ascii)
	m.status = m.tr(textSettingsSaved)
	m.state = m.previous
	return m, nil
}

func (m *Model) openSettingsCenter() {
	m.settingsForm = newSettingsForm(m.config.Settings)
	m.state = app.StateSettingsCenter
}

func (m Model) handleFileManagerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.fileSearching {
		switch msg.String() {
		case keyEsc:
			m.fileSearching = false
			m.fileSearchInput.Blur()
			return m, nil
		case keyEnter:
			m.applyCurrentFileFilter(m.fileSearchInput.Value())
			m.fileSearching = false
			m.fileSearchInput.Blur()
			return m, nil
		default:
			var cmd tea.Cmd
			m.fileSearchInput, cmd = m.fileSearchInput.Update(msg)
			return m, cmd
		}
	}
	if m.renaming {
		switch msg.String() {
		case keyEsc:
			m.renaming = false
			m.renameInput.Blur()
			return m, nil
		case keyEnter:
			m.renaming = false
			m.renameInput.Blur()
			return m, m.renameCurrentCmd(m.renameInput.Value())
		default:
			var cmd tea.Cmd
			m.renameInput, cmd = m.renameInput.Update(msg)
			return m, cmd
		}
	}
	if m.creatingDir {
		switch msg.String() {
		case keyEsc:
			m.creatingDir = false
			m.mkdirInput.Blur()
			return m, nil
		case keyEnter:
			m.creatingDir = false
			m.mkdirInput.Blur()
			return m, m.mkdirCurrentCmd(m.mkdirInput.Value())
		default:
			var cmd tea.Cmd
			m.mkdirInput, cmd = m.mkdirInput.Update(msg)
			return m, cmd
		}
	}
	files := m.currentFiles()
	cursor := m.currentFileCursor()
	switch msg.String() {
	case "q":
		if m.ssh != nil {
			m.previous = m.state
			m.state = app.StateShell
			return m, m.runShellCmd()
		}
		m.state = app.StateServerList
	case "/":
		m.fileSearching = true
		m.fileSearchInput.SetValue(m.currentFileFilter())
		m.fileSearchInput.Focus()
		return m, textinput.Blink
	case "tab":
		if m.config.Settings.DefaultViewMode != config.ViewSplit {
			m.activePane = 1
			m.remoteCursor = clampCursor(m.remoteCursor, len(filteredFileItems(m.remoteFiles, m.remoteFileFilter)))
			return m, nil
		}
		m.activePane = 1 - m.activePane
	case keyUp, "k":
		if cursor > 0 {
			m.setCurrentFileCursor(cursor - 1)
		}
	case keyDown, "j":
		if cursor < len(files)-1 {
			m.setCurrentFileCursor(cursor + 1)
		}
	case " ":
		m.toggleSelected()
	case "a":
		m.selectAll(true)
	case "c":
		m.selectAll(false)
	case "b":
		if m.config.Settings.DefaultViewMode == config.ViewSplit {
			m.config.Settings.DefaultViewMode = config.ViewSingle
			m.activePane = 1
			m.remoteCursor = clampCursor(m.remoteCursor, len(filteredFileItems(m.remoteFiles, m.remoteFileFilter)))
		} else {
			m.config.Settings.DefaultViewMode = config.ViewSplit
		}
	case "t":
		m.previous = m.state
		m.state = app.StateTaskCenter
		return m, taskCenterTickCmd()
	case "S":
		m.previous = m.state
		m.openSettingsCenter()
	case keyEnter:
		if len(files) == 0 || cursor >= len(files) {
			return m, nil
		}
		item := files[cursor]
		if !item.IsDir {
			if m.config.Settings.DefaultViewMode == config.ViewSingle || m.activePane == 1 {
				return m, m.prepareRemoteEditCmd(item)
			}
			if item.Name == ".." {
				return m, nil
			}
			return m, m.openLocalFileEditorCmd(item)
		}
		if m.config.Settings.DefaultViewMode != config.ViewSingle && m.activePane == 0 {
			m.localDir = item.Path
			m.localCursor = 0
		} else {
			m.remoteDir = item.Path
			m.remoteCursor = 0
		}
		return m, m.refreshFilePanesCmd()
	case "r":
		if len(files) == 0 || cursor >= len(files) || files[cursor].Name == ".." {
			m.err = "no file selected for rename"
			return m, nil
		}
		m.renaming = true
		m.renameInput.SetValue(files[cursor].Name)
		m.renameInput.Focus()
		return m, textinput.Blink
	case "n":
		m.creatingDir = true
		m.mkdirInput.SetValue("")
		m.mkdirInput.Focus()
		return m, textinput.Blink
	case "x":
		items := selectedFileItems(files, cursor)
		if len(items) == 0 {
			m.err = "no file selected for delete"
			return m, nil
		}
		m.previous = m.state
		m.modalKind = modalFileDelete
		m.pendingFileDelete = items
		m.pendingDeleteRemote = m.config.Settings.DefaultViewMode == config.ViewSingle || m.activePane == 1
		m.state = app.StateConfirmModal
	case "y":
		items := selectedFileItems(files, cursor)
		if len(items) == 0 {
			m.err = "no file selected for copy"
			return m, nil
		}
		m.clipboardFiles = append([]fileItem(nil), items...)
		m.clipboardRemote = m.config.Settings.DefaultViewMode == config.ViewSingle || m.activePane == 1
		m.status = fmt.Sprintf("Copied %d item(s).", len(items))
	case "v":
		return m, m.pasteClipboardCmd(false)
	case "M":
		return m, m.pasteClipboardCmd(true)
	case "u":
		return m, m.startUploadCmd(false, nil)
	case "d":
		return m, m.startDownloadCmd(false, nil)
	case "=":
		return m.startCompareFromSelection(files, cursor)
	case "E":
		if m.config.Settings.DefaultViewMode != config.ViewSingle && m.activePane == 0 {
			m.err = "remote edit failed: focus the remote pane"
			return m, nil
		}
		if len(files) == 0 || cursor >= len(files) {
			m.err = "remote edit failed: no remote file selected"
			return m, nil
		}
		return m, m.prepareRemoteEditCmd(files[cursor])
	case "m":
		m.showFileTime = !m.showFileTime
	case "R":
		return m, m.refreshFilePanesCmd()
	}
	return m, nil
}

func (m Model) connectShellCmd(srv config.Server) tea.Cmd {
	return func() tea.Msg {
		client := sshnet.NewClient(m.config.Settings, config.OSKeyring{})
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := client.Connect(ctx, srv); err != nil {
			var unknown *sshnet.UnknownHostKeyError
			if errors.As(err, &unknown) {
				return hostKeyPromptMsg{err: unknown, server: srv, action: hostKeyActionShell}
			}
			return errMsg{err}
		}
		return shellConnectedMsg{client: client, server: srv}
	}
}

type errMsg struct{ err error }
type serverHealthMsg struct {
	id     string
	health serverHealth
}
type updateAvailableMsg struct{ release updater.Release }
type updateInstallMsg struct {
	progress updater.Progress
	done     bool
	err      error
}
type hostKeyPromptMsg struct {
	err    *sshnet.UnknownHostKeyError
	server config.Server
	action string
}
type shellConnectedMsg struct {
	client *sshnet.Client
	server config.Server
}
type fileManagerConnectedMsg struct {
	client *sshnet.Client
	server config.Server
}
type filePanesLoadedMsg struct {
	local  []fileItem
	remote []fileItem
	err    error
}
type compareResultMsg struct {
	result string
	err    error
}
type transferStartedMsg struct {
	message string
	err     error
}
type draftRetryStartedMsg struct {
	draftID string
	message string
	err     error
}
type overwritePromptMsg struct {
	direction transfer.Direction
	items     []fileItem
	targets   []string
}

func (m Model) checkUpdateCmd() tea.Cmd {
	return func() tea.Msg {
		rel, newer, err := updater.CheckLatest(context.Background(), version.String())
		if err != nil || !newer {
			return nil
		}
		if rel.Version == m.config.Settings.SkippedUpdateVersion {
			return nil
		}
		return updateAvailableMsg{release: rel}
	}
}

func (m Model) checkServerHealthCmd() tea.Cmd {
	servers := append([]config.Server(nil), m.config.Servers...)
	if len(servers) == 0 {
		return nil
	}
	cmds := make([]tea.Cmd, 0, len(servers))
	for _, srv := range servers {
		server := srv
		cmds = append(cmds, func() tea.Msg {
			start := time.Now()
			addr := net.JoinHostPort(server.Host, fmt.Sprintf("%d", server.Port))
			conn, err := net.DialTimeout("tcp", addr, 800*time.Millisecond)
			if err != nil {
				return serverHealthMsg{id: server.ID, health: serverHealth{Checked: true, Online: false, Error: err.Error()}}
			}
			_ = conn.Close()
			return serverHealthMsg{id: server.ID, health: serverHealth{Checked: true, Online: true, Latency: time.Since(start)}}
		})
	}
	return tea.Batch(cmds...)
}

func (m Model) startDraftRetryCmd(task *transfer.Task, draftID string) tea.Cmd {
	return func() tea.Msg {
		if m.ssh == nil {
			return draftRetryStartedMsg{draftID: draftID, err: fmt.Errorf("retry draft failed: ssh client is not connected")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		client, err := m.ssh.OpenSFTP(ctx)
		if err != nil {
			return draftRetryStartedMsg{draftID: draftID, err: err}
		}
		m.tasks.Start(context.Background(), client, task)
		return draftRetryStartedMsg{draftID: draftID, message: fmt.Sprintf("Started draft retry task %s.", task.ID)}
	}
}

func startUpdateInstallCmd(ctx context.Context, rel updater.Release, ch chan updateInstallMsg) tea.Cmd {
	return func() tea.Msg {
		go func() {
			err := updater.InstallLatestWithProgress(ctx, rel, func(progress updater.Progress) {
				if ctx.Err() != nil {
					return
				}
				ch <- updateInstallMsg{progress: progress}
			})
			ch <- updateInstallMsg{done: true, err: err}
		}()

		select {
		case msg := <-ch:
			return msg
		case <-ctx.Done():
			return updateInstallMsg{done: true, err: ctx.Err()}
		}
	}
}

func waitUpdateInstallCmd(ch chan updateInstallMsg) tea.Cmd {
	return func() tea.Msg {
		if ch == nil {
			return nil
		}
		return <-ch
	}
}

type taskTickMsg struct{}
type shellFinishedMsg struct {
	action sshnet.EscapeResult
	err    error
}

func taskCenterTickCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
		return taskTickMsg{}
	})
}

func (m Model) runShellCmd() tea.Cmd {
	return tea.Exec(&shellExecCommand{client: m.ssh, title: m.shellFrameTitle(), width: m.width}, func(err error) tea.Msg {
		if finished, ok := err.(shellExitError); ok {
			return shellFinishedMsg{action: finished.action}
		}
		return shellFinishedMsg{err: err}
	})
}

func (m Model) reconnectCmd() tea.Cmd {
	return func() tea.Msg {
		m.tasks.CancelAll()
		waitCtx, waitCancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = m.tasks.Wait(waitCtx)
		waitCancel()
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := m.ssh.Reconnect(ctx); err != nil {
			var unknown *sshnet.UnknownHostKeyError
			if errors.As(err, &unknown) {
				return hostKeyPromptMsg{err: unknown, server: m.activeServer, action: hostKeyActionReconnect}
			}
			return errMsg{err}
		}
		return shellConnectedMsg{client: m.ssh, server: m.activeServer}
	}
}

func (m *Model) shutdown() {
	if m.tasks != nil {
		m.tasks.CancelAll()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = m.tasks.Wait(ctx)
		cancel()
	}
	if m.ssh != nil {
		_ = m.ssh.Close()
		m.ssh = nil
	}
}

type shellExecCommand struct {
	client *sshnet.Client
	title  string
	width  int
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

func (c *shellExecCommand) SetStdin(r io.Reader)  { c.stdin = r }
func (c *shellExecCommand) SetStdout(w io.Writer) { c.stdout = w }
func (c *shellExecCommand) SetStderr(w io.Writer) { c.stderr = w }

func (c *shellExecCommand) Run() error {
	if c.client == nil {
		return fmt.Errorf("ssh client is not connected")
	}
	in, ok := c.stdin.(*os.File)
	if !ok {
		return fmt.Errorf("shell stdin is not a terminal")
	}
	out, ok := c.stdout.(*os.File)
	if !ok {
		return fmt.Errorf("shell stdout is not a terminal")
	}
	errOut, ok := c.stderr.(*os.File)
	if !ok {
		errOut = os.Stderr
	}
	c.writeShellHeader(out)
	var action sshnet.EscapeResult
	err := c.client.RunInteractiveShell(context.Background(), in, out, errOut, func(res sshnet.EscapeResult) {
		if res.Command != "help" && res.Command != "send" && !res.Unknown {
			action = res
		}
	})
	if action.Local {
		return shellExitError{action: action}
	}
	return err
}

func (c *shellExecCommand) writeShellHeader(out *os.File) {
	width := c.width
	if width <= 0 {
		width = 88
	}
	if width < 60 {
		width = 60
	}
	_, _ = fmt.Fprintln(out, shellTopBorder(width, c.title))
}

type shellExitError struct {
	action sshnet.EscapeResult
}

func (e shellExitError) Error() string {
	return "shell exited by local command: " + e.action.Command
}

func (m Model) filteredServers() []config.Server {
	if m.filter == "" {
		return m.config.Servers
	}
	var out []config.Server
	f := strings.ToLower(m.filter)
	for _, srv := range m.config.Servers {
		haystack := strings.ToLower(strings.Join([]string{
			srv.ID,
			srv.Name,
			srv.Env,
			srv.Host,
			srv.User,
			srv.AuthType,
			srv.Desc,
			strings.Join(srv.Tags, ","),
		}, " "))
		if strings.Contains(haystack, f) {
			out = append(out, srv)
		}
	}
	return out
}

func (m *Model) refreshLocalFiles() {
	entries, err := os.ReadDir(m.localDir)
	if err != nil {
		m.localFiles = nil
		m.err = err.Error()
		return
	}
	items := []fileItem{{Name: "..", Path: filepath.Dir(m.localDir), Mode: os.ModeDir | 0o755, IsDir: true}}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		items = append(items, fileItem{
			Name:    entry.Name(),
			Path:    filepath.Join(m.localDir, entry.Name()),
			Mode:    info.Mode(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
			IsDir:   entry.IsDir(),
		})
	}
	sort.Slice(items[1:], func(i, j int) bool {
		a := items[i+1]
		b := items[j+1]
		if a.IsDir != b.IsDir {
			return a.IsDir
		}
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})
	m.localFiles = items
}

func (m Model) currentFiles() []fileItem {
	if m.config.Settings.DefaultViewMode == config.ViewSingle {
		return filteredFileItems(m.remoteFiles, m.remoteFileFilter)
	}
	if m.activePane == 0 {
		return filteredFileItems(m.localFiles, m.localFileFilter)
	}
	return filteredFileItems(m.remoteFiles, m.remoteFileFilter)
}

func (m Model) currentFileCursor() int {
	if m.config.Settings.DefaultViewMode == config.ViewSingle || m.activePane == 1 {
		return m.remoteCursor
	}
	return m.localCursor
}

func (m *Model) setCurrentFileCursor(cursor int) {
	if m.config.Settings.DefaultViewMode == config.ViewSingle || m.activePane == 1 {
		m.remoteCursor = clampCursor(cursor, len(filteredFileItems(m.remoteFiles, m.remoteFileFilter)))
		return
	}
	m.localCursor = clampCursor(cursor, len(filteredFileItems(m.localFiles, m.localFileFilter)))
}

func (m *Model) toggleSelected() {
	files := m.currentFiles()
	cursor := m.currentFileCursor()
	if cursor < 0 || cursor >= len(files) || files[cursor].Name == ".." {
		return
	}
	targetPath := files[cursor].Path
	if m.config.Settings.DefaultViewMode != config.ViewSingle && m.activePane == 0 {
		for i := range m.localFiles {
			if m.localFiles[i].Path == targetPath {
				m.localFiles[i].Selected = !m.localFiles[i].Selected
				return
			}
		}
	}
	for i := range m.remoteFiles {
		if m.remoteFiles[i].Path == targetPath {
			m.remoteFiles[i].Selected = !m.remoteFiles[i].Selected
			return
		}
	}
}

func (m *Model) selectAll(selected bool) {
	visible := m.currentFiles()
	visiblePaths := map[string]bool{}
	for _, item := range visible {
		if item.Name != ".." {
			visiblePaths[item.Path] = true
		}
	}
	target := &m.localFiles
	if m.config.Settings.DefaultViewMode == config.ViewSingle || m.activePane == 1 {
		target = &m.remoteFiles
	}
	for i := range *target {
		if visiblePaths[(*target)[i].Path] {
			(*target)[i].Selected = selected
		}
	}
}

func (m Model) currentFileFilter() string {
	if m.config.Settings.DefaultViewMode == config.ViewSingle || m.activePane == 1 {
		return m.remoteFileFilter
	}
	return m.localFileFilter
}

func (m *Model) applyCurrentFileFilter(filter string) {
	filter = strings.TrimSpace(filter)
	if m.config.Settings.DefaultViewMode == config.ViewSingle || m.activePane == 1 {
		m.remoteFileFilter = filter
		m.remoteCursor = 0
		return
	}
	m.localFileFilter = filter
	m.localCursor = 0
}

func filteredFileItems(items []fileItem, filter string) []fileItem {
	filter = strings.ToLower(strings.TrimSpace(filter))
	if filter == "" {
		return items
	}
	out := make([]fileItem, 0, len(items))
	for _, item := range items {
		if item.Name == ".." || strings.Contains(strings.ToLower(item.Name), filter) {
			out = append(out, item)
		}
	}
	return out
}

func (m *Model) enterFile() {
	if m.activePane != 0 || m.localCursor >= len(m.localFiles) {
		return
	}
	item := m.localFiles[m.localCursor]
	if !item.IsDir {
		return
	}
	m.localDir = item.Path
	m.localCursor = 0
	m.refreshLocalFiles()
}

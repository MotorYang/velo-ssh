# VeloSSH AISpec

## 0. Purpose

This document is the implementation specification for AI coding agents working on VeloSSH (`vssh`). It translates `docs/prd.md` into executable engineering tasks, module boundaries, data contracts, behavioral rules, and acceptance criteria.

When the PRD and this AISpec conflict, prefer this AISpec for implementation details and prefer the PRD for product intent.

## 1. Product Summary

VeloSSH is a single-binary Go CLI/TUI application for managing SSH servers and transferring files through SFTP. The core experience has four UI areas:

1. Server list: browse, search, add, edit, delete, and connect to servers.
2. File manager: dual-pane local/remote file browser with upload and download tasks.
3. Task center: background transfer status, pause/resume, and cancellation.
4. Settings center: global settings, security settings, and UI preferences.

The SSH shell must not use `Ctrl` hotkeys for switching back into local TUI views. It must use line-based local escape commands:

```text
:vssh files
:vssh tasks
:vssh settings
:vssh back
:vssh reconnect
:vssh quit
:vssh help
:vssh send <text>
```

## 2. Implementation Principles

Use Go 1.22+ and keep the binary self-contained.

Use these dependencies unless there is a strong reason not to:

```text
github.com/spf13/cobra
github.com/charmbracelet/bubbletea
github.com/charmbracelet/bubbles
github.com/charmbracelet/lipgloss
github.com/mattn/go-runewidth
golang.org/x/crypto/ssh
github.com/pkg/sftp
github.com/zalando/go-keyring
```

Implementation rules:

1. Keep transport, storage, and TUI state separate.
2. Do not store passwords or key passphrases in JSON config files.
3. Do not overwrite remote files directly; upload to a temporary file and rename after verification.
4. Do not read large files fully into memory.
5. Do not use `len(string)` for terminal layout width.
6. Do not intercept arbitrary `:` input in SSH shell mode. Only intercept complete lines beginning with `:vssh ` or exactly `:vssh`.
7. Keep v1.0 focused. Route advanced items to explicit stubs or future milestones instead of half-implementing them.

## 3. Milestones

### 3.1 MVP v1.0

Implement these features first:

1. Cobra CLI root command and `vssh config`.
2. Local config storage under `~/.config/vssh/`.
3. Server CRUD in TUI.
4. Keyring-backed password/passphrase storage.
5. Server list with search and environment filtering.
6. SSH interactive shell connection.
7. Local escape command parser for `:vssh <command>`.
8. File manager with local pane, remote pane, navigation, select, upload, download, rename.
9. Background task center with progress, cancel, and best-effort pause/resume.
10. Remote atomic upload through temporary files and final rename.
11. KeepAlive for idle SSH clients.
12. PTY resize propagation.
13. ASCII fallback mode.

### 3.2 v1.1

Implement after MVP:

1. Folder tar.gz streaming optimization.
2. `.vsshignore` support.
3. Draft retry center for failed remote edit uploads.
4. Export/import.
5. Hash compare and text diff.

### 3.3 Later

Treat these as non-MVP:

1. Cross-server direct transfer.
2. Multi-part parallel resume with chunk manifest.
3. Rich theme marketplace.

## 4. Repository Layout

Target structure:

```text
velo-ssh/
├── cmd/
│   ├── root.go
│   ├── config.go
│   ├── export.go
│   └── import.go
├── internal/
│   ├── app/
│   │   ├── state.go
│   │   └── events.go
│   ├── config/
│   │   ├── model.go
│   │   ├── storage.go
│   │   ├── keyring.go
│   │   └── drafts.go
│   ├── sshnet/
│   │   ├── client.go
│   │   ├── hostkey.go
│   │   ├── keepalive.go
│   │   ├── shell.go
│   │   ├── escape.go
│   │   └── sftp_worker.go
│   ├── transfer/
│   │   ├── task.go
│   │   ├── manager.go
│   │   ├── atomic_upload.go
│   │   └── ignore.go
│   ├── tui/
│   │   ├── model.go
│   │   ├── keys.go
│   │   ├── styles.go
│   │   ├── view_server_list.go
│   │   ├── view_file_manager.go
│   │   ├── view_task_center.go
│   │   ├── view_settings.go
│   │   └── view_modal.go
│   └── term/
│       ├── width.go
│       └── capabilities.go
├── docs/
│   ├── prd.md
│   └── aispec.md
├── go.mod
└── main.go
```

Use `internal/` to keep implementation packages private.

## 5. Data Models

### 5.1 Config File

Path:

```text
~/.config/vssh/config.json
```

Schema:

```json
{
  "version": 1,
  "settings": {
    "defaultViewMode": "split",
    "asciiFallback": "auto",
    "fallbackRemotePath": "/tmp",
    "draftTTLDays": 30,
    "transferConcurrency": 4,
    "keepAliveSeconds": 20,
    "theme": "default",
    "confirmOverwrite": true,
    "knownHostsPolicy": "strict"
  },
  "servers": [
    {
      "id": "prod-web-01",
      "name": "Prod-Web-01",
      "env": "prod",
      "host": "10.0.4.21",
      "port": 22,
      "user": "root",
      "authType": "key",
      "keyPath": "~/.ssh/id_ed25519",
      "passwordRef": "",
      "passphraseRef": "vssh:prod-web-01:passphrase",
      "desc": "Production Nginx frontend node",
      "tags": ["web", "nginx"],
      "defaultRemotePath": "/var/www/html",
      "createdAt": "2026-07-05T00:00:00+08:00",
      "updatedAt": "2026-07-05T00:00:00+08:00"
    }
  ]
}
```

Rules:

1. `id` must be stable and unique.
2. `authType` values: `key`, `password`, `agent`.
3. `passwordRef` and `passphraseRef` are keyring lookup keys, not secrets.
4. Write config atomically: write temp file, fsync when practical, rename.
5. Create the config directory with mode `0700`.
6. Create config files with mode `0600`.

### 5.2 Draft File

Path:

```text
~/.config/vssh/drafts.json
```

Schema:

```json
{
  "version": 1,
  "drafts": [
    {
      "id": "draft-uuid",
      "serverId": "prod-web-01",
      "remotePath": "/etc/nginx/nginx.conf",
      "localPath": "/Users/me/.config/vssh/drafts/prod-web-01/nginx.conf",
      "baseRemoteSize": 1234,
      "baseRemoteMTime": "2026-07-05T00:00:00+08:00",
      "localSHA256": "hex",
      "status": "pending",
      "createdAt": "2026-07-05T00:00:00+08:00",
      "updatedAt": "2026-07-05T00:00:00+08:00"
    }
  ]
}
```

Statuses:

```text
pending
syncing
failed
resolved
expired
```

## 6. CLI Contract

### 6.1 Root

```text
vssh
```

Open the server list TUI.

### 6.2 Config

```text
vssh config
```

Open the settings TUI directly.

### 6.3 Connect

```text
vssh connect <server-id-or-name>
```

Connect directly to an SSH shell. If multiple names match, show an error with candidates.

### 6.4 Export and Import

v1.1 commands:

```text
vssh export --output backup.json [--include-secrets]
vssh import backup.json
```

For MVP, commands may exist but should return a clear "not implemented in v1.0" error if not built.

## 7. App States

Define app states:

```go
type AppState int

const (
	StateServerList AppState = iota
	StateServerForm
	StateShell
	StateFileManager
	StateConfirmModal
	StateTaskCenter
	StateSettingsCenter
	StateHelp
)
```

Global transitions:

```text
ServerList + Enter -> Shell
ServerList + f -> FileManager
ServerList + Shift+S -> SettingsCenter
ServerList + a/e/d -> ServerForm or ConfirmModal
Shell + ":vssh files" -> FileManager
Shell + ":vssh tasks" -> TaskCenter
Shell + ":vssh settings" -> SettingsCenter
Shell + ":vssh back" -> ServerList
Shell + ":vssh reconnect" -> Shell with new SSH session
Shell + ":vssh quit" -> ServerList or process exit, depending entry path
FileManager + t -> TaskCenter
FileManager + Esc -> previous state
SettingsCenter + Esc -> previous state
TaskCenter + Esc/t -> previous state
```

Track `previousState` for modal/task/settings returns.

## 8. SSH Shell Escape Command Contract

### 8.1 Command Grammar

Recognize only complete input lines after Enter:

```text
:vssh
:vssh <command>
:vssh send <text>
```

Supported commands:

```text
files
tasks
settings
back
reconnect
quit
help
send <text>
```

Parsing rules:

1. Match only when line starts at local line buffer position 0.
2. Match `:vssh` exactly or `:vssh ` followed by arguments.
3. Preserve ordinary `:` input.
4. Preserve all input when the line does not match `:vssh`.
5. `:vssh send <text>` sends `<text>` plus newline to the remote session.
6. Unknown commands show local help and do not send the line to remote.
7. Empty `:vssh` shows local help and does not send the line to remote.

### 8.2 Stream Wrapper Behavior

The shell wrapper sits between local terminal input and remote SSH stdin.

Minimum viable behavior:

1. Put local terminal in raw mode while shell is active.
2. Maintain a small local line buffer.
3. Append printable bytes to the line buffer.
4. Remove one rune on Backspace/Delete.
5. On Enter, inspect the buffered line.
6. If it is a local escape command, execute local action and do not forward that line.
7. Otherwise, forward buffered bytes and newline to remote.

Important limitation:

This line-buffering approach can affect full-screen remote programs if all bytes are delayed until Enter. Prefer an implementation that forwards bytes immediately but still detects `:vssh` at a fresh prompt line. If immediate forwarding is implemented, suppressing already-forwarded escape text is hard. Therefore MVP may use an explicit local command capture mode only at shell prompt boundaries if reliable prompt detection exists; otherwise document the limitation clearly and add tests around it.

Recommended robust approach:

1. Forward normal bytes immediately.
2. Detect a candidate only when the local line buffer begins with `:`.
3. Temporarily buffer bytes while the prefix may become `:vssh`.
4. If the prefix diverges from `:vssh`, flush buffered bytes to remote immediately.
5. If the line completes as `:vssh ...`, execute locally and do not flush.

Current MVP implementation:

1. Normal bytes are forwarded to the remote stdin immediately.
2. At local line start, bytes beginning with `:` are temporarily buffered only while they may still become `:vssh`.
3. Ordinary remote commands such as `:wq`, `:vsshx files`, or ` echo :vssh files` are flushed to remote.
4. Complete local commands such as `:vssh files` are intercepted and are not sent to remote.
5. `:vssh send <text>` force-sends `<text>` plus newline to remote.

This keeps normal shell typing responsive and limits interception to the explicit escape prefix.

## 9. SSH and SFTP Contract

### 9.1 SSH Client

`internal/sshnet.Client` should own:

```text
server config
ssh.Client
sftp.Client
active shell session
keepalive goroutine
connection state
```

Required methods:

```go
Connect(ctx context.Context, server config.Server) error
OpenShell(ctx context.Context, size PtySize) (*Shell, error)
OpenSFTP(ctx context.Context) (*sftp.Client, error)
Reconnect(ctx context.Context) error
Close() error
WindowChange(height, width int) error
```

### 9.2 Host Key Policy

MVP must not silently accept changed host keys.

Policies:

```text
strict: require known_hosts match
ask: first connection asks user to trust fingerprint
insecure: allow any host key, visibly marked as insecure
```

Default: `strict` if known_hosts exists; otherwise `ask`.

### 9.3 KeepAlive

Send SSH global keepalive request every `settings.keepAliveSeconds`.

Behavior:

1. Stop keepalive on close.
2. Mark connection stale after repeated failures.
3. Surface stale state in TUI.

## 10. Transfer Contract

### 10.1 Task Model

```go
type TransferDirection string

const (
	Upload TransferDirection = "upload"
	Download TransferDirection = "download"
)

type TaskStatus string

const (
	TaskQueued    TaskStatus = "queued"
	TaskRunning   TaskStatus = "running"
	TaskPaused    TaskStatus = "paused"
	TaskSucceeded TaskStatus = "succeeded"
	TaskFailed    TaskStatus = "failed"
	TaskCanceled  TaskStatus = "canceled"
)
```

Task fields:

```text
id
serverId
direction
sourcePath
targetPath
bytesTotal
bytesDone
status
error
startedAt
updatedAt
```

### 10.2 Atomic Upload

For remote upload:

1. Resolve target directory and base name.
2. Create temp file in the same remote directory:

```text
.<target>.vssh.tmp.<task-id>
```

3. Upload content to temp path.
4. Flush/close remote file.
5. Verify size. Use SHA-256 for optional strong verification.
6. If overwriting an existing file, capture existing file mode before upload.
7. Apply mode to temp file where possible.
8. Rename temp path to target path.
9. Clean temp file on cancellation or failure when safe.

Do not use shell `mv` when SFTP `Rename` is available. Use protocol-native rename first.

### 10.3 Download

For local download:

1. Download to a local temp file in the target directory.
2. Verify size.
3. Rename local temp file to final file.
4. Preserve mode and mtime where practical.

### 10.4 Pause and Resume

MVP:

1. Pause means stop scheduling new reads/writes and keep current task state.
2. Resume continues from the current byte offset only for sequential transfers.
3. If exact resume is unsafe, restart the temp file transfer and show status.

v1.1:

1. Add chunk manifest.
2. Support parallel `WriteAt`.
3. Verify every chunk.

## 11. TUI Contract

### 11.1 Layout

Use Bubble Tea MVU. Keep all state changes in `Update`. Keep rendering pure in `View`.

Minimum terminal size:

```text
80 columns x 24 rows
```

If smaller, show a compact message asking the user to resize.

### 11.2 Width and Unicode

Use `go-runewidth` or Lipgloss width helpers for all layout calculations.

ASCII fallback:

1. Replace rounded borders with `+`, `-`, `|`.
2. Replace emoji indicators with text labels: `[UP]`, `[DOWN]`, `[OK]`, `[ERR]`.
3. Avoid ambiguous-width icons in Linux TTY or when `asciiFallback` is enabled.

### 11.3 Key Bindings

Server list:

```text
up/down or j/k: move
/: search
Enter: connect shell
f: file manager
Shift+S: settings
a/e/d: add/edit/delete
q: quit
```

File manager:

```text
Tab: switch pane
Space: select
a: select all
c: clear selection in MVP; cross-server copy is v1.1/later
u: upload
d: download
r: rename
t: task center
b: split/single view
=: compare in v1.1
Esc: back
```

Settings:

```text
Tab: switch side or field group
j/k or up/down: move
Space: toggle
Enter: edit or confirm
s: save
Esc: cancel
```

Avoid `Ctrl+S` because terminal flow control can freeze traditional TTYs.

## 12. Error Handling

Every user-facing error must include:

1. What failed.
2. The target server/path if applicable.
3. The short reason.
4. A recoverable next action where possible.

Common cases:

```text
auth failed
host key mismatch
connection timeout
permission denied
disk full
file exists
remote path missing
local path missing
keyring unavailable
terminal too small
transfer canceled
connection stale
```

Do not panic on expected runtime failures.

## 13. Security Requirements

1. Never log passwords, passphrases, private key content, or keyring values.
2. Do not export secrets unless the user explicitly opts in.
3. Prefer OS keyring for secrets.
4. If keyring is unavailable, ask before falling back to prompt-only credentials.
5. Mark insecure host key policy clearly in UI.
6. Use `known_hosts` verification by default.
7. Do not follow symlinks recursively during folder upload unless explicitly enabled.

## 14. Testing Plan

### 14.1 Unit Tests

Required:

```text
config load/save/migration
server CRUD validation
keyring ref generation
escape command parser
ASCII fallback detection
width calculation with CJK and emoji
transfer task state transitions
atomic temp path generation
.vsshignore parser when implemented
```

### 14.2 Integration Tests

Use a local test SSH/SFTP server or container.

Required:

```text
password auth connection
key auth connection
known_hosts mismatch rejection
upload creates temp then final file
failed upload does not destroy existing target
download creates temp then final file
keepalive stops after client close
PTY resize calls WindowChange
```

### 14.3 TUI Tests

Use Bubble Tea model tests where possible.

Required:

```text
server list navigation
search filtering
file pane focus switching
selection behavior
task center open/close
settings save/cancel
small terminal fallback
```

## 15. Acceptance Criteria

MVP is acceptable when:

1. `go test ./...` passes.
2. `go vet ./...` passes.
3. `vssh` opens server list without a config file and offers first-run setup.
4. A server can be added, edited, deleted, and persisted.
5. Secrets are stored in keyring, not config JSON.
6. Connecting to a server opens an interactive shell.
7. `:vssh files` from shell switches to file manager and is not sent to remote.
8. `:vssh send :vssh files` sends `:vssh files` to remote.
9. `:vssh quit` disconnects the current SSH session cleanly.
10. Uploading a file never truncates an existing remote target on connection failure.
11. Downloading a file never leaves a corrupt final local file after failure.
12. Task center shows progress and final status.
13. Window resizing updates TUI layout and remote PTY size.
14. Linux TTY or forced ASCII mode does not render emoji or rounded borders.
15. The app exits cleanly without leaked goroutines in normal flows.

## 16. Implementation Order for AI Agents

Follow this order unless the user explicitly asks otherwise:

1. Initialize Go module and dependency set.
2. Create config models and atomic storage.
3. Add Cobra root and config commands.
4. Build minimal Bubble Tea app shell with app states.
5. Implement server list view with CRUD backed by config storage.
6. Implement keyring adapter and auth config validation.
7. Implement SSH client connection, known_hosts policy, shell, keepalive, and PTY resize.
8. Implement escape command parser with tests.
9. Integrate shell escape commands with TUI state transitions.
10. Implement SFTP file listing and file manager navigation.
11. Implement upload/download task manager.
12. Implement atomic upload and local temp download.
13. Implement task center.
14. Implement settings center.
15. Add ASCII fallback and width-safe rendering.
16. Add integration tests and polish errors.

At each step:

1. Keep the build passing.
2. Add focused tests for new behavior.
3. Avoid implementing v1.1 features unless the MVP path requires an interface stub.

## 17. Known PRD Corrections to Apply During Implementation

1. Use `tar -xzf .tmp_pack.tar.gz -C /target/dir && rm .tmp_pack.tar.gz` when implementing tar extraction. Do not use the PRD's older `tar -fvxz` spelling.
2. Prefer SHA-256 for strong file verification. MD5 may only be used as a fast non-security checksum if clearly labeled.
3. Use `s` or `Enter` for settings save, not `Ctrl+S`.
4. Treat cross-server direct transfer as later work, not v1.0.
5. Treat parallel chunk resume as later work unless a chunk manifest is implemented.

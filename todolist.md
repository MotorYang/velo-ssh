# VeloSSH MVP Todo List

This todo list is based on `docs/aispec.md` MVP v1.0 and the current implementation status.

## Done

- [x] Initialize Go module and target package layout.
- [x] Add Cobra CLI root command.
- [x] Implement `vssh`.
- [x] Implement `vssh config`.
- [x] Implement `vssh connect <server-id-or-name>`.
- [x] Add `vssh export` placeholder with clear v1.0 not implemented error.
- [x] Add `vssh import` placeholder with clear v1.0 not implemented error.
- [x] Implement config model.
- [x] Store config at `~/.config/vssh/config.json`.
- [x] Store drafts at `~/.config/vssh/drafts.json`.
- [x] Create config directory with `0700`.
- [x] Create config files with `0600`.
- [x] Write config atomically with temporary file and rename.
- [x] Implement keyring adapter.
- [x] Store `passwordRef` / `passphraseRef` in JSON instead of secret values.
- [x] Implement server list TUI.
- [x] Implement server search/filter.
- [x] Implement server add/edit/delete flow.
- [x] Implement server connection from list.
- [x] Implement settings center entry point.
- [x] Implement SSH auth with password, key, and agent.
- [x] Implement interactive SSH shell with PTY.
- [x] Implement PTY resize propagation through `WindowChange`.
- [x] Implement KeepAlive for idle SSH clients.
- [x] Implement `:vssh <command>` parser.
- [x] Support `:vssh files`.
- [x] Support `:vssh tasks`.
- [x] Support `:vssh settings`.
- [x] Support `:vssh back`.
- [x] Support `:vssh reconnect`.
- [x] Support `:vssh quit`.
- [x] Support `:vssh help`.
- [x] Support `:vssh send <text>`.
- [x] Implement file manager with default remote single-pane view.
- [x] Implement optional local pane toggle.
- [x] Implement local/remote dual-pane layout.
- [x] Implement independent local and remote scrolling.
- [x] Implement resize-aware file manager layout.
- [x] Implement directory enter and parent navigation.
- [x] Implement file selection, select all, and clear selection.
- [x] Implement file rename.
- [x] Implement file upload.
- [x] Implement file download.
- [x] Display file permission mode.
- [x] Display human-readable file size.
- [x] Support optional modified-time display with `m`.
- [x] Keep file manager shortcut hints at the bottom.
- [x] Render shortcut footer as two lines with top border.
- [x] Fix file manager column alignment and long-name truncation.
- [x] Implement transfer task model.
- [x] Implement remote atomic upload with temporary file and final rename.
- [x] Verify uploaded size before final rename.
- [x] Implement local atomic download with temporary file and final rename.
- [x] Verify downloaded size before final rename.
- [x] Implement task cancellation.
- [x] Implement thread-safe task snapshots for TUI rendering.
- [x] Implement task center view.
- [x] Display task direction, status, progress, size, paths, and error.
- [x] Support task center selection with `j/k`.
- [x] Support cancel selected task with `x`.
- [x] Auto-refresh task center progress.
- [x] Implement minimum terminal size fallback at `80x24`.
- [x] Use `lipgloss` / `go-runewidth` for visual width calculations.
- [x] Implement basic ASCII fallback detection.
- [x] Add unit tests for config storage.
- [x] Add unit tests for server CRUD.
- [x] Add unit tests for escape parser.
- [x] Add unit tests for transfer task state.
- [x] Add unit tests for file manager layout and selection.
- [x] Add unit tests for task center interaction.
- [x] Display current VeloSSH version in the server manager.
- [x] Check GitHub Releases for newer versions.
- [x] Support disabling update checks in settings.
- [x] Support skipping a specific update version.
- [x] Download and install matching release assets automatically.
- [x] Show update download/install progress in a modal.
- [x] Add GitHub Actions release workflow for prod branch platform artifacts.
- [x] Generate release notes from `CHANGELOG.md`.
- [x] Add language setting with English and Simplified Chinese options.
- [x] Localize manager, settings, forms, modals, file manager, task center, and `:vssh help`.
- [x] Return to server list when a remote SSH shell exits with `exit`.
- [x] Generate server IDs automatically in the server form.
- [x] Use option controls for auth type and related form settings.
- [x] Store password and passphrase values in the secret store instead of config JSON.
- [x] Group server list entries by tag, with untagged servers under `default`.
- [x] Show environment labels in the server list.
- [x] Support cloning servers from the manager.
- [x] Warn before leaving dirty server forms.
- [x] Implement real `vssh export` for configuration backups.
- [x] Implement real `vssh import` for configuration backups.
- [x] Support explicit secret export/import with `--include-secrets`.
- [x] Implement `.vsshignore` filtering for recursive local folder uploads.
- [x] Implement SHA-256 hash compare for one local file and one remote file.
- [x] Implement small text file diff output for differing compared files.
- [x] Implement tar.gz archive optimization for selected local folder uploads.
- [x] Implement draft retry center inside Task Center.
- [x] Prune expired drafts using `draftTTLDays`.
- [x] Implement cross-server remote file copy through `vssh copy`.
- [x] Implement multipart parallel upload resume with local chunk manifests.

## High Priority

- [x] Complete settings center as an editable form.
- [x] Edit `defaultViewMode` in settings center.
- [x] Edit `asciiFallback` in settings center.
- [x] Edit `fallbackRemotePath` in settings center.
- [x] Edit `draftTTLDays` in settings center.
- [x] Edit `transferConcurrency` in settings center.
- [x] Edit `keepAliveSeconds` in settings center.
- [x] Edit `theme` in settings center.
- [x] Edit `confirmOverwrite` in settings center.
- [x] Edit `knownHostsPolicy` in settings center.
- [x] Save settings with `s` or `Enter`.
- [x] Avoid `Ctrl+S` as a save shortcut.
- [x] Implement transfer queue scheduling based on `transferConcurrency`.
- [x] Implement task pause.
- [x] Implement task resume.
- [x] For unsafe resume cases, restart temp-file transfer and show a clear status.
- [x] Refresh file manager panes after transfer success.
- [x] Add overwrite confirmation before upload/download replaces an existing target.
- [x] Respect `confirmOverwrite`.
- [x] Improve known_hosts `ask` policy with an interactive confirmation flow.
- [x] Persist newly accepted host keys when policy allows.
- [x] Ensure host key mismatch is rejected and clearly explained.

## File Manager

- [x] Add create local directory.
- [x] Add create remote directory.
- [x] Add delete local file.
- [x] Add delete remote file.
- [x] Add delete local directory.
- [x] Add delete remote directory.
- [x] Add confirmation modal for destructive file operations.
- [x] Add same-pane local copy/paste.
- [x] Add same-pane local move.
- [x] Add same-pane remote copy/paste.
- [x] Add same-pane remote move.
- [x] Reject cross-pane paste with upload/download guidance.
- [x] Add recursive folder upload using per-file atomic transfer tasks.
- [x] Add recursive folder download using per-file atomic transfer tasks.
- [x] Add clear error when folder upload is attempted in MVP.
- [x] Add clear error when folder download is attempted in MVP.
- [x] Keep existing target files intact on failed upload.
- [x] Keep existing target files intact on failed download.
- [x] Handle permission denied errors with action/path/recovery wording.
- [x] Handle disk full errors with action/path/recovery wording.
- [x] Handle file exists errors with action/path/recovery wording.
- [x] Handle missing path errors with action/path/recovery wording.

## SSH Shell

- [x] Document current line-buffered shell escape limitation.
- [x] Improve escape command handling without degrading full-screen remote TUI programs.
- [x] Add integration test that `:vssh files` is not sent to remote.
- [x] Add integration test that `:vssh send :vssh files` is sent to remote.
- [x] Ensure `:vssh quit` always returns to server manager when launched from TUI.
- [x] Ensure reconnect rebuilds current SSH/SFTP channels.
- [x] Ensure shell close stops KeepAlive goroutine.
- [x] Preserve the existing remote shell session when opening and closing FileManager.
- [x] Buffer detached shell output and flush it when returning from FileManager.

## Error Experience

- [x] Define shared error formatting helper.
- [x] Include failed action in user-facing errors.
- [x] Include target server/path in user-facing errors.
- [x] Include underlying reason in user-facing errors.
- [x] Include recovery action when practical.
- [x] Improve authentication failure error text.
- [x] Improve host key mismatch error text.
- [x] Improve connection timeout error text.
- [x] Improve permission denied error text.
- [x] Improve keyring unavailable error text.
- [x] Improve terminal too small error text.
- [x] Improve task canceled error text.
- [x] Improve stale connection error text.

## Lifecycle And Reliability

- [x] Audit goroutine lifecycle for shell sessions.
- [x] Audit goroutine lifecycle for SFTP clients.
- [x] Audit goroutine lifecycle for KeepAlive.
- [x] Audit goroutine lifecycle for transfer tasks.
- [x] Ensure disconnect cancels active shell resources.
- [x] Ensure app quit cancels active transfer resources or reports continuing background tasks.
- [x] Ensure reconnect does not leak old sessions.
- [x] Add race tests for shell and KeepAlive lifecycle.
- [x] Add race tests for transfer progress updates.

## Integration Tests

- [x] Add local SSH/SFTP test server fixture.
- [x] Test password auth against local SSH/SFTP server.
- [x] Test key auth against local SSH/SFTP server.
- [x] Test known_hosts mismatch rejection.
- [x] Test upload temp file becomes final file only after success.
- [x] Test failed upload does not truncate existing remote target.
- [x] Test failed download does not create corrupt final local file.
- [x] Test KeepAlive stops after client close.
- [x] Test resize triggers PTY `WindowChange`.

## Current Verification Commands

```bash
go test ./...
go vet ./...
go test -race ./internal/transfer ./internal/tui ./internal/sshnet
go build ./...
go build -o ./velo-ssh .
```

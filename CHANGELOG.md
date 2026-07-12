# Changelog

- Added support cancelling in-progress updates
- Reworked install scripts into one-line Release binary installers for macOS, Linux, and Windows.
- Fixed File Manager split-pane rendering so filters, column headers, and shortcut footer remain visible at minimum terminal sizes.
- Fixed File Manager Enter behavior for regular files: local files open in the configured editor and remote files start the remote edit flow.
- Fixed remote edit sync and upload overwrite finalization by using OpenSSH POSIX rename when replacing existing remote files.

## v1.1.1.26070701

- Fixed folder archive uploads to run as visible background tasks.
- Fixed transfer completion refresh for file manager panes after uploads/downloads.
- Fixed compare flow to show download progress, support cancellation, and return to File Manager.
- Improved cleanup of temporary `.vssh.tmp` paths after successful transfers.
- Fixed SSH Manager navigation so `j/k` and arrow keys follow the grouped visible server order.
- Fixed SSH Shell pasted commands with URL colons so bracketed paste input is forwarded in order.

## v1.1.0.26070601

- Added real `vssh export` and `vssh import` configuration backup commands with optional secret backup.
- Added `.vsshignore` filtering for recursive local folder uploads.
- Added file manager SHA-256 compare and small text diff for selected local/remote files.
- Added tar.gz archive optimization for selected local folder uploads.
- Added draft retry center support in Task Center with retry/resolve actions and TTL pruning.
- Added `vssh copy <source-server>:<path> <target-server>:<path>` for cross-server remote file transfer.
- Added multipart parallel upload resume with local chunk manifests for large files.
- Fixed multipart resume to reuse a stable remote temporary file path across retries.
- Added remote file edit flow with local drafts and automatic failed-upload retry records.
- Added asynchronous server list online checks with latency display.
- Added default `.vsshignore` exclusions for `.DS_Store`, `Thumbs.db`, and `.git/`.
- Added AES-256-GCM encrypted backup export/import with passphrase support.

## v1.0.0.26070601

- Added VeloSSH version display in the server manager.
- Added update checking with update, cancel, and skip-this-version choices.
- Added automatic update download and installation with an in-app progress modal.
- Added a setting to disable update checks.
- Added release workflow support for publishing platform builds from changelog release notes.
- Improved SSH shell exit handling so `exit` returns to the server list.

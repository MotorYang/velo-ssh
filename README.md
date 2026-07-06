<div align="center">

# VeloSSH (vssh)

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/motoryang/velo-ssh?logo=github)](https://github.com/motoryang/velo-ssh/releases)
[![Website](https://img.shields.io/badge/Website-000000?logo=githubpages)](https://motoryang.github.io/velo-ssh/)

**A lightweight TUI-based SSH manager and dual-pane SFTP file transfer tool**

Author: [motoryang](https://github.com/motoryang) | [View Releases](https://github.com/motoryang/velo-ssh/releases)

**English** | [简体中文](README_CN.md)

</div>

---

## Features

### Server Management
- Environment-based grouping (dev / prod / test, etc.) with search and filtering
- Async online status detection with latency display
- Multiple authentication methods: password, SSH key, passphrase
- System Keyring integration — credentials securely stored in OS-native keystore
- `vssh export` / `vssh import` for configuration backup, supports AES-256-GCM encrypted export

### Dual-Pane File Manager
- Local ↔ Remote split view, toggleable to single-pane mode
- Multi-file selection, batch upload / download
- Inline rename, new directory, paste, move, delete
- File search, SHA-256 checksum verification and text-level diff comparison
- Drag-and-drop upload and download support
- `.vsshignore` filtering (skips `.DS_Store`, `Thumbs.db`, `.git/`, etc.)
- Smart folder archiving (tar.gz) with automatic extraction on the remote side
- Remote file editing with offline draft saving and retry on reconnection

### SFTP Transfer Engine
- Concurrent multi-threaded transfers with pause / resume / cancel
- **Multi-part parallel upload with resume for large files**
- **Atomic overwrite protection** — writes to temp file first, then atomic rename; target file is never corrupted
- Cross-server direct transfer (FXP proxy stream) without data touching local disk

### SSH Interactive Shell
- Native SSH shell sessions with PTY window resize support
- Local escape commands (`:vssh files`, `:vssh tasks`, etc.) — seamlessly switch panels from within the shell
- Connection reuse — reuses the same SSH connection when switching from shell to file manager

### Others
- Built-in update checker with automatic download and installation
- Multi-language support (English / Simplified Chinese)
- Multiple theme options
- KeepAlive heartbeat to prevent connection timeout

---

## Installation

### Option 1: Build from Source

```bash
# Clone the repository
git clone https://github.com/motoryang/velo-ssh.git
cd velo-ssh

# Build and install (auto-injects version from VERSION file)
chmod +x scripts/install.sh && ./scripts/install.sh
```

The binary is installed to `/usr/local/bin/vssh` by default. Customize via environment variables:

```bash
PREFIX=~/.local ./scripts/install.sh
```

### Option 2: Go Install

```bash
go install github.com/motoryang/velo-ssh@latest
```

### Option 3: Download Release Binary

Download the latest binary for your platform from the [Releases page](https://github.com/motoryang/velo-ssh/releases) and place it in your `PATH`.

> **Windows users**: You can also use the `scripts/install.ps1` script.

---

## Quick Start

```bash
# Launch the TUI (main interface)
vssh

# Connect directly to a configured server shell
vssh connect <server-id-or-name>

# Open the settings panel
vssh config

# Transfer files directly between two configured servers
vssh copy <serverA>:<remote-path> <serverB>:<remote-path>

# Export configuration backup
vssh export --output backup.json --include-secrets --encrypt --passphrase "your-passphrase"

# Import configuration backup
vssh import backup.json --passphrase "your-passphrase"
```

On first launch, press `a` to add a server node.

---

## Keybindings

### Server List

| Key | Action |
|-----|--------|
| `j/k` or `↑/↓` | Move cursor |
| `/` | Search / filter servers |
| `Enter` | Establish SSH connection |
| `f` | Open file manager |
| `S` | Open settings center |
| `a` / `e` / `c` / `d` | Add / Edit / Clone / Delete server |
| `q` | Quit |

### File Manager (Split View)

| Key | Action |
|-----|--------|
| `Tab` | Switch left / right pane |
| `Space` | Select / deselect current file |
| `a` / `c` | Select all / Clear selection |
| `u` / `d` | Upload / Download |
| `y` / `v` | Copy / Paste |
| `M` | Move |
| `r` | Rename |
| `n` | New directory |
| `x` | Delete |
| `E` | Edit remote file |
| `=` | Checksum verification and text diff |
| `b` | Toggle local pane visibility |
| `/` | File search |
| `t` | Open task center |
| `R` | Refresh |

### Task Center

| Key | Action |
|-----|--------|
| `p` / `r` | Pause / Resume transfer |
| `x` | Cancel task |
| `D` | Retry drafts |
| `t` / `q` / `Esc` | Back |

### SSH Shell Escape Commands

Type at the beginning of a shell line:

| Command | Action |
|---------|--------|
| `:vssh files` | Switch to file manager |
| `:vssh tasks` | Open task center |
| `:vssh settings` | Open settings center |
| `:vssh back` | Return to server list |
| `:vssh reconnect` | Reconnect current SSH session |
| `:vssh quit` | Disconnect SSH and exit |
| `:vssh send <text>` | Force-send text to remote |

---

## Configuration & Data

Configuration and data storage paths:

| Platform | Path |
|----------|------|
| macOS / Linux | `~/.config/vssh/` |
| Windows | `%APPDATA%\vssh\` |

Directory contents:

| File | Description |
|------|-------------|
| `config.json` | Server configuration and global settings |
| `drafts.json` | Remote file edit drafts |
| `known_hosts` | Known host key records |
| `secrets/` | Keyring references for passwords and passphrases |

---

## Development

### Prerequisites

- Go 1.26+
- macOS / Linux / Windows

### Local Development

```bash
# Clone
git clone https://github.com/motoryang/velo-ssh.git
cd velo-ssh

# Build
go build -trimpath -ldflags "-X github.com/motoryang/velo-ssh/internal/version.Current=$(cat VERSION)" -o vssh .

# Run
./vssh

# Test
go test ./...
```

### Directory Structure

```
velo-ssh/
├── cmd/              # Cobra CLI command routing
│   ├── root.go       # Root command, launches TUI
│   ├── connect.go    # vssh connect
│   ├── config.go     # vssh config
│   ├── copy.go       # vssh copy
│   ├── export.go     # vssh export
│   └── import.go     # vssh import
├── internal/
│   ├── app/          # Application state definitions
│   ├── config/       # Configuration management & persistence
│   ├── ignore/       # .vsshignore filtering rules
│   ├── sshnet/       # SSH/SFTP network layer
│   ├── term/         # Terminal capability detection
│   ├── transfer/     # SFTP transfer engine
│   ├── tui/          # Bubble Tea TUI views
│   ├── updater/      # Automatic updater
│   └── version/      # Version info
├── scripts/          # Installation scripts
├── docs/             # Design documents
├── main.go           # Program entry point
└── VERSION           # Current version
```

---

## Tech Stack

- **Go** — compiled language, single-binary distribution
- **[Bubbletea](https://github.com/charmbracelet/bubbletea)** — TUI framework (MVU architecture)
- **[Bubbles](https://github.com/charmbracelet/bubbles)** — Standard TUI component library
- **[Lipgloss](https://github.com/charmbracelet/lipgloss)** — Terminal styling and layout
- **[Cobra](https://github.com/spf13/cobra)** — CLI routing framework
- **[go-keyring](https://github.com/zalando/go-keyring)** — OS keychain integration
- **[pkg/sftp](https://github.com/pkg/sftp)** — SFTP protocol implementation
- **[golang.org/x/crypto/ssh](https://golang.org/x/crypto/ssh)** — SSH protocol implementation

---

## License

This project is open-sourced under the **MIT License**. See the [LICENSE](LICENSE) file for details.

```
MIT License

Copyright (c) 2026 motoryang

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

---

## Acknowledgments

- The [Charmbracelet](https://charm.sh/) team for their excellent TUI ecosystem
- All contributors and users for their support
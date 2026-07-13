<div align="center">

# VeloSSH (vssh)

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/motoryang/velo-ssh?logo=github)](https://github.com/motoryang/velo-ssh/releases)
[![Website](https://img.shields.io/badge/Website-000000?logo=githubpages)](https://motoryang.github.io/velo-ssh/)

**一款基于 TUI 的轻量级 SSH 节点管理与双栏 SFTP 文件传输工具**

作者: [motoryang](https://github.com/motoryang) | [查看 Release](https://github.com/motoryang/velo-ssh/releases)

[English](README.md) | **简体中文**

</div>

---

## 功能特性

### 服务器管理
- 环境分组管理（dev / prod / test 等），支持搜索过滤
- 异步在线状态检测与延迟显示
- 支持密码、SSH 密钥、密钥短语等多种认证方式
- 系统 Keyring 集成，凭证安全托管于 OS 原生密钥链
- `vssh export` / `vssh import` 配置备份，支持 AES-256-GCM 加密导出

### 双栏文件管理器
- 本地 ↔ 远程双栏视图，支持单栏切换
- 文件多选、批量上传/下载
- 文件原地重命名、新建目录、粘贴、移动、删除
- 文件搜索过滤、哈希校验（SHA-256）与文本级 Diff 对比
- 支持拖拽上传和拖拽下载
- `.vsshignore` 规则过滤（自动跳过 `.DS_Store`、`Thumbs.db`、`.git/` 等）
- 文件夹智能打包压缩上传（tar.gz），远端自动解压
- 远程文件编辑，断网自动暂存草稿，重连后恢复

### SFTP 传输引擎
- 多线程并发传输，支持暂停/继续/取消
- **大文件多分块并行上传与断点续传**
- **原子覆写保护**：先写临时文件，成功后原子重命名，目标文件绝不损坏
- 跨服务器直接对传（FXP 代理流），数据不落本地

### SSH 交互式终端
- 原生 SSH Shell 会话，支持 PTY 窗口自适应调整
- 本地逃逸命令（`:vssh files`、`:vssh tasks` 等），Shell 内无缝切换面板
- 连接复用：Shell 切到文件管理器时复用同一 SSH 连接

### 其他
- 内置更新检查与自动安装
- 多语言支持（English / 简体中文）
- 多种主题切换
- KeepAlive 心跳保活，防止连接超时断开

---

## 安装

### 方式一：一键安装

```bash
# macOS / Linux
curl -fsSL https://raw.githubusercontent.com/motoryang/velo-ssh/main/scripts/install.sh | sh
```

```powershell
# Windows PowerShell
irm https://raw.githubusercontent.com/motoryang/velo-ssh/main/scripts/install.ps1 | iex
```

安装脚本会下载最新 Release 二进制到 `/opt/velossh/bin/vssh`，并在 `/usr/local/bin` 下创建 `vssh` 命令链接。
可以通过环境变量指定安装目录或版本：

```bash
INSTALL_DIR=~/.local/opt/velossh LINK_DIR=~/.local/bin VERSION=v1.1.1.26070701 sh -c "$(curl -fsSL https://raw.githubusercontent.com/motoryang/velo-ssh/main/scripts/install.sh)"
```

### 方式二：Go 安装

```bash
go install github.com/motoryang/velo-ssh@latest
```

### 方式三：下载 Release 二进制

从 [Releases 页面](https://github.com/motoryang/velo-ssh/releases) 下载对应平台的最新二进制文件，置于 `PATH` 中即可。

### 方式四：从源码构建

```bash
git clone https://github.com/motoryang/velo-ssh.git
cd velo-ssh
go build -trimpath -o vssh .
```

---

## 快速开始

```bash
# 启动 TUI（主界面）
vssh

# 直接连接到已配置的服务器 Shell
vssh connect <server-id-or-name>

# 打开设置面板
vssh config

# 两台已配置服务器之间直接传输文件
vssh copy <serverA>:<remote-path> <serverB>:<remote-path>

# 导出配置备份
vssh export --output backup.json --include-secrets --encrypt --passphrase "your-passphrase"

# 导入配置备份
vssh import backup.json --passphrase "your-passphrase"
```

首次启动后，通过快捷键 `a` 添加服务器节点。

---

## 快捷键速查

### 服务器列表

| 快捷键 | 功能 |
|--------|------|
| `j/k` 或 `↑/↓` | 移动光标 |
| `/` | 搜索过滤服务器 |
| `Enter` | 建立 SSH 连接 |
| `f` | 进入文件管理器 |
| `S` | 打开设置中心 |
| `a` / `e` / `c` / `d` | 新增 / 编辑 / 克隆 / 删除服务器 |
| `q` | 退出 |

### 文件管理器（双栏视图）

| 快捷键 | 功能 |
|--------|------|
| `Tab` | 切换左右面板 |
| `Space` | 选中/取消当前文件 |
| `a` / `c` | 全选 / 清空选择 |
| `u` / `d` | 上传 / 下载 |
| `y` / `v` | 复制 / 粘贴 |
| `M` | 移动 |
| `r` | 重命名 |
| `n` | 新建目录 |
| `x` | 删除 |
| `E` | 编辑远程文件 |
| `=` | 哈希校验与文本对比 |
| `b` | 隐藏/显示本地面板 |
| `/` | 文件搜索 |
| `t` | 打开任务中心 |
| `R` | 刷新 |

### 任务中心

| 快捷键 | 功能 |
|--------|------|
| `p` / `r` | 暂停 / 继续传输 |
| `x` | 取消任务 |
| `D` | 草稿重试 |
| `t` / `q` / `Esc` | 返回 |

### SSH Shell 逃逸命令

在 Shell 行首输入：

| 命令 | 功能 |
|------|------|
| `:vssh files` | 切换到文件管理器 |
| `:vssh tasks` | 打开任务中心 |
| `:vssh settings` | 打开设置中心 |
| `:vssh back` | 返回服务器列表 |
| `:vssh reconnect` | 重连当前 SSH |
| `:vssh quit` | 断开当前 SSH 并退出 |
| `:vssh send <text>` | 强制发送文本到远端 |

---

## 配置与数据

配置文件与数据存储路径：

| 平台 | 路径 |
|------|------|
| macOS / Linux | `~/.config/vssh/` |
| Windows | `%APPDATA%\vssh\` |

目录内容：

| 文件 | 说明 |
|------|------|
| `config.json` | 服务器节点配置与全局设置 |
| `drafts.json` | 远程文件编辑草稿暂存 |
| `known_hosts` | 已知主机密钥记录 |
| `secrets/` | 密码与密钥短语的 Keyring 引用 |

---

## 开发

### 环境要求

- Go 1.26+
- macOS / Linux / Windows

### 本地开发

```bash
# 克隆
git clone https://github.com/motoryang/velo-ssh.git
cd velo-ssh

# 构建
go build -trimpath -ldflags "-X github.com/motoryang/velo-ssh/internal/version.Current=$(cat VERSION)" -o vssh .

# 运行
./vssh

# 测试
go test ./...
```

### 目录结构

```
velo-ssh/
├── cmd/              # Cobra CLI 命令路由
│   ├── root.go       # 根命令，启动 TUI
│   ├── connect.go    # vssh connect
│   ├── config.go     # vssh config
│   ├── copy.go       # vssh copy
│   ├── export.go     # vssh export
│   └── import.go     # vssh import
├── internal/
│   ├── app/          # 应用状态定义
│   ├── config/       # 配置管理与持久化
│   ├── ignore/       # .vsshignore 过滤规则
│   ├── sshnet/       # SSH/SFTP 网络层
│   ├── term/         # 终端能力检测
│   ├── transfer/     # SFTP 传输引擎
│   ├── tui/          # Bubble Tea TUI 界面
│   ├── updater/      # 自动更新
│   └── version/      # 版本信息
├── scripts/          # 安装脚本
├── docs/             # 设计文档
├── main.go           # 程序入口
└── VERSION           # 当前版本号
```

---

## 技术栈

- **Go** — 编译语言，单二进制分发
- **[Bubbletea](https://github.com/charmbracelet/bubbletea)** — TUI 核心框架（MVU 架构）
- **[Bubbles](https://github.com/charmbracelet/bubbles)** — TUI 标准组件库
- **[Lipgloss](https://github.com/charmbracelet/lipgloss)** — 终端样式与布局
- **[Cobra](https://github.com/spf13/cobra)** — CLI 路由骨架
- **[go-keyring](https://github.com/zalando/go-keyring)** — 系统密钥链集成
- **[pkg/sftp](https://github.com/pkg/sftp)** — SFTP 协议实现
- **[golang.org/x/crypto/ssh](https://golang.org/x/crypto/ssh)** — SSH 协议实现

---

## 许可证

本项目采用 **MIT 许可证** 开源。详情请参见 [LICENSE](LICENSE) 文件。

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

## 致谢

- [Charmbracelet](https://charm.sh/) 团队提供的优秀 TUI 生态
- 所有贡献者和用户的支持

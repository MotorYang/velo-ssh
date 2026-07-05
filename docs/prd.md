# VeloSSH (vssh)

## 技术需求与设计规范文档 (PRD)

---

## 1. 项目概述与框架规范

### 1.1 项目定位

**VeloSSH (`vssh`)** 是一个基于终端用户界面（TUI）的轻量级、工业级 SSH 节点管理与双栏文件传输工具。工具采用单二进制文件分发，具备极速启动、内存占用低的特性，完美适配本地开发环境及无 GUI 的纯文本远程 Linux 服务器（跳板机）。

### 1.2 核心技术栈选型

* **开发语言**：Go (Golang) 1.22+（充分利用其轻量级协程与单文件编译特性）。
* **命令行路由**：`github.com/spf13/cobra` (作为整机 CLI 架构的基础骨架)。
* **TUI 核心框架**：`github.com/charmbracelet/bubbletea` (基于 MVU 架构的事件循环驱动引擎)。
* **TUI 标准组件**：`github.com/charmbracelet/bubbles` (提供内置 List、Textinput、Progress、Help 等组件)。
* **终端样式与排版**：`github.com/charmbracelet/lipgloss` (基于 CSS-like 理念的终端色彩与流式布局工具)。
* **底层协议驱动**：
* SSH 连接：`golang.org/x/crypto/ssh`
* SFTP 传输：`github.com/pkg/sftp`
* 凭证托管：`github.com/zalando/go-keyring`



### 1.3 运行环境适配规范

* **多端适配**：必须兼容跨平台终端模拟器（macOS Ghostty/iTerm2, Windows Terminal, Linux TTY）。
* **自适应降级**：当检测到无外网、纯文本 Linux 服务器环境时，UI 自动适配标准 ASCII 字符，防止复杂的 Unicode 符号产生排版错位。

---

## 2. UI 界面设计布局

为了将终端的方寸空间利用到极致，工具共规划四大核心面板，整体采用 **圆角边框 (Rounded Border)**、**核心光标高亮 (Cursor Highlight)** 以及 **动态底栏帮助 (Dynamic Help Bar)**。

### 2.1 面板一：主服务器列表 (Main Server List)

进入工具的第一层视图。左侧显示环境分组，右侧显示服务器矩阵，后台异步发起 Ping 检测在线状态。

```text
  ┌─────────────────────── VeloSSH MANAGER 🚀 ──────────────────┐
  │                                                             │
  │  🔎 Search... [prod______________________] (Press / to scan)│
  ├──────────────────────┬──────────────────────────────────────┤
  │ 📂 ALL ENVS          │  [prod]  🟢 Prod-Web-01  10.0.4.21   │
  │   dev   (2)          │         Host: root@10.0.4.21:22      │
  │ ❯ prod  (3)          │         Desc: 生产环境Nginx前端节点  │
  │   test  (1)          │ ──────────────────────────────────── │
  │                      │  [prod]  🔴 Prod-DB-01   10.0.5.11   │
  │ 🛠️ ACTIONS           │         Host: mysql@10.0.5.11:22     │
  │   [a] Add New        │         Desc: 数据库主节点 (离线)    │
  │   [e] Edit Selected  │ ──────────────────────────────────── │
  │   [d] Delete         │  [prod]  🟢 Prod-Redis   10.0.5.12   │
  │                      │                                      │
  ├──────────────────────┴──────────────────────────────────────┤
  │ 📝 Local Drafts: [1] pending sync (Press Ctrl+D to view)    │
  ├─────────────────────────────────────────────────────────────┤
  │ [↑/↓] Move | [/] Filter | [Enter] Connect | [f] Files | [q] │
  └─────────────────────────────────────────────────────────────┘

```

### 2.2 面板二：文件管理双栏视图 (File Manager)

默认开启左右分栏（左侧本地，右侧远端），可在设置中关闭。支持多选状态标记 `[*]` 与原地重命名输入框。

```text
  ┌───────────────────────── 📂 FILE MANAGER ─────────────────────────┐
  │                                                                   │
  │  LOCAL: /Users/username/Downloads   │ REMOTE: /var/www/html/dist/ │
  ├─────────────────────────────────────┼─────────────────────────────┤
  │   .. (Up a dir)                     │   .. (Up a dir)             │
  │   [ ] 📁 project-v2/                │   [ ] 📁 assets/            │
  │ ❯ [*] 📄 notes.txt             2 KB │   [ ] 📄 index.html      12 KB  │
  │   [*] 📄 production.zip      145 MB │ ❯ [ ] [ nginx.conf_       ] │
  │   [ ] 📄 report.pdf            4 MB │   [ ] 📄 main.js        145 KB  │
  │                                     │                             │
  ├─────────────────────────────────────┴─────────────────────────────┤
  │ 💡 Selected: 2 files (Total 145.02 MB)                            │
  │ 🔄 [2 Tasks Running]  Total: [██████░░░░] 60% (Press t)           │
  ├───────────────────────────────────────────────────────────────────┤
  │ [Space] Select | [a] Select All | [r] Rename | [u] Push | [d] Pull│
  └───────────────────────────────────────────────────────────────────┘

```

### 2.3 面板三：全域捕获 Confirm 弹窗 & 隐藏式任务中心

当外部有文件被拽入终端，或后台任务正在执行时，界面的交互与流转形态如下：

#### 文件拖入瞬时弹窗 (Confirm Modal)

```text
  │ ┌─────────────────────────────────────────────────────────┐ │
  │ │ 📥 DETECTED INCOMING FILE / FOLDER                      │ │
  │ │ ─────────────────────────────────────────────────────── │ │
  │ │  Source: /Users/username/Downloads/production-v2.zip    │ │
  │ │  Target: /var/www/html/dist/production-v2.zip           │ │
  │ │                                                         │ │
  │ │           [Enter] Confirm   |   [Esc] Cancel            │ │
  │ └─────────────────────────────────────────────────────────┘ │

```

#### 隐藏式任务中心 (Task Center，按 `t` 键展开)

```text
  ┌─────────────────── 📋 TASK CENTER (SFTP) ───────────────────┐
  │                                                             │
  │ ❯ 📥 [UP] production-v2.zip    [████████░░░░░] 65%  4.2MB/s │
  │      └─ [Space] Pause  |  [x] Cancel                        │
  │   📤 [DL] access.log           [████▒░░░░░░░░] 25%  (PAUSED)│
  │                                                             │
  ├─────────────────────────────────────────────────────────────┤
  │ [↑/↓] Select Task | [Space] Toggle Pause | [x] Delete/Cancel│
  └─────────────────────────────────────────────────────────────┘

```

### 2.4 面板四：配置与设置中心 (Settings Center)

专门管理软件全局运行逻辑的专属表单面板。支持通过快捷键或独立子命令快速呼出。

```text
  ┌────────────────────── ⚙️ VeloSSH SETTINGS ─────────────────────┐
  │                                                                │
  │ 📂 CATEGORIES         │  FileManager Settings:                 │
  │ ❯ 📁 File Manager     │ ────────────────────────────────────── │
  │   📁 SSH Conn         │  Default View Mode:  ❯ [X] Split View  │
  │   📁 Security         │                        [ ] Single View │
  │   📁 Theme / UI       │                                        │
  │                       │  Auto Tar.gz Compression: [X] Enable   │
  │ 📝 STATUS             │                                        │
  │   [Ctrl+S] Save       │  Fallback Linux Path:    [ /tmp______ ]│
  │   [Esc] Cancel        │                                        │
  │                       │  Draft Retention (TTL):  [ 30__ ] days │
  │                       │                                        │
  ├──────────────────────┴────────────────────────────────────────┤
  │ [Tab] Switch Side | [j/k] Move | [Space] Toggle | [Esc] Exit  │
  └────────────────────────────────────────────────────────────────┘

```

---

## 3. 极简快捷键体系设计

为降低用户的认知负载，设计分为**状态隔离**与**肌肉记忆兼容**两大原则：

| 所在界面 (状态) | 快捷键 | 触发行为 | 设计考量 |
| --- | --- | --- | --- |
| **全局核心** | `Ctrl+C` | 强行安全退出程序 | 终端通用规范 |
| **主服务器列表** | `↑ / ↓` 或 `j / k` | 上下移动光标 | 兼顾普通用户与 Vim 极客 |
|  | `/` | 开启模糊搜索过滤面板 | 继承 Charm 官方 List 肌肉记忆 |
|  | `Enter` | 一键建立 SSH 交互式会话 (Shell) | 确认的直觉行为 |
|  | `f` | 进入当前选中的服务器文件管理面板 | **F**iles 首字母暗示 |
|  | `Shift+S` | 原地进入软件全局配置/设置中心 | **S**ettings 首字母大写强切 |
|  | `a` / `e` / `d` | 新增 / 编辑 / 删除服务器节点配置 | **A**dd / **E**dit / **D**elete |
| **文件管理双栏** | `Tab` | 在左侧（本地）与右侧（远端）来回切换焦点 | 多端切换通用标准 |
|  | `Space` | 勾选/取消勾选当前文件（支持多选） | 光标勾选后自动下移一行 |
|  | `a` / `c` | 全选 (**A**ll) / 清空全选 (**C**lear) | 批量操作提效 |
|  | `u` / `d` | 批量上传 (**U**pload) / 批量下载 (**D**ownload) | 流式一键触发 |
|  | `r` | 原地发起“重命名”文本框输入 | **R**ename 首字母暗示 |
|  | `t` | 展开 / 隐藏后台传输任务中心 | **T**asks 状态查看开关 |
|  | `b` | 动态切换“双栏展示”与“纯远端单栏” | **B**oth sides 布局切换 |
|  | `c` / `v` | 跨服务器复制 / 粘贴 (远程对传) | 极速 FXP 对传操作流 |
|  | `=` | 触发两栏选中文件的哈希/行级 Diff 对比 | 经典的“对比”逻辑符号 |
| **SSH 交互终端** | `:vssh <command>` | 在 Shell 行首输入本地逃逸命令并回车，切换 VeloSSH 本地功能 | 避免与 Vim、tmux、nano、emacs 等远端程序快捷键冲突 |

#### SSH Shell 本地逃逸命令

不使用 `Ctrl` 类热键作为 Shell/TUI 切换入口。为避免与 Vim、tmux、nano、emacs 等远端程序冲突，VeloSSH 在交互式 Shell 中引入本地逃逸命令机制。用户在当前 Shell 行首输入 `:vssh <command>` 并回车时，Stream Wrapper 在本地拦截该完整行，解析为 VeloSSH 控制指令，不发送至远端 SSH 会话。

| 命令 | 触发行为 |
| --- | --- |
| `:vssh files` | 挂起当前 Shell，切换到文件管理器 |
| `:vssh tasks` | 打开后台任务中心 |
| `:vssh settings` | 打开设置中心 |
| `:vssh back` | 挂起 Shell，返回主服务器列表 |
| `:vssh reconnect` | 重连当前 SSH 会话 |
| `:vssh quit` | 断开当前 SSH 会话 |
| `:vssh help` | 显示本地逃逸命令帮助 |
| `:vssh send <text>` | 强制把 `<text>` 发送给远端 |

**逃逸与透传**：
只有在 SSH Shell 的“本地命令模式”中才拦截 `:vssh <command>`；普通输入的 `:` 会原样发给远端。若用户确实需要把 `:vssh` 开头的文本发送给远端，可输入：

```text
:vssh send :vssh files
```

---

## 4. 功能架构与底线防卫设计

### 4.1 弱网与大文件防卫机制 (工业级稳健)

为防止在网络动荡或传输大文件时导致远端服务器上的原配置文件损坏（如变成 `0 字节` 文件），必须引入以下防卫铁三角：

1. **远端原子级覆写**：
* 绝不直接覆写目标文件。上传时，先在远端同目录下生成 `.target.tmp.xxxx` 隐藏文件。
* 传输且校验成功后，通过底层协议发送原子级重命名指令 `mv .target.tmp.xxxx target`。若中途断线，原文件毫发无损。


2. **本地草稿箱暂存 (`drafts.json`)**：
* 当调用本地编辑器（Vim）盲改远端文件时，若回传因断网失败，禁止清理本地临时文件。
* 将该草稿路径与关联节点写入本地持久化文件 `~/.config/vssh/drafts.json`。
* **清理时机**：下一次用户连入该节点时提示重试，重试上传成功后，索引与本地草稿同步销毁；超过 `draft_ttl_days`（默认 30 天）未处理的草稿执行滚动淘汰。


3. **多线程切块与断点续传**：
* 大文件采用分块（Multi-part）并发写入，使用 `sftpClient.WriteAt(chunk, offset)`。
* 弱网断线后，通过 `Stat` 判定远端已接收的字节偏移量，直接 `Seek` 续传，免去重新传输的代价。



### 4.2 SSH Shell 本地逃逸命令逻辑

利用本地“中间人拦截器（Stream Wrapper）”读取用户在交互式 Shell 中提交的完整输入行。当输入行以 `:vssh ` 开头并匹配内置命令时，VeloSSH 在本地解析并执行控制指令，不将该行发送至远端 SSH 会话。

* **普通透传阶段**：默认情况下，`os.Stdin` 的输入字节持续转发给远端 SSH 会话，用户输入的普通 `:`、Vim 命令、数据库客户端命令均不被 VeloSSH 截获。
* **本地命令判定阶段**：仅当用户在当前 Shell 行首输入 `:vssh <command>` 并回车后，Stream Wrapper 才拦截该完整行并进行本地命令解析。
* **本地切换阶段**：执行 `:vssh files`、`:vssh tasks`、`:vssh settings`、`:vssh back` 等命令时，暂缓向远端会话喂流，并将控制权交还给 Bubble Tea 对应面板。
* **连接复用阶段**：切入文件管理器时，同一套底层 `ssh.Client` 实例化出 `sftp.Client`，复用当前连接，避免重新建立握手。
* **强制透传阶段**：若用户需要把 `:vssh` 开头的文本发送到远端，可输入 `:vssh send <text>`，例如 `:vssh send :vssh files`。
* **退出与重连阶段**：`:vssh quit` 断开当前 SSH 会话；`:vssh reconnect` 关闭旧会话并使用当前节点配置重新建立 SSH/SFTP 通道。

---

## 5. 高级优化项设计 (v1.1+ 路线图)

### 5.1 极速文件夹分发：智能压缩流传输

针对含有海量小文件的文件夹（如前端 `dist` 目录），通过 SFTP 逐个传输会消耗剧烈的网络握手开销。

* **逻辑**：用户触发文件夹上传时，工具在本地调用 `archive/tar` 与 `compress/gzip` 静默打包为单一流文件 `.tmp_pack.tar.gz`。
* **释放**：单文件经多线程满带宽传至远端后，工具自动通过 SSH 通道静默向远端发送解压解包并自毁的单行原子指令：
```bash
tar -fvxz .tmp_pack.tar.gz -C /target/dir && rm .tmp_pack.tar.gz

```



### 5.2 系统级安全凭证托管 (Keyring 集成)

拒绝明文或弱加密形式将敏感密码硬编码于本地配置文件中。

* **机制**：引入 `github.com/zalando/go-keyring`，解耦配置项。`config.json` 仅保留非敏感字段。
* **底层分配**：密码与密钥短语（Passphrase）在用户录入时暗中托管至操作系统原生安全库：
* macOS: `Security/Keychain`
* Windows: `Credential Manager`
* Linux: `Secret Service API via D-Bus`



### 5.3 交互式 Shell 窗口动态自适应 (PTY Resize)

防止用户在远程终端执行 `top`、`vim` 等强渲染工具时，因本地拖拽终端窗口导致远端画面卡死或缩水。

* **逻辑**：Bubble Tea 的全局事件循环持续捕获 `tea.WindowSizeMsg`。
* **同步**：在重绘本地 TUI 的同时，通过底层 SSH 信道即刻调用 `session.WindowChange(msg.Height, msg.Width)`，强制远端伪终端（PTY）完成像素级对齐与无缝重绘。

### 5.4 跨服务器直接对传通道 (FXP 代理流)

打通“服务器到服务器”的数据高速公路。

* **场景**：在服务器 A 选中文件按 `c` (Copy)，切换至服务器 B 选中目录按 `v` (Paste)。
* **流向**：数据绝不落地充洗用户本地物理硬盘。若两台机器网络互通，工具通过 SSH 在 A 机器直接下发 `rsync/scp` 直连 B 传；若不互通，则在工具运行内存中构建双向流管道桥接（`io.Copy(B_SFTP_Session, A_SFTP_Session)`），完全绕开本地网络带宽瓶颈。

### 5.5 跨机快速导入导出 (vssh export/import)

* 支持通过命令行 `vssh export --output backup.json` 一键打包并导出服务器节点矩阵配置。
* 导出的敏感资产凭证可选择置空，或使用用户实时输入的临时 Passphrase 触发 AES-256 高强度加密，确保配置在团队内部共享或换机分发时的绝对安全。

### 5.6 智能哈希校验与行级 Diff

* 文件管理双栏中引入 `=` 快捷键。针对跨栏对应的同名文件，后台自动发起对等流读取并计算 MD5 值，若完全一致则在状态栏高亮提示 `🟢 哈希一致`。
* 针对不一致的非二进制文件（如 `.yaml`, `.conf`），支持原地触发差异渲染视图，利用 Lipgloss 左右流拼接，以红绿两色直观展示远端与本地配置文件的详细代差。

### 5.7 传输防卫：`.vsshignore` 静态过滤器

* 工具在执行文件夹递归上传时，内部维护一套静态黑名单过滤器，默认跳过 `.DS_Store`、`Thumbs.db`、`.git/` 等系统冗余缓存。
* 允许用户在本地项目根目录下放置 `.vsshignore` 规则文件，声明需要被过滤掉的巨型开发依赖包（如 `node_modules/`）或本地生成的大体积日志，最大化释放多线程通道的网络传输带宽。

---

## 6. 系统核心代码框架

### 6.1 目录结构规划

```text
velo-ssh/
├── cmd/
│   ├── root.go            # Cobra 根路由命令 (vssh)
│   └── config.go          # Cobra 配置专有子命令 (vssh config)
├── config/
│   ├── storage.go         # 节点配置与 drafts.json 持久化存储
│   └── theme.go           # Lipgloss 样式方案定义
├── tui/
│   ├── model.go           # Bubble Tea 主状态 Model
│   ├── view_list.go       # 面板一：服务器列表渲染
│   ├── view_file.go       # 面板二：双栏文件列表渲染
│   ├── view_task.go       # 面板三：任务中心与表单渲染
│   └── view_settings.go   # 面板四：全局配置设置中心渲染
├── sshnet/
│   ├── client.go          # SSH/SFTP 双通道连接池管理
│   ├── sftp_worker.go     # 多线程并发传输、暂停、取消、压缩传输逻辑
│   └── interceptor.go     # Stdin 流中间人与 :vssh 本地逃逸命令解析器
└── main.go                # 程序主入口

```

### 6.2 Cobra 命令行路由分流设计

在 `cmd/config.go` 中拦截 `vssh config` 命令，使其跳过主列表，直达设置状态。

```go
package cmd

import (
	"fmt"
	"os"
	"github.com/spf13/cobra"
	"github.com/charmbracelet/bubbletea"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Open VeloSSH interactive settings center",
	Run: func(cmd *cobra.Command, args []string) {
		// 1. 实例化 TUI 模型，并将初始状态强制指定为设置面板
		initialModel := NewModel(StateSettingsCenter) 
		
		p := tea.NewProgram(initialModel, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running config editor: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}

```

### 6.3 核心 TUI 状态机骨架代码

```go
package main

import (
	"strings"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"
)

type AppState int
const (
	StateServerList AppState = iota
	StateFileManager
	StateConfirmUpload
	StateTaskCenter
	StateSettingsCenter
)

type fileItem struct {
	name     string
	size     int64
	isDir    bool
	selected bool
}

type model struct {
	state        AppState
	serverList   list.Model
	hiddenInput  textinput.Model 
	localFiles   []fileItem
	remoteFiles  []fileItem
	cursorIndex  int
	detectedPath string          
	isSplitView  bool            
}

// 构造函数支持动态注入起始状态（契合 vssh config 命令）
func NewModel(startState AppState) model {
	return model{
		state:       startState,
		isSplitView: true,
		// 初始化其余子组件...
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.serverList.SetSize(msg.Width, msg.Height)
		
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		// 全局强切设置热键
		if msg.String() == "S" {
			m.state = StateSettingsCenter
			return m, nil
		}

		switch m.state {
		case StateServerList:
			if msg.String() == "enter" {
				return m, tea.Quit 
			}
			if msg.String() == "f" {
				m.state = StateFileManager
				return m, nil
			}

		case StateFileManager:
			switch msg.String() {
			case "tab":
			case " ":
				m.remoteFiles[m.cursorIndex].selected = !m.remoteFiles[m.cursorIndex].selected
			case "t":
				m.state = StateTaskCenter
			case "=":
				// 触发双栏哈希校验 / 文本对比逻辑
			case "esc":
				m.state = StateServerList
			}

		case StateSettingsCenter:
			switch msg.String() {
			case "ctrl+s":
				// 执行持久化保存：io.WriteFile 到 config.json
				m.state = StateServerList // 保存后退回主页
				return m, nil
			case "esc":
				m.state = StateServerList // 放弃更改退回主页
				return m, nil
			}
		}
	}

	if m.state == StateFileManager {
		var cmd tea.Cmd
		m.hiddenInput, cmd = m.hiddenInput.Update(msg)
		if m.hiddenInput.Value() != "" {
			m.detectedPath = strings.Trim(m.hiddenInput.Value(), " '\"")
			m.hiddenInput.SetValue("") 
			m.state = StateConfirmUpload
			return m, cmd
		}
	}

	return m, nil
}

```

---

## 7. 关键注意点与避坑指南

1. **终端挂起锁死风险 (`Ctrl+S`)**：
* 不要在主界面的高频操作中设计 `Ctrl+S`。在许多传统 Linux 虚拟终端（TTY）中，`Ctrl+S` 会触发 XOFF 流控制，直接挂起并卡死整个终端输入流，需按 `Ctrl+Q` 才能恢复。本工具在设置中心采用 `Ctrl+S` 作为保存触发表单，该行为仅在 TUI 捕获状态下有效，退出后必须彻底释放，防止污染用户的正常伪终端交互。


2. **SSH 线程池的连接数阈值**：
* 多线程上传/下载的并发 Worker 数量默认应收敛在 **3 ~ 5 个**。
* 盲目追求高并发会导致瞬间向目标机器发起过多 SSH 会话，极易触发远端服务器 `/etc/ssh/sshd_config` 中 `MaxStartups` 或 `MaxSessions` 的安全阈值。


3. **大文件内存穿透预防与流缓存**：
* 严禁在多线程切块读取或执行 **5.1 智能压缩传输** 与 **5.4 跨机器对传** 时使用 `io.ReadAll` 将整个文件或压缩流一次性载入内存。
* 必须限定每个并发分块缓存（Buffer）的最大尺寸（如 4MB），读完即刻写入通道并刷新。


4. **字符宽度计算错位（双栏对齐破坏）**：
* 中文字符或 Emoji 字符在终端中占据两个光标位（Ambiguous Width）。在计算左右双栏 50% 宽度时，直接使用 `len(string)` 会引发灾难性的排版错位。
* 必须引入 **`github.com/mattn/go-runewidth`** 库来精确计算文本上实际占据的物理视觉宽度。


5. **静默心跳机制预防信道超时 (KeepAlive)**：
* 用户可能在文件面板或配置设置面板长时间停留，这会导致底层连接因长时间无数据流过而被远端服务器静默踢掉。
* 必须在底层连接池启动定时器，每隔 20 秒向远端发送一个标准的全局空请求（Global Request），维持信道热度。

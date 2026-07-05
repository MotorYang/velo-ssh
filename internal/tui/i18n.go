package tui

import "github.com/motoryang/velo-ssh/internal/config"

const (
	textManagerTitle            = "manager.title"
	textFilter                  = "manager.filter"
	textSearch                  = "manager.search"
	textNoServers               = "manager.noServers"
	textSettingsTitle           = "settings.title"
	textSettingsGuide           = "settings.guide"
	textSettingsOK              = "settings.ok"
	textSettingsCancel          = "settings.cancel"
	textSettingsSaved           = "settings.saved"
	textFooterServerSearch      = "footer.server.search"
	textFooterServerList        = "footer.server.list"
	textFooterServerForm        = "footer.server.form"
	textFooterSettingsDiscard   = "footer.settings.discard"
	textFooterBack              = "footer.back"
	textFooterConfirm           = "footer.confirm"
	textFooterCancel            = "footer.cancel"
	textFooterKeepEditing       = "footer.keepEditing"
	textConfirmTitle            = "confirm.title"
	textAddServerTitle          = "serverForm.add"
	textEditServerTitle         = "serverForm.edit"
	textCloneServerTitle        = "serverForm.clone"
	textServerFormGuide         = "serverForm.guide"
	textDeleteServerPrompt      = "modal.deleteServer"
	textDeleteServerBody        = "modal.deleteServer.body"
	textMissingHostKeyContext   = "modal.hostKey.missing"
	textTrustHostKeyPrompt      = "modal.hostKey.trust"
	textTarget                  = "label.target"
	textKnownHosts              = "label.knownHosts"
	textHostKeyWarning          = "modal.hostKey.warning"
	textTrustAndRetry           = "modal.hostKey.action"
	textOverwritePrompt         = "modal.overwrite"
	textOverwriteBody           = "modal.overwrite.body"
	textOverwriteAction         = "modal.overwrite.action"
	textDeletePathsPrompt       = "modal.deletePaths"
	textDeletePathsBody         = "modal.deletePaths.body"
	textDeleteAction            = "modal.delete.action"
	textCancelTaskPrompt        = "modal.cancelTask"
	textTask                    = "label.task"
	textPath                    = "label.path"
	textCancelTaskBody          = "modal.cancelTask.body"
	textCancelTaskAction        = "modal.cancelTask.action"
	textKeepTaskAction          = "modal.cancelTask.keep"
	textDiscardServerPrompt     = "modal.discardServer"
	textDiscardServerBody       = "modal.discardServer.body"
	textDiscardAction           = "modal.discard.action"
	textKeepEditingAction       = "modal.keepEditing.action"
	textServerAdded             = "status.server.added"
	textServerUpdated           = "status.server.updated"
	textServerDeleted           = "status.server.deleted"
	textServerDiscarded         = "status.server.discarded"
	textTransferOverwriteCancel = "status.transfer.overwriteCancel"
	textDeleteCanceled          = "status.delete.canceled"
	textErrorPrefix             = "error.prefix"
	textSearchServerPlaceholder = "placeholder.serverSearch"
	textSearchFilePlaceholder   = "placeholder.fileSearch"
	textFileManagerTitle        = "file.title"
	textLocal                   = "file.local"
	textRemote                  = "file.remote"
	textRemoteRequiresSSH       = "file.remoteRequiresSSH"
	textSearchInput             = "file.searchInput"
	textRows                    = "file.rows"
	textFilteredFrom            = "file.filteredFrom"
	textRename                  = "file.rename"
	textNewDirectory            = "file.newDirectory"
	textSelectedColumn          = "file.column.selected"
	textModeColumn              = "file.column.mode"
	textNameColumn              = "file.column.name"
	textSizeColumn              = "file.column.size"
	textModifiedColumn          = "file.column.modified"
	textTaskCenterTitle         = "task.title"
	textNoTransferTasks         = "task.none"
	textFooterFileSearch        = "footer.file.search"
	textFooterRename            = "footer.file.rename"
	textFooterCreateDir         = "footer.file.createDir"
	textFooterFileSingle        = "footer.file.single"
	textFooterFileSplit         = "footer.file.split"
	textFooterTaskCenter        = "footer.taskCenter"
)

var zhText = map[string]string{
	textManagerTitle:            "VeloSSH 服务器管理",
	textFilter:                  "筛选",
	textSearch:                  "搜索",
	textNoServers:               "没有已配置或匹配的服务器。",
	textSettingsTitle:           "VeloSSH 设置",
	textSettingsGuide:           "Tab/上/下移动焦点。左/右或空格切换选项。Enter 确认/取消。",
	textSettingsOK:              "确定",
	textSettingsCancel:          "取消",
	textSettingsSaved:           "设置已保存。",
	textFooterServerSearch:      "[Enter] 应用筛选 | [Esc] 清空筛选",
	textFooterServerList:        "[j/k] 移动 | [/] 筛选 | [Enter] 连接 | [f] 文件 | [S] 设置 | [a/e/c/d] 新增/编辑/克隆/删除 | [q] 退出",
	textFooterServerForm:        "[Tab/下] 下一项 | [Shift+Tab/上] 上一项 | [左/右/空格] 切换选项 | [Enter] 下一项/保存 | [Esc] 取消",
	textFooterSettingsDiscard:   "[Enter]/[y] 丢弃修改 | [Esc]/[n] 继续编辑",
	textFooterBack:              "[Esc]/[q] 返回",
	textFooterConfirm:           "[Enter]/[y] 确认 | [Esc]/[n] 取消",
	textFooterCancel:            "[Esc]/[n] 取消",
	textFooterKeepEditing:       "[Esc]/[n] 继续编辑",
	textConfirmTitle:            "确认",
	textAddServerTitle:          "新增服务器",
	textEditServerTitle:         "编辑服务器",
	textCloneServerTitle:        "克隆服务器",
	textServerFormGuide:         "ID 会自动生成。Auth Type：使用左/右/空格切换。",
	textDeleteServerPrompt:      "删除服务器",
	textDeleteServerBody:        "这会从 ~/.config/vssh/config.json 中移除该服务器。",
	textMissingHostKeyContext:   "缺少主机密钥确认上下文。",
	textTrustHostKeyPrompt:      "信任 SSH 主机密钥",
	textTarget:                  "目标",
	textKnownHosts:              "Known hosts",
	textHostKeyWarning:          "仅当该指纹与预期服务器一致时才接受。",
	textTrustAndRetry:           "[Enter]/[y] 信任并重试 | [Esc]/[n] 取消",
	textOverwritePrompt:         "覆盖已存在的目标文件？",
	textOverwriteBody:           "已有目标只会在原子传输成功后被替换。",
	textOverwriteAction:         "[Enter]/[y] 覆盖 | [Esc]/[n] 取消",
	textDeletePathsPrompt:       "删除选中的路径？",
	textDeletePathsBody:         "该操作无法由 VeloSSH 撤销。",
	textDeleteAction:            "[Enter]/[y] 删除 | [Esc]/[n] 取消",
	textCancelTaskPrompt:        "取消并移除传输任务？",
	textTask:                    "任务",
	textPath:                    "路径",
	textCancelTaskBody:          "正在运行的传输会被取消，临时文件会尽量清理，并从任务列表移除记录。",
	textCancelTaskAction:        "[Enter]/[y] 取消任务",
	textKeepTaskAction:          "[Esc]/[n] 保留任务",
	textDiscardServerPrompt:     "丢弃未保存的服务器修改？",
	textDiscardServerBody:       "你已经修改了服务器表单字段。现在离开会丢失这些修改。",
	textDiscardAction:           "[Enter]/[y] 丢弃修改",
	textKeepEditingAction:       "[Esc]/[n] 继续编辑",
	textServerAdded:             "服务器已添加。",
	textServerUpdated:           "服务器已更新。",
	textServerDeleted:           "已删除服务器 %s。",
	textServerDiscarded:         "服务器表单修改已丢弃。",
	textTransferOverwriteCancel: "覆盖前已取消传输。",
	textDeleteCanceled:          "删除已取消。",
	textErrorPrefix:             "错误",
	textSearchServerPlaceholder: "按名称、环境、主机、用户、标签筛选",
	textSearchFilePlaceholder:   "筛选文件",
	textFileManagerTitle:        "文件管理器",
	textLocal:                   "本地",
	textRemote:                  "远端",
	textRemoteRequiresSSH:       "远端面板需要有效的 SSH/SFTP 连接。",
	textSearchInput:             "搜索输入",
	textRows:                    "行",
	textFilteredFrom:            "过滤自",
	textRename:                  "重命名",
	textNewDirectory:            "新建目录",
	textSelectedColumn:          "选中",
	textModeColumn:              "权限",
	textNameColumn:              "名称",
	textSizeColumn:              "大小",
	textModifiedColumn:          "修改时间",
	textTaskCenterTitle:         "任务中心",
	textNoTransferTasks:         "没有传输任务。",
	textFooterFileSearch:        "[Enter] 应用文件搜索 | [Esc] 取消搜索",
	textFooterRename:            "[Enter] 保存重命名 | [Esc] 取消重命名",
	textFooterCreateDir:         "[Enter] 创建目录 | [Esc] 取消",
	textFooterFileSingle:        "[/] 搜索 | [b] 显示本地 | [q] SSH 面板 | [Enter] 打开 | [Space] 选择 | [y] 复制 | [v] 粘贴 | [M] 移动 | [n] 新建目录 | [x] 删除 | [r] 重命名 | [m] 切换时间 | [d] 下载 | [R] 刷新 | [t] 任务",
	textFooterFileSplit:         "[/] 搜索 | [Tab] 面板 | [b] 隐藏本地 | [q] SSH 面板 | [Enter] 打开 | [Space] 选择 | [a] 全选 | [c] 清空 | [y] 复制 | [v] 粘贴 | [M] 移动 | [n] 新建目录 | [x] 删除 | [r] 重命名 | [u] 上传 | [m] 切换时间 | [d] 下载 | [R] 刷新 | [t] 任务",
	textFooterTaskCenter:        "[j/k] 移动 | [p] 暂停 | [r] 继续 | [x] 取消任务 | [R] 刷新 | [t]/[q]/[Esc] 返回",
}

func (m Model) tr(key string) string {
	if m.config.Settings.Language == config.LanguageSimplifiedChinese {
		if value, ok := zhText[key]; ok {
			return value
		}
	}
	return enText(key)
}

func enText(key string) string {
	switch key {
	case textManagerTitle:
		return "VeloSSH Manager"
	case textFilter:
		return "Filter"
	case textSearch:
		return "Search"
	case textNoServers:
		return "No servers configured or matched."
	case textSettingsTitle:
		return "VeloSSH Settings"
	case textSettingsGuide:
		return "Tab/Up/Down move focus. Left/Right or Space changes options. Enter OK/Cancel."
	case textSettingsOK:
		return "OK"
	case textSettingsCancel:
		return "Cancel"
	case textSettingsSaved:
		return "Settings saved."
	case textFooterServerSearch:
		return "[Enter] Apply Filter | [Esc] Clear Filter"
	case textFooterServerList:
		return "[j/k] Move | [/] Filter | [Enter] Connect | [f] Files | [S] Settings | [a/e/c/d] Add/Edit/Clone/Delete | [q] Quit"
	case textFooterServerForm:
		return "[Tab/Down] Next | [Shift+Tab/Up] Previous | [Left/Right/Space] Change Option | [Enter] Next/Save | [Esc] Cancel"
	case textFooterSettingsDiscard:
		return "[Enter]/[y] Discard Changes | [Esc]/[n] Keep Editing"
	case textFooterBack:
		return "[Esc]/[q] Back"
	case textFooterConfirm:
		return "[Enter]/[y] Confirm | [Esc]/[n] Cancel"
	case textFooterCancel:
		return "[Esc]/[n] Cancel"
	case textFooterKeepEditing:
		return "[Esc]/[n] Keep Editing"
	case textConfirmTitle:
		return "Confirm"
	case textAddServerTitle:
		return "Add Server"
	case textEditServerTitle:
		return "Edit Server"
	case textCloneServerTitle:
		return "Clone Server"
	case textServerFormGuide:
		return "ID is generated automatically. Auth Type: use Left/Right/Space to change."
	case textDeleteServerPrompt:
		return "Delete server"
	case textDeleteServerBody:
		return "This removes it from ~/.config/vssh/config.json."
	case textMissingHostKeyContext:
		return "Missing host key confirmation context."
	case textTrustHostKeyPrompt:
		return "Trust SSH host key"
	case textTarget:
		return "Target"
	case textKnownHosts:
		return "Known hosts"
	case textHostKeyWarning:
		return "Accept only if this fingerprint matches the server you expect."
	case textTrustAndRetry:
		return "[Enter]/[y] Trust and retry | [Esc]/[n] Cancel"
	case textOverwritePrompt:
		return "Overwrite existing target file(s)?"
	case textOverwriteBody:
		return "Existing targets are only replaced after the atomic transfer succeeds."
	case textOverwriteAction:
		return "[Enter]/[y] Overwrite | [Esc]/[n] Cancel"
	case textDeletePathsPrompt:
		return "Delete selected path(s)?"
	case textDeletePathsBody:
		return "This operation cannot be undone by VeloSSH."
	case textDeleteAction:
		return "[Enter]/[y] Delete | [Esc]/[n] Cancel"
	case textCancelTaskPrompt:
		return "Cancel and remove transfer task?"
	case textTask:
		return "Task"
	case textPath:
		return "Path"
	case textCancelTaskBody:
		return "The running transfer is canceled, temporary files are removed when possible, and the task record is removed from the list."
	case textCancelTaskAction:
		return "[Enter]/[y] Cancel Task"
	case textKeepTaskAction:
		return "[Esc]/[n] Keep Task"
	case textDiscardServerPrompt:
		return "Discard unsaved server changes?"
	case textDiscardServerBody:
		return "You have edited fields in this server form. Leaving now will lose those changes."
	case textDiscardAction:
		return "[Enter]/[y] Discard Changes"
	case textKeepEditingAction:
		return "[Esc]/[n] Keep Editing"
	case textServerAdded:
		return "Server added."
	case textServerUpdated:
		return "Server updated."
	case textServerDeleted:
		return "Deleted server %s."
	case textServerDiscarded:
		return "Server form changes discarded."
	case textTransferOverwriteCancel:
		return "Transfer canceled before overwriting existing target."
	case textDeleteCanceled:
		return "Delete canceled."
	case textErrorPrefix:
		return "ERROR"
	case textSearchServerPlaceholder:
		return "filter by name, env, host, user, tag"
	case textSearchFilePlaceholder:
		return "filter files"
	case textFileManagerTitle:
		return "File Manager"
	case textLocal:
		return "LOCAL"
	case textRemote:
		return "REMOTE"
	case textRemoteRequiresSSH:
		return "Remote pane requires an active SSH/SFTP connection."
	case textSearchInput:
		return "Search input"
	case textRows:
		return "rows"
	case textFilteredFrom:
		return "filtered from"
	case textRename:
		return "Rename"
	case textNewDirectory:
		return "New directory"
	case textSelectedColumn:
		return "Sel"
	case textModeColumn:
		return "Mode"
	case textNameColumn:
		return "Name"
	case textSizeColumn:
		return "Size"
	case textModifiedColumn:
		return "Modified"
	case textTaskCenterTitle:
		return "Task Center"
	case textNoTransferTasks:
		return "No transfer tasks."
	case textFooterFileSearch:
		return "[Enter] Apply File Search | [Esc] Cancel Search"
	case textFooterRename:
		return "[Enter] Save Rename | [Esc] Cancel Rename"
	case textFooterCreateDir:
		return "[Enter] Create Directory | [Esc] Cancel"
	case textFooterFileSingle:
		return "[/] Search | [b] Show Local | [q] SSH Panel | [Enter] Open | [Space] Select | [y] Copy | [v] Paste | [M] Move | [n] New Dir | [x] Delete | [r] Rename | [m] Toggle Time | [d] Download | [R] Refresh | [t] Tasks"
	case textFooterFileSplit:
		return "[/] Search | [Tab] Pane | [b] Hide Local | [q] SSH Panel | [Enter] Open | [Space] Select | [a] All | [c] Clear | [y] Copy | [v] Paste | [M] Move | [n] New Dir | [x] Delete | [r] Rename | [u] Upload | [m] Toggle Time | [d] Download | [R] Refresh | [t] Tasks"
	case textFooterTaskCenter:
		return "[j/k] Move | [p] Pause | [r] Resume | [x] Cancel Task | [R] Refresh | [t]/[q]/[Esc] Back"
	default:
		return key
	}
}

func serverFormLabel(index int, language string) string {
	if language == config.LanguageSimplifiedChinese {
		switch index {
		case serverFieldID:
			return "ID"
		case serverFieldName:
			return "名称"
		case serverFieldEnv:
			return "环境"
		case serverFieldHost:
			return "主机"
		case serverFieldPort:
			return "端口"
		case serverFieldUser:
			return "用户"
		case serverFieldAuthType:
			return "认证类型"
		case serverFieldKeyPath:
			return "密钥路径"
		case serverFieldPasswordRef:
			return "密码引用"
		case serverFieldPasswordSecret:
			return "密码"
		case serverFieldPassphraseRef:
			return "密钥短语引用"
		case serverFieldPassphraseSecret:
			return "密钥短语"
		case serverFieldDesc:
			return "描述"
		case serverFieldDefaultRemotePath:
			return "默认远端路径"
		case serverFieldTags:
			return "标签（逗号分隔）"
		}
	}
	return serverFormLabels[index]
}

func settingsLabel(index int, language string) string {
	if language == config.LanguageSimplifiedChinese {
		switch index {
		case settingsFieldDefaultViewMode:
			return "默认视图模式"
		case settingsFieldASCIIFallback:
			return "ASCII 兼容模式"
		case settingsFieldFallbackRemotePath:
			return "默认远端路径"
		case settingsFieldDraftTTLDays:
			return "草稿保留天数"
		case settingsFieldTransferConcurrency:
			return "传输并发数"
		case settingsFieldKeepAliveSeconds:
			return "KeepAlive 秒数"
		case settingsFieldTheme:
			return "主题"
		case settingsFieldLanguage:
			return "语言"
		case settingsFieldConfirmOverwrite:
			return "覆盖前确认"
		case settingsFieldKnownHostsPolicy:
			return "Known Hosts 策略"
		}
	}
	return settingsFormLabels[index]
}

func optionLabel(field int, value string) string {
	if field == settingsFieldLanguage {
		switch value {
		case config.LanguageEnglish:
			return "English"
		case config.LanguageSimplifiedChinese:
			return "简体中文"
		}
	}
	return value
}

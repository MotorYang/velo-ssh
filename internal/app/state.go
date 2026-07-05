package app

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

func (s AppState) String() string {
	switch s {
	case StateServerList:
		return "server_list"
	case StateServerForm:
		return "server_form"
	case StateShell:
		return "shell"
	case StateFileManager:
		return "file_manager"
	case StateConfirmModal:
		return "confirm_modal"
	case StateTaskCenter:
		return "task_center"
	case StateSettingsCenter:
		return "settings_center"
	case StateHelp:
		return "help"
	default:
		return "unknown"
	}
}

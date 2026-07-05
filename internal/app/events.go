package app

type EscapeAction string

const (
	EscapeNone      EscapeAction = ""
	EscapeFiles     EscapeAction = "files"
	EscapeTasks     EscapeAction = "tasks"
	EscapeSettings  EscapeAction = "settings"
	EscapeBack      EscapeAction = "back"
	EscapeReconnect EscapeAction = "reconnect"
	EscapeQuit      EscapeAction = "quit"
	EscapeHelp      EscapeAction = "help"
	EscapeSend      EscapeAction = "send"
)

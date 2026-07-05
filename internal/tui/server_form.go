package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/motoryang/velo-ssh/internal/config"
)

const (
	serverFieldID = iota
	serverFieldName
	serverFieldEnv
	serverFieldHost
	serverFieldPort
	serverFieldUser
	serverFieldAuthType
	serverFieldKeyPath
	serverFieldPasswordRef
	serverFieldPasswordSecret
	serverFieldPassphraseRef
	serverFieldPassphraseSecret
	serverFieldDesc
	serverFieldDefaultRemotePath
	serverFieldTags
)

type serverFormValue struct {
	Server     config.Server
	Password   string
	Passphrase string
}

func newServerForm(mode string, srv config.Server) serverForm {
	defaultRemote := srv.DefaultRemotePath
	if defaultRemote == "" {
		defaultRemote = defaultRemotePathForUser(srv.User)
	}
	values := []string{
		srv.ID,
		srv.Name,
		srv.Env,
		srv.Host,
		strconv.Itoa(defaultPort(srv.Port)),
		srv.User,
		defaultString(srv.AuthType, config.AuthAgent),
		srv.KeyPath,
		srv.PasswordRef,
		"",
		srv.PassphraseRef,
		"",
		srv.Desc,
		defaultRemote,
		strings.Join(srv.Tags, ","),
	}
	fields := make([]textinput.Model, len(serverFormLabels))
	for i := range fields {
		fields[i] = textinput.New()
		fields[i].Prompt = ""
		fields[i].Placeholder = serverFormLabels[i]
		fields[i].SetValue(values[i])
		fields[i].CharLimit = 256
	}
	fields[serverFieldPasswordSecret].EchoMode = textinput.EchoPassword
	fields[serverFieldPassphraseSecret].EchoMode = textinput.EchoPassword
	start := serverFieldName
	fields[start].Focus()
	initialValues := make([]string, len(values))
	copy(initialValues, values)
	return serverForm{
		mode:             mode,
		originalID:       srv.ID,
		fields:           fields,
		initialValues:    initialValues,
		index:            start,
		remotePathManual: srv.DefaultRemotePath != "" && srv.DefaultRemotePath != defaultRemotePathForUser(srv.User),
	}
}

func (f *serverForm) focusNext() {
	f.fields[f.index].Blur()
	f.index = f.nextVisibleIndex(1)
	f.fields[f.index].Focus()
}

func (f *serverForm) focusPrev() {
	f.fields[f.index].Blur()
	f.index = f.nextVisibleIndex(-1)
	f.fields[f.index].Focus()
}

func (f serverForm) server() (serverFormValue, error) {
	port, err := strconv.Atoi(strings.TrimSpace(f.fields[serverFieldPort].Value()))
	if err != nil {
		return serverFormValue{}, fmt.Errorf("port must be a number")
	}
	authType := strings.TrimSpace(f.fields[serverFieldAuthType].Value())
	if authType == "" {
		authType = config.AuthAgent
	}
	srv := config.Server{
		ID:                strings.TrimSpace(f.fields[serverFieldID].Value()),
		Name:              strings.TrimSpace(f.fields[serverFieldName].Value()),
		Env:               strings.TrimSpace(f.fields[serverFieldEnv].Value()),
		Host:              strings.TrimSpace(f.fields[serverFieldHost].Value()),
		Port:              port,
		User:              strings.TrimSpace(f.fields[serverFieldUser].Value()),
		AuthType:          authType,
		KeyPath:           strings.TrimSpace(f.fields[serverFieldKeyPath].Value()),
		PasswordRef:       strings.TrimSpace(f.fields[serverFieldPasswordRef].Value()),
		PassphraseRef:     strings.TrimSpace(f.fields[serverFieldPassphraseRef].Value()),
		Desc:              strings.TrimSpace(f.fields[serverFieldDesc].Value()),
		DefaultRemotePath: strings.TrimSpace(f.fields[serverFieldDefaultRemotePath].Value()),
		Tags:              splitTags(f.fields[serverFieldTags].Value()),
	}
	now := time.Now()
	srv.CreatedAt = now
	srv.UpdatedAt = now
	if srv.Env == "" {
		srv.Env = "default"
	}
	if srv.DefaultRemotePath == "" {
		srv.DefaultRemotePath = defaultRemotePathForUser(srv.User)
	}
	return serverFormValue{
		Server:     srv,
		Password:   f.fields[serverFieldPasswordSecret].Value(),
		Passphrase: f.fields[serverFieldPassphraseSecret].Value(),
	}, nil
}

func (f serverForm) visibleFields() []int {
	fields := []int{
		serverFieldName,
		serverFieldEnv,
		serverFieldHost,
		serverFieldPort,
		serverFieldUser,
		serverFieldAuthType,
	}
	switch f.authType() {
	case config.AuthKey:
		fields = append(fields, serverFieldKeyPath, serverFieldPassphraseSecret)
	case config.AuthPassword:
		fields = append(fields, serverFieldPasswordSecret)
	}
	fields = append(fields, serverFieldDesc, serverFieldDefaultRemotePath, serverFieldTags)
	return fields
}

func (f serverForm) isVisible(index int) bool {
	for _, field := range f.visibleFields() {
		if field == index {
			return true
		}
	}
	return false
}

func (f serverForm) nextVisibleIndex(direction int) int {
	if direction == 0 {
		return f.index
	}
	index := f.index
	for i := 0; i < len(f.fields); i++ {
		index += direction
		if index < 0 {
			index = len(f.fields) - 1
		}
		if index >= len(f.fields) {
			index = 0
		}
		if f.isVisible(index) {
			return index
		}
	}
	return f.index
}

func (f serverForm) lastVisibleIndex() int {
	fields := f.visibleFields()
	return fields[len(fields)-1]
}

func (f serverForm) authType() string {
	return defaultString(f.fields[serverFieldAuthType].Value(), config.AuthAgent)
}

func (f serverForm) dirty() bool {
	if len(f.initialValues) != len(f.fields) {
		return true
	}
	for i, field := range f.fields {
		if field.Value() != f.initialValues[i] {
			return true
		}
	}
	return false
}

func (f *serverForm) cycleAuthType(direction int) {
	options := []string{config.AuthAgent, config.AuthKey, config.AuthPassword}
	current := f.authType()
	idx := 0
	for i, option := range options {
		if option == current {
			idx = i
			break
		}
	}
	idx += direction
	if idx < 0 {
		idx = len(options) - 1
	}
	if idx >= len(options) {
		idx = 0
	}
	f.fields[serverFieldAuthType].SetValue(options[idx])
	if !f.isVisible(f.index) {
		f.focusNext()
	}
}

func (f *serverForm) setUser(value string) {
	oldUser := f.fields[serverFieldUser].Value()
	oldDefault := defaultRemotePathForUser(oldUser)
	f.fields[serverFieldUser].SetValue(value)
	currentRemote := strings.TrimSpace(f.fields[serverFieldDefaultRemotePath].Value())
	if !f.remotePathManual || currentRemote == "" || currentRemote == oldDefault {
		f.fields[serverFieldDefaultRemotePath].SetValue(defaultRemotePathForUser(value))
		f.remotePathManual = false
	}
}

func defaultRemotePathForUser(user string) string {
	user = strings.TrimSpace(user)
	if user == "" {
		return "/tmp"
	}
	if user == "root" {
		return "/root"
	}
	return "/home/" + user
}

func splitTags(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	tags := make([]string, 0, len(parts))
	for _, part := range parts {
		tag := strings.TrimSpace(part)
		if tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags
}

func defaultPort(port int) int {
	if port <= 0 {
		return 22
	}
	return port
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func uniqueServerID(servers []config.Server, srv config.Server) string {
	base := slug(defaultString(srv.Name, srv.Host))
	if base == "" {
		base = "server"
	}
	used := map[string]bool{}
	for _, existing := range servers {
		used[existing.ID] = true
	}
	if !used[base] {
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if !used[candidate] {
			return candidate
		}
	}
}

func slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	dash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
			continue
		}
		if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func indexServerByID(servers []config.Server, id string) int {
	for i, srv := range servers {
		if srv.ID == id {
			return i
		}
	}
	return 0
}

func findServerByID(servers []config.Server, id string) (config.Server, bool) {
	for _, srv := range servers {
		if srv.ID == id {
			return srv, true
		}
	}
	return config.Server{}, false
}

func clampCursor(cursor, length int) int {
	if length <= 0 {
		return 0
	}
	if cursor < 0 {
		return 0
	}
	if cursor >= length {
		return length - 1
	}
	return cursor
}

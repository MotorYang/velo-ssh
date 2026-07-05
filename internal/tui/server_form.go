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
		defaultString(srv.DefaultRemotePath, "/tmp"),
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
	fields[0].Focus()
	return serverForm{
		mode:       mode,
		originalID: srv.ID,
		fields:     fields,
		index:      0,
	}
}

func (f *serverForm) focusNext() {
	f.fields[f.index].Blur()
	if f.index < len(f.fields)-1 {
		f.index++
	} else {
		f.index = 0
	}
	f.fields[f.index].Focus()
}

func (f *serverForm) focusPrev() {
	f.fields[f.index].Blur()
	if f.index > 0 {
		f.index--
	} else {
		f.index = len(f.fields) - 1
	}
	f.fields[f.index].Focus()
}

func (f serverForm) server() (serverFormValue, error) {
	port, err := strconv.Atoi(strings.TrimSpace(f.fields[serverFieldPort].Value()))
	if err != nil {
		return serverFormValue{}, fmt.Errorf("port must be a number")
	}
	srv := config.Server{
		ID:                strings.TrimSpace(f.fields[serverFieldID].Value()),
		Name:              strings.TrimSpace(f.fields[serverFieldName].Value()),
		Env:               strings.TrimSpace(f.fields[serverFieldEnv].Value()),
		Host:              strings.TrimSpace(f.fields[serverFieldHost].Value()),
		Port:              port,
		User:              strings.TrimSpace(f.fields[serverFieldUser].Value()),
		AuthType:          strings.TrimSpace(f.fields[serverFieldAuthType].Value()),
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
		srv.DefaultRemotePath = "/tmp"
	}
	return serverFormValue{
		Server:     srv,
		Password:   f.fields[serverFieldPasswordSecret].Value(),
		Passphrase: f.fields[serverFieldPassphraseSecret].Value(),
	}, nil
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

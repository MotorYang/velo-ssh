package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/motoryang/velo-ssh/internal/config"
)

const (
	settingsFieldDefaultViewMode = iota
	settingsFieldASCIIFallback
	settingsFieldFallbackRemotePath
	settingsFieldDraftTTLDays
	settingsFieldTransferConcurrency
	settingsFieldKeepAliveSeconds
	settingsFieldTheme
	settingsFieldLanguage
	settingsFieldConfirmOverwrite
	settingsFieldKnownHostsPolicy
	settingsFieldCheckUpdates
)

var settingsFormLabels = []string{
	"Default View Mode",
	"ASCII Fallback",
	"Fallback Remote Path",
	"Draft TTL Days",
	"Transfer Concurrency",
	"KeepAlive Seconds",
	"Theme",
	"Language",
	"Confirm Overwrite",
	"Known Hosts Policy",
	"Check Updates",
}

var settingsFieldOptions = map[int][]string{
	settingsFieldDefaultViewMode:  {config.ViewSingle, config.ViewSplit},
	settingsFieldASCIIFallback:    {config.ASCIIFallbackAuto, config.ASCIIFallbackAlways, config.ASCIIFallbackDisabled},
	settingsFieldLanguage:         {config.LanguageEnglish, config.LanguageSimplifiedChinese},
	settingsFieldConfirmOverwrite: {"true", "false"},
	settingsFieldKnownHostsPolicy: {config.HostKeyStrict, config.HostKeyAsk, config.HostKeyInsecure},
	settingsFieldCheckUpdates:     {"true", "false"},
}

type settingsForm struct {
	fields []textinput.Model
	index  int
}

func newSettingsForm(settings config.Settings) settingsForm {
	values := []string{
		defaultString(settings.DefaultViewMode, config.ViewSingle),
		defaultString(settings.ASCIIFallback, config.ASCIIFallbackAuto),
		defaultString(settings.FallbackRemotePath, "/tmp"),
		strconv.Itoa(defaultInt(settings.DraftTTLDays, 30)),
		strconv.Itoa(defaultInt(settings.TransferConcurrency, 4)),
		strconv.Itoa(defaultInt(settings.KeepAliveSeconds, 20)),
		defaultString(settings.Theme, "default"),
		defaultString(settings.Language, config.LanguageEnglish),
		strconv.FormatBool(settings.ConfirmOverwrite),
		defaultString(settings.KnownHostsPolicy, config.HostKeyAsk),
		strconv.FormatBool(!settings.DisableUpdateCheck),
	}
	fields := make([]textinput.Model, len(settingsFormLabels))
	for i := range fields {
		fields[i] = textinput.New()
		fields[i].Prompt = ""
		fields[i].Placeholder = settingsFormLabels[i]
		fields[i].SetValue(values[i])
		fields[i].CharLimit = 256
		fields[i].Width = 34
	}
	fields[0].Focus()
	return settingsForm{fields: fields}
}

func (f *settingsForm) focusNext() {
	f.blurCurrent()
	if f.index < f.focusCount()-1 {
		f.index++
	} else {
		f.index = 0
	}
	f.focusCurrent()
}

func (f *settingsForm) focusPrev() {
	f.blurCurrent()
	if f.index > 0 {
		f.index--
	} else {
		f.index = f.focusCount() - 1
	}
	f.focusCurrent()
}

func (f settingsForm) focusCount() int {
	return len(f.fields) + 2
}

func (f settingsForm) okIndex() int {
	return len(f.fields)
}

func (f settingsForm) cancelIndex() int {
	return len(f.fields) + 1
}

func (f settingsForm) fieldFocused() bool {
	return f.index >= 0 && f.index < len(f.fields)
}

func (f settingsForm) optionFocused() bool {
	if !f.fieldFocused() {
		return false
	}
	_, ok := settingsFieldOptions[f.index]
	return ok
}

func (f settingsForm) inputFocused() bool {
	return f.fieldFocused() && !f.optionFocused()
}

func (f settingsForm) okFocused() bool {
	return f.index == f.okIndex()
}

func (f settingsForm) cancelFocused() bool {
	return f.index == f.cancelIndex()
}

func (f *settingsForm) blurCurrent() {
	if f.fieldFocused() {
		f.fields[f.index].Blur()
	}
}

func (f *settingsForm) focusCurrent() {
	if f.inputFocused() {
		f.fields[f.index].Focus()
	}
}

func (f *settingsForm) cycleOption(delta int) {
	if !f.optionFocused() {
		return
	}
	options := settingsFieldOptions[f.index]
	current := strings.TrimSpace(f.fields[f.index].Value())
	currentIndex := 0
	for i, option := range options {
		if current == option {
			currentIndex = i
			break
		}
	}
	next := (currentIndex + delta) % len(options)
	if next < 0 {
		next += len(options)
	}
	f.fields[f.index].SetValue(options[next])
}

func (f settingsForm) settings() (config.Settings, error) {
	defaultViewMode := strings.TrimSpace(f.fields[settingsFieldDefaultViewMode].Value())
	if !oneOf(defaultViewMode, config.ViewSingle, config.ViewSplit) {
		return config.Settings{}, fmt.Errorf("defaultViewMode must be single or split")
	}
	asciiFallback := strings.TrimSpace(f.fields[settingsFieldASCIIFallback].Value())
	if !oneOf(asciiFallback, config.ASCIIFallbackAuto, config.ASCIIFallbackAlways, config.ASCIIFallbackDisabled) {
		return config.Settings{}, fmt.Errorf("asciiFallback must be auto, always, or disabled")
	}
	fallbackRemotePath := strings.TrimSpace(f.fields[settingsFieldFallbackRemotePath].Value())
	if fallbackRemotePath == "" {
		return config.Settings{}, fmt.Errorf("fallbackRemotePath cannot be empty")
	}
	draftTTLDays, err := positiveInt(f.fields[settingsFieldDraftTTLDays].Value(), "draftTTLDays")
	if err != nil {
		return config.Settings{}, err
	}
	transferConcurrency, err := positiveInt(f.fields[settingsFieldTransferConcurrency].Value(), "transferConcurrency")
	if err != nil {
		return config.Settings{}, err
	}
	keepAliveSeconds, err := positiveInt(f.fields[settingsFieldKeepAliveSeconds].Value(), "keepAliveSeconds")
	if err != nil {
		return config.Settings{}, err
	}
	theme := strings.TrimSpace(f.fields[settingsFieldTheme].Value())
	if theme == "" {
		return config.Settings{}, fmt.Errorf("theme cannot be empty")
	}
	language := strings.TrimSpace(f.fields[settingsFieldLanguage].Value())
	if !oneOf(language, config.LanguageEnglish, config.LanguageSimplifiedChinese) {
		return config.Settings{}, fmt.Errorf("language must be en or zh-CN")
	}
	confirmOverwrite, err := parseBool(f.fields[settingsFieldConfirmOverwrite].Value(), "confirmOverwrite")
	if err != nil {
		return config.Settings{}, err
	}
	knownHostsPolicy := strings.TrimSpace(f.fields[settingsFieldKnownHostsPolicy].Value())
	if !oneOf(knownHostsPolicy, config.HostKeyStrict, config.HostKeyAsk, config.HostKeyInsecure) {
		return config.Settings{}, fmt.Errorf("knownHostsPolicy must be strict, ask, or insecure")
	}
	checkUpdates, err := parseBool(f.fields[settingsFieldCheckUpdates].Value(), "checkUpdates")
	if err != nil {
		return config.Settings{}, err
	}
	return config.Settings{
		DefaultViewMode:     defaultViewMode,
		ASCIIFallback:       asciiFallback,
		FallbackRemotePath:  fallbackRemotePath,
		DraftTTLDays:        draftTTLDays,
		TransferConcurrency: transferConcurrency,
		KeepAliveSeconds:    keepAliveSeconds,
		Theme:               theme,
		Language:            language,
		ConfirmOverwrite:    confirmOverwrite,
		KnownHostsPolicy:    knownHostsPolicy,
		DisableUpdateCheck:  !checkUpdates,
	}, nil
}

func positiveInt(value, name string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("%s must be a positive number", name)
	}
	return n, nil
}

func parseBool(value, name string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "yes", "y", "1", "on":
		return true, nil
	case "false", "no", "n", "0", "off":
		return false, nil
	default:
		return false, fmt.Errorf("%s must be true or false", name)
	}
}

func oneOf(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}

func defaultInt(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

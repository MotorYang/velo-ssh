package config

import "time"

const (
	Version = 1

	AuthKey      = "key"
	AuthPassword = "password"
	AuthAgent    = "agent"

	ViewSplit  = "split"
	ViewSingle = "single"

	ASCIIFallbackAuto     = "auto"
	ASCIIFallbackAlways   = "always"
	ASCIIFallbackDisabled = "disabled"

	HostKeyStrict   = "strict"
	HostKeyAsk      = "ask"
	HostKeyInsecure = "insecure"

	LanguageEnglish           = "en"
	LanguageSimplifiedChinese = "zh-CN"
)

type File struct {
	Version  int      `json:"version"`
	Settings Settings `json:"settings"`
	Servers  []Server `json:"servers"`
}

type Settings struct {
	DefaultViewMode     string `json:"defaultViewMode"`
	ASCIIFallback       string `json:"asciiFallback"`
	FallbackRemotePath  string `json:"fallbackRemotePath"`
	DraftTTLDays        int    `json:"draftTTLDays"`
	TransferConcurrency int    `json:"transferConcurrency"`
	KeepAliveSeconds    int    `json:"keepAliveSeconds"`
	Theme               string `json:"theme"`
	Language            string `json:"language"`
	ConfirmOverwrite    bool   `json:"confirmOverwrite"`
	KnownHostsPolicy    string `json:"knownHostsPolicy"`
}

type Server struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Env               string    `json:"env"`
	Host              string    `json:"host"`
	Port              int       `json:"port"`
	User              string    `json:"user"`
	AuthType          string    `json:"authType"`
	KeyPath           string    `json:"keyPath"`
	PasswordRef       string    `json:"passwordRef"`
	PassphraseRef     string    `json:"passphraseRef"`
	Desc              string    `json:"desc"`
	Tags              []string  `json:"tags"`
	DefaultRemotePath string    `json:"defaultRemotePath"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

func DefaultSettings() Settings {
	return Settings{
		DefaultViewMode:     ViewSingle,
		ASCIIFallback:       ASCIIFallbackAuto,
		FallbackRemotePath:  "/tmp",
		DraftTTLDays:        30,
		TransferConcurrency: 4,
		KeepAliveSeconds:    20,
		Theme:               "default",
		Language:            LanguageEnglish,
		ConfirmOverwrite:    true,
		KnownHostsPolicy:    HostKeyAsk,
	}
}

func DefaultFile() File {
	return File{
		Version:  Version,
		Settings: DefaultSettings(),
		Servers:  []Server{},
	}
}

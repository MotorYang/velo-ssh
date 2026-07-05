package config

import (
	"fmt"

	"github.com/zalando/go-keyring"
)

const KeyringService = "vssh"

type SecretStore interface {
	Get(ref string) (string, error)
	Set(ref, value string) error
	Delete(ref string) error
}

type OSKeyring struct{}

func PasswordRef(serverID string) string {
	return fmt.Sprintf("vssh:%s:password", serverID)
}

func PassphraseRef(serverID string) string {
	return fmt.Sprintf("vssh:%s:passphrase", serverID)
}

func (OSKeyring) Get(ref string) (string, error) {
	return keyring.Get(KeyringService, ref)
}

func (OSKeyring) Set(ref, value string) error {
	return keyring.Set(KeyringService, ref, value)
}

func (OSKeyring) Delete(ref string) error {
	return keyring.Delete(KeyringService, ref)
}

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

const KeyringService = "vssh"

type SecretStore interface {
	Get(ref string) (string, error)
	Set(ref, value string) error
	Delete(ref string) error
}

type OSKeyring struct{}

type FallbackSecretStore struct {
	Primary  SecretStore
	Fallback SecretStore
}

type FileSecretStore struct {
	Path string
}

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

func NewSecretStore(fallbackPath string) SecretStore {
	return FallbackSecretStore{
		Primary:  OSKeyring{},
		Fallback: FileSecretStore{Path: fallbackPath},
	}
}

func (s FallbackSecretStore) Get(ref string) (string, error) {
	if s.Primary != nil {
		value, err := s.Primary.Get(ref)
		if err == nil {
			return value, nil
		}
	}
	if s.Fallback == nil {
		return "", fmt.Errorf("secret %q not found", ref)
	}
	return s.Fallback.Get(ref)
}

func (s FallbackSecretStore) Set(ref, value string) error {
	if s.Primary != nil {
		if err := s.Primary.Set(ref, value); err == nil {
			return nil
		}
	}
	if s.Fallback == nil {
		return fmt.Errorf("store secret %q: no fallback secret store configured", ref)
	}
	return s.Fallback.Set(ref, value)
}

func (s FallbackSecretStore) Delete(ref string) error {
	var primaryErr error
	if s.Primary != nil {
		primaryErr = s.Primary.Delete(ref)
	}
	if s.Fallback != nil {
		if err := s.Fallback.Delete(ref); err != nil {
			return err
		}
	}
	return primaryErr
}

func (s FileSecretStore) Get(ref string) (string, error) {
	values, err := s.load()
	if err != nil {
		return "", err
	}
	value, ok := values[ref]
	if !ok {
		return "", fmt.Errorf("secret %q not found in fallback store", ref)
	}
	return value, nil
}

func (s FileSecretStore) Set(ref, value string) error {
	values, err := s.load()
	if err != nil {
		return err
	}
	values[ref] = value
	return s.save(values)
}

func (s FileSecretStore) Delete(ref string) error {
	values, err := s.load()
	if err != nil {
		return err
	}
	delete(values, ref)
	return s.save(values)
}

func (s FileSecretStore) load() (map[string]string, error) {
	if s.Path == "" {
		return nil, fmt.Errorf("fallback secret store path is empty")
	}
	data, err := os.ReadFile(s.Path)
	if os.IsNotExist(err) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	values := map[string]string{}
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, err
	}
	return values, nil
}

func (s FileSecretStore) save(values map[string]string) error {
	if s.Path == "" {
		return fmt.Errorf("fallback secret store path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		return err
	}
	if err := os.Chmod(filepath.Dir(s.Path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(values, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(s.Path), "."+filepath.Base(s.Path)+".tmp.")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, s.Path)
}

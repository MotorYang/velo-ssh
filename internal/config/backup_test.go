package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestExportBackupOmitsSecretsByDefault(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "config"))
	secretStore := FileSecretStore{Path: filepath.Join(dir, "secrets.json")}
	cfg := DefaultFile()
	cfg.Servers = []Server{{
		ID:          "prod",
		Name:        "Prod",
		Host:        "example.com",
		Port:        22,
		User:        "root",
		AuthType:    AuthPassword,
		PasswordRef: PasswordRef("prod"),
	}}
	if err := store.Save(cfg); err != nil {
		t.Fatal(err)
	}
	if err := secretStore.Set(PasswordRef("prod"), "secret"); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(dir, "backup.json")
	if err := ExportBackup(store, secretStore, output, false); err != nil {
		t.Fatal(err)
	}
	var backup BackupFile
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &backup); err != nil {
		t.Fatal(err)
	}
	if len(backup.Secrets) != 0 {
		t.Fatalf("secrets exported by default: %#v", backup.Secrets)
	}
	info, err := os.Stat(output)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("backup mode = %v, want 0600", got)
	}
}

func TestExportImportBackupWithSecrets(t *testing.T) {
	dir := t.TempDir()
	source := NewStore(filepath.Join(dir, "source"))
	sourceSecrets := FileSecretStore{Path: filepath.Join(dir, "source-secrets.json")}
	cfg := DefaultFile()
	cfg.Servers = []Server{{
		ID:          "prod",
		Name:        "Prod",
		Host:        "example.com",
		Port:        22,
		User:        "root",
		AuthType:    AuthPassword,
		PasswordRef: PasswordRef("prod"),
	}}
	if err := source.Save(cfg); err != nil {
		t.Fatal(err)
	}
	if err := sourceSecrets.Set(PasswordRef("prod"), "secret"); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(dir, "backup.json")
	if err := ExportBackup(source, sourceSecrets, output, true); err != nil {
		t.Fatal(err)
	}
	target := NewStore(filepath.Join(dir, "target"))
	targetSecrets := FileSecretStore{Path: filepath.Join(dir, "target-secrets.json")}
	if err := ImportBackup(target, targetSecrets, output); err != nil {
		t.Fatal(err)
	}
	imported, err := target.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(imported.Servers) != 1 || imported.Servers[0].ID != "prod" {
		t.Fatalf("imported servers = %#v", imported.Servers)
	}
	secret, err := targetSecrets.Get(PasswordRef("prod"))
	if err != nil {
		t.Fatal(err)
	}
	if secret != "secret" {
		t.Fatalf("imported secret = %q", secret)
	}
}

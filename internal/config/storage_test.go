package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreLoadMissingReturnsDefaults(t *testing.T) {
	store := NewStore(t.TempDir())
	cfg, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Version != Version {
		t.Fatalf("version = %d, want %d", cfg.Version, Version)
	}
	if cfg.Settings.DefaultViewMode != ViewSingle {
		t.Fatalf("default view mode = %q", cfg.Settings.DefaultViewMode)
	}
	if cfg.Settings.Language != LanguageEnglish {
		t.Fatalf("default language = %q, want en", cfg.Settings.Language)
	}
	if cfg.Settings.DisableUpdateCheck {
		t.Fatalf("update checks should be enabled by default")
	}
	if len(cfg.Servers) != 0 {
		t.Fatalf("servers = %d, want 0", len(cfg.Servers))
	}
}

func TestStoreSaveCreatesPrivateFile(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	if err := store.Save(DefaultFile()); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(store.ConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("config mode = %v, want 0600", got)
	}
	dirInfo, err := os.Stat(filepath.Dir(store.ConfigPath()))
	if err != nil {
		t.Fatal(err)
	}
	if got := dirInfo.Mode().Perm(); got&0o077 != 0 {
		t.Fatalf("config dir mode too open: %v", got)
	}
}

func TestUpsertFindDeleteServer(t *testing.T) {
	store := NewStore(t.TempDir())
	srv := Server{
		ID:       "prod-web-01",
		Name:     "Prod-Web-01",
		Env:      "prod",
		Host:     "10.0.0.1",
		Port:     22,
		User:     "root",
		AuthType: AuthAgent,
	}
	if err := store.UpsertServer(srv); err != nil {
		t.Fatal(err)
	}
	found, err := store.FindServer("Prod-Web-01")
	if err != nil {
		t.Fatal(err)
	}
	if found.ID != srv.ID {
		t.Fatalf("found id = %q, want %q", found.ID, srv.ID)
	}
	if err := store.DeleteServer(srv.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.FindServer(srv.ID); err == nil {
		t.Fatal("expected deleted server lookup to fail")
	}
}

func TestKeyringRefs(t *testing.T) {
	if PasswordRef("abc") != "vssh:abc:password" {
		t.Fatalf("unexpected password ref")
	}
	if PassphraseRef("abc") != "vssh:abc:passphrase" {
		t.Fatalf("unexpected passphrase ref")
	}
}

func TestFileSecretStoreStoresPrivateFallbackFile(t *testing.T) {
	store := FileSecretStore{Path: filepath.Join(t.TempDir(), "secrets.json")}
	if err := store.Set("ref", "secret"); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get("ref")
	if err != nil {
		t.Fatal(err)
	}
	if got != "secret" {
		t.Fatalf("secret = %q", got)
	}
	info, err := os.Stat(store.Path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("fallback secret file mode = %v, want 0600", got)
	}
}

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const BackupVersion = 1

type BackupFile struct {
	Version    int               `json:"version"`
	CreatedAt  time.Time         `json:"createdAt"`
	Config     File              `json:"config"`
	Secrets    map[string]string `json:"secrets,omitempty"`
	SecretNote string            `json:"secretNote,omitempty"`
}

func ExportBackup(store *Store, secrets SecretStore, outputPath string, includeSecrets bool) error {
	if outputPath == "" {
		return fmt.Errorf("export backup: output path is required")
	}
	cfg, err := store.Load()
	if err != nil {
		return fmt.Errorf("export backup: load config: %w", err)
	}
	backup := BackupFile{
		Version:   BackupVersion,
		CreatedAt: time.Now().UTC(),
		Config:    cfg,
	}
	if includeSecrets {
		backup.SecretNote = "Secret values are stored in plaintext in this backup file. Protect or delete this file after import."
		backup.Secrets = map[string]string{}
		for _, ref := range secretRefs(cfg) {
			value, err := secrets.Get(ref)
			if err != nil {
				return fmt.Errorf("export backup: read secret %q: %w", ref, err)
			}
			backup.Secrets[ref] = value
		}
	}
	return writeBackupAtomic(outputPath, backup)
}

func ImportBackup(store *Store, secrets SecretStore, inputPath string) error {
	if inputPath == "" {
		return fmt.Errorf("import backup: input path is required")
	}
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("import backup: read %s: %w", inputPath, err)
	}
	var backup BackupFile
	if err := json.Unmarshal(data, &backup); err != nil {
		return fmt.Errorf("import backup: parse %s: %w", inputPath, err)
	}
	if backup.Version == 0 {
		return fmt.Errorf("import backup: unsupported backup version 0")
	}
	normalizeFile(&backup.Config)
	for _, srv := range backup.Config.Servers {
		if err := ValidateServer(srv); err != nil {
			return fmt.Errorf("import backup: invalid server %q: %w", srv.ID, err)
		}
	}
	if err := store.Save(backup.Config); err != nil {
		return fmt.Errorf("import backup: save config: %w", err)
	}
	for ref, value := range backup.Secrets {
		if err := secrets.Set(ref, value); err != nil {
			return fmt.Errorf("import backup: store secret %q: %w", ref, err)
		}
	}
	return nil
}

func secretRefs(cfg File) []string {
	seen := map[string]bool{}
	var refs []string
	for _, srv := range cfg.Servers {
		for _, ref := range []string{srv.PasswordRef, srv.PassphraseRef} {
			if ref == "" || seen[ref] {
				continue
			}
			seen[ref] = true
			refs = append(refs, ref)
		}
	}
	return refs
}

func BackupPath(path string) string {
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return path
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

func writeBackupAtomic(path string, v any) error {
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp.")
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
	return os.Rename(tmpName, path)
}

package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/scrypt"
)

const BackupVersion = 1

type BackupFile struct {
	Version    int               `json:"version"`
	CreatedAt  time.Time         `json:"createdAt"`
	Config     File              `json:"config"`
	Secrets    map[string]string `json:"secrets,omitempty"`
	SecretNote string            `json:"secretNote,omitempty"`
}

type EncryptedBackupFile struct {
	Version    int              `json:"version"`
	CreatedAt  time.Time        `json:"createdAt"`
	Encryption BackupEncryption `json:"encryption"`
	Payload    string           `json:"payload"`
}

type BackupEncryption struct {
	Algorithm string `json:"algorithm"`
	KDF       string `json:"kdf"`
	Salt      string `json:"salt"`
	Nonce     string `json:"nonce"`
	N         int    `json:"n"`
	R         int    `json:"r"`
	P         int    `json:"p"`
	KeyLen    int    `json:"keyLen"`
}

func ExportBackup(store *Store, secrets SecretStore, outputPath string, includeSecrets bool) error {
	return ExportBackupWithPassphrase(store, secrets, outputPath, includeSecrets, "")
}

func ExportBackupWithPassphrase(store *Store, secrets SecretStore, outputPath string, includeSecrets bool, passphrase string) error {
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
		if passphrase == "" {
			backup.SecretNote = "Secret values are stored in plaintext in this backup file. Protect or delete this file after import."
		}
		backup.Secrets = map[string]string{}
		for _, ref := range secretRefs(cfg) {
			value, err := secrets.Get(ref)
			if err != nil {
				return fmt.Errorf("export backup: read secret %q: %w", ref, err)
			}
			backup.Secrets[ref] = value
		}
	}
	if passphrase != "" {
		encrypted, err := encryptBackup(backup, passphrase)
		if err != nil {
			return err
		}
		return writeBackupAtomic(outputPath, encrypted)
	}
	return writeBackupAtomic(outputPath, backup)
}

func ImportBackup(store *Store, secrets SecretStore, inputPath string) error {
	return ImportBackupWithPassphrase(store, secrets, inputPath, "")
}

func ImportBackupWithPassphrase(store *Store, secrets SecretStore, inputPath string, passphrase string) error {
	if inputPath == "" {
		return fmt.Errorf("import backup: input path is required")
	}
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("import backup: read %s: %w", inputPath, err)
	}
	backup, err := decodeBackup(data, passphrase)
	if err != nil {
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

func decodeBackup(data []byte, passphrase string) (BackupFile, error) {
	var encrypted EncryptedBackupFile
	if err := json.Unmarshal(data, &encrypted); err == nil && encrypted.Encryption.Algorithm != "" {
		if passphrase == "" {
			return BackupFile{}, fmt.Errorf("backup is encrypted; passphrase is required")
		}
		return decryptBackup(encrypted, passphrase)
	}
	var backup BackupFile
	if err := json.Unmarshal(data, &backup); err != nil {
		return BackupFile{}, err
	}
	return backup, nil
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

func encryptBackup(backup BackupFile, passphrase string) (EncryptedBackupFile, error) {
	plain, err := json.Marshal(backup)
	if err != nil {
		return EncryptedBackupFile{}, err
	}
	salt := make([]byte, 16)
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return EncryptedBackupFile{}, err
	}
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return EncryptedBackupFile{}, err
	}
	const n, r, p, keyLen = 32768, 8, 1, 32
	key, err := scrypt.Key([]byte(passphrase), salt, n, r, p, keyLen)
	if err != nil {
		return EncryptedBackupFile{}, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return EncryptedBackupFile{}, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return EncryptedBackupFile{}, err
	}
	ciphertext := aead.Seal(nil, nonce, plain, nil)
	return EncryptedBackupFile{
		Version:   BackupVersion,
		CreatedAt: time.Now().UTC(),
		Encryption: BackupEncryption{
			Algorithm: "AES-256-GCM",
			KDF:       "scrypt",
			Salt:      base64.StdEncoding.EncodeToString(salt),
			Nonce:     base64.StdEncoding.EncodeToString(nonce),
			N:         n,
			R:         r,
			P:         p,
			KeyLen:    keyLen,
		},
		Payload: base64.StdEncoding.EncodeToString(ciphertext),
	}, nil
}

func decryptBackup(encrypted EncryptedBackupFile, passphrase string) (BackupFile, error) {
	if encrypted.Encryption.Algorithm != "AES-256-GCM" || encrypted.Encryption.KDF != "scrypt" {
		return BackupFile{}, fmt.Errorf("unsupported backup encryption %s/%s", encrypted.Encryption.Algorithm, encrypted.Encryption.KDF)
	}
	salt, err := base64.StdEncoding.DecodeString(encrypted.Encryption.Salt)
	if err != nil {
		return BackupFile{}, err
	}
	nonce, err := base64.StdEncoding.DecodeString(encrypted.Encryption.Nonce)
	if err != nil {
		return BackupFile{}, err
	}
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted.Payload)
	if err != nil {
		return BackupFile{}, err
	}
	keyLen := encrypted.Encryption.KeyLen
	if keyLen == 0 {
		keyLen = 32
	}
	key, err := scrypt.Key([]byte(passphrase), salt, encrypted.Encryption.N, encrypted.Encryption.R, encrypted.Encryption.P, keyLen)
	if err != nil {
		return BackupFile{}, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return BackupFile{}, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return BackupFile{}, err
	}
	plain, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return BackupFile{}, fmt.Errorf("decrypt backup: %w", err)
	}
	var backup BackupFile
	if err := json.Unmarshal(plain, &backup); err != nil {
		return BackupFile{}, err
	}
	return backup, nil
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

package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Store struct {
	dir string
}

func NewStore(dir string) *Store {
	return &Store{dir: dir}
}

func DefaultDir() (string, error) {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfg, "vssh"), nil
}

func (s *Store) ConfigPath() string {
	return filepath.Join(s.dir, "config.json")
}

func (s *Store) DraftsPath() string {
	return filepath.Join(s.dir, "drafts.json")
}

func (s *Store) Load() (File, error) {
	path := s.ConfigPath()
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return DefaultFile(), nil
	}
	if err != nil {
		return File{}, err
	}
	var f File
	if err := json.Unmarshal(b, &f); err != nil {
		return File{}, err
	}
	normalizeFile(&f)
	return f, nil
}

func (s *Store) Save(f File) error {
	normalizeFile(&f)
	return writeJSONAtomic(s.ConfigPath(), f)
}

func (s *Store) LoadDrafts() (DraftFile, error) {
	path := s.DraftsPath()
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return DefaultDraftFile(), nil
	}
	if err != nil {
		return DraftFile{}, err
	}
	var f DraftFile
	if err := json.Unmarshal(b, &f); err != nil {
		return DraftFile{}, err
	}
	if f.Version == 0 {
		f.Version = Version
	}
	if f.Drafts == nil {
		f.Drafts = []Draft{}
	}
	return f, nil
}

func (s *Store) SaveDrafts(f DraftFile) error {
	if f.Version == 0 {
		f.Version = Version
	}
	if f.Drafts == nil {
		f.Drafts = []Draft{}
	}
	return writeJSONAtomic(s.DraftsPath(), f)
}

func (s *Store) FindServer(query string) (Server, error) {
	f, err := s.Load()
	if err != nil {
		return Server{}, err
	}
	var matches []Server
	for _, srv := range f.Servers {
		if srv.ID == query || strings.EqualFold(srv.Name, query) {
			matches = append(matches, srv)
		}
	}
	if len(matches) == 0 {
		return Server{}, fmt.Errorf("server %q not found", query)
	}
	if len(matches) > 1 {
		return Server{}, fmt.Errorf("server %q is ambiguous; matched %d entries", query, len(matches))
	}
	return matches[0], nil
}

func (s *Store) UpsertServer(srv Server) error {
	f, err := s.Load()
	if err != nil {
		return err
	}
	now := time.Now()
	if srv.CreatedAt.IsZero() {
		srv.CreatedAt = now
	}
	srv.UpdatedAt = now
	if err := ValidateServer(srv); err != nil {
		return err
	}
	for i := range f.Servers {
		if f.Servers[i].ID == srv.ID {
			f.Servers[i] = srv
			return s.Save(f)
		}
	}
	f.Servers = append(f.Servers, srv)
	return s.Save(f)
}

func (s *Store) DeleteServer(id string) error {
	f, err := s.Load()
	if err != nil {
		return err
	}
	out := f.Servers[:0]
	found := false
	for _, srv := range f.Servers {
		if srv.ID == id {
			found = true
			continue
		}
		out = append(out, srv)
	}
	if !found {
		return fmt.Errorf("server %q not found", id)
	}
	f.Servers = out
	return s.Save(f)
}

func ValidateServer(srv Server) error {
	if strings.TrimSpace(srv.ID) == "" {
		return errors.New("server id is required")
	}
	if strings.TrimSpace(srv.Name) == "" {
		return errors.New("server name is required")
	}
	if strings.TrimSpace(srv.Host) == "" {
		return errors.New("server host is required")
	}
	if strings.TrimSpace(srv.User) == "" {
		return errors.New("server user is required")
	}
	if srv.Port <= 0 {
		return errors.New("server port must be positive")
	}
	switch srv.AuthType {
	case AuthKey, AuthPassword, AuthAgent:
	default:
		return fmt.Errorf("unsupported auth type %q", srv.AuthType)
	}
	return nil
}

func normalizeFile(f *File) {
	if f.Version == 0 {
		f.Version = Version
	}
	if f.Settings.DefaultViewMode == "" {
		f.Settings = DefaultSettings()
	}
	if f.Servers == nil {
		f.Servers = []Server{}
	}
	for i := range f.Servers {
		if f.Servers[i].Port == 0 {
			f.Servers[i].Port = 22
		}
		if f.Servers[i].AuthType == "" {
			f.Servers[i].AuthType = AuthAgent
		}
	}
}

func writeJSONAtomic(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if err := os.Chmod(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp.")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(b); err != nil {
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

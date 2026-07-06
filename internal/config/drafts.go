package config

import (
	"fmt"
	"time"
)

const (
	DraftPending  = "pending"
	DraftSyncing  = "syncing"
	DraftFailed   = "failed"
	DraftResolved = "resolved"
	DraftExpired  = "expired"
)

type DraftFile struct {
	Version int     `json:"version"`
	Drafts  []Draft `json:"drafts"`
}

type Draft struct {
	ID              string    `json:"id"`
	ServerID        string    `json:"serverId"`
	RemotePath      string    `json:"remotePath"`
	LocalPath       string    `json:"localPath"`
	BaseRemoteSize  int64     `json:"baseRemoteSize"`
	BaseRemoteMTime time.Time `json:"baseRemoteMTime"`
	LocalSHA256     string    `json:"localSHA256"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

func DefaultDraftFile() DraftFile {
	return DraftFile{Version: Version, Drafts: []Draft{}}
}

func (s *Store) UpsertDraft(draft Draft) error {
	f, err := s.LoadDrafts()
	if err != nil {
		return err
	}
	now := time.Now()
	if draft.ID == "" {
		return fmt.Errorf("draft id is required")
	}
	if draft.CreatedAt.IsZero() {
		draft.CreatedAt = now
	}
	draft.UpdatedAt = now
	if draft.Status == "" {
		draft.Status = DraftPending
	}
	for i := range f.Drafts {
		if f.Drafts[i].ID == draft.ID {
			f.Drafts[i] = draft
			return s.SaveDrafts(f)
		}
	}
	f.Drafts = append(f.Drafts, draft)
	return s.SaveDrafts(f)
}

func (s *Store) UpdateDraftStatus(id, status string) error {
	f, err := s.LoadDrafts()
	if err != nil {
		return err
	}
	for i := range f.Drafts {
		if f.Drafts[i].ID == id {
			f.Drafts[i].Status = status
			f.Drafts[i].UpdatedAt = time.Now()
			return s.SaveDrafts(f)
		}
	}
	return fmt.Errorf("draft %q not found", id)
}

func (s *Store) PruneExpiredDrafts(ttlDays int) (int, error) {
	if ttlDays <= 0 {
		return 0, nil
	}
	f, err := s.LoadDrafts()
	if err != nil {
		return 0, err
	}
	cutoff := time.Now().AddDate(0, 0, -ttlDays)
	out := f.Drafts[:0]
	removed := 0
	for _, draft := range f.Drafts {
		if draft.UpdatedAt.Before(cutoff) {
			draft.Status = DraftExpired
			removed++
			continue
		}
		out = append(out, draft)
	}
	f.Drafts = out
	if removed == 0 {
		return 0, nil
	}
	return removed, s.SaveDrafts(f)
}

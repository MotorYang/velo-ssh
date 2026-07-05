package config

import "time"

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

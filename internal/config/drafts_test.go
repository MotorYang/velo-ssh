package config

import (
	"testing"
	"time"
)

func TestUpsertDraftAndUpdateStatus(t *testing.T) {
	store := NewStore(t.TempDir())
	draft := Draft{ID: "d1", ServerID: "srv", RemotePath: "/etc/app.conf", LocalPath: "/tmp/app.conf"}
	if err := store.UpsertDraft(draft); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateDraftStatus("d1", DraftResolved); err != nil {
		t.Fatal(err)
	}
	f, err := store.LoadDrafts()
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Drafts) != 1 || f.Drafts[0].Status != DraftResolved {
		t.Fatalf("drafts = %#v", f.Drafts)
	}
}

func TestPruneExpiredDrafts(t *testing.T) {
	store := NewStore(t.TempDir())
	old := Draft{
		ID:        "old",
		Status:    DraftFailed,
		CreatedAt: time.Now().AddDate(0, 0, -40),
		UpdatedAt: time.Now().AddDate(0, 0, -40),
	}
	fresh := Draft{
		ID:        "fresh",
		Status:    DraftFailed,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := store.SaveDrafts(DraftFile{Version: Version, Drafts: []Draft{old, fresh}}); err != nil {
		t.Fatal(err)
	}
	removed, err := store.PruneExpiredDrafts(30)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Fatalf("removed = %d, want 1", removed)
	}
	f, err := store.LoadDrafts()
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Drafts) != 1 || f.Drafts[0].ID != "fresh" {
		t.Fatalf("drafts = %#v", f.Drafts)
	}
}

package resolver

import (
	"path/filepath"
	"testing"
	"time"
)

func TestSnapshotPathUsesDefaultGroup(t *testing.T) {
	got := SnapshotPath("/tmp/root", "", time.Date(2026, 3, 15, 14, 22, 0, 0, time.UTC))
	want := filepath.Join("/tmp/root", SnapshotDir, "default.2026-03-15T14:22:00Z.lock")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestSaveAndLoadSnapshotRoundTrip(t *testing.T) {
	root := t.TempDir()
	snapshot := Layer3Snapshot{
		Group:     "research-team",
		Timestamp: time.Date(2026, 3, 15, 14, 22, 0, 0, time.UTC),
		Packages:  map[string]string{"requests": "2.31.0"},
		Backend:   "cpu",
		MLflowIDs: []string{"abc"},
	}
	if err := SaveSnapshot(root, snapshot); err != nil {
		t.Fatalf("save: %v", err)
	}

	path := SnapshotPath(root, snapshot.Group, snapshot.Timestamp)
	got, err := LoadSnapshot(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.Group != snapshot.Group || !got.Timestamp.Equal(snapshot.Timestamp) || got.Packages["requests"] != "2.31.0" || got.Backend != "cpu" {
		t.Fatalf("round trip mismatch: %#v", got)
	}
}

func TestLatestSnapshotReturnsNewest(t *testing.T) {
	root := t.TempDir()
	older := Layer3Snapshot{Group: "team", Timestamp: time.Date(2026, 3, 15, 14, 20, 0, 0, time.UTC), Packages: map[string]string{"a": "1"}}
	newer := Layer3Snapshot{Group: "team", Timestamp: time.Date(2026, 3, 15, 14, 22, 0, 0, time.UTC), Packages: map[string]string{"a": "2"}}
	if err := SaveSnapshot(root, older); err != nil {
		t.Fatalf("save older: %v", err)
	}
	if err := SaveSnapshot(root, newer); err != nil {
		t.Fatalf("save newer: %v", err)
	}

	got, err := LatestSnapshot(root, "team")
	if err != nil {
		t.Fatalf("latest: %v", err)
	}
	if got.Packages["a"] != "2" {
		t.Fatalf("latest snapshot is not newest: %#v", got)
	}
}

func TestPreviousSnapshotErrorsOnSingleSnapshot(t *testing.T) {
	root := t.TempDir()
	snapshot := Layer3Snapshot{Group: "team", Timestamp: time.Date(2026, 3, 15, 14, 20, 0, 0, time.UTC), Packages: map[string]string{"a": "1"}}
	if err := SaveSnapshot(root, snapshot); err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, err := PreviousSnapshot(root, "team"); err == nil {
		t.Fatal("expected error")
	}
}

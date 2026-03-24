package resolver

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// SnapshotPath returns the path to a snapshot file for a group and timestamp.
func SnapshotPath(workspaceRoot, group string, t time.Time) string {
	return filepath.Join(workspaceRoot, SnapshotDir, fmt.Sprintf("%s.%s.lock", snapshotGroup(group), t.UTC().Format(time.RFC3339)))
}

// SaveSnapshot writes a snapshot atomically to disk.
func SaveSnapshot(workspaceRoot string, snapshot Layer3Snapshot) error {
	dir := filepath.Join(workspaceRoot, SnapshotDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create snapshot dir: %w", err)
	}

	path := SnapshotPath(workspaceRoot, snapshot.Group, snapshot.Timestamp)
	temp, err := os.CreateTemp(dir, ".snapshot-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp snapshot: %w", err)
	}

	cleanup := func() {
		_ = temp.Close()
		_ = os.Remove(temp.Name())
	}

	encoder := json.NewEncoder(temp)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(snapshot); err != nil {
		cleanup()
		return fmt.Errorf("encode snapshot: %w", err)
	}
	if err := temp.Sync(); err != nil {
		cleanup()
		return fmt.Errorf("sync snapshot: %w", err)
	}
	if err := temp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close snapshot: %w", err)
	}
	if err := os.Rename(temp.Name(), path); err != nil {
		cleanup()
		return fmt.Errorf("move snapshot into place: %w", err)
	}
	return nil
}

// LoadSnapshot reads a snapshot file from disk.
func LoadSnapshot(path string) (Layer3Snapshot, error) {
	file, err := os.Open(path)
	if err != nil {
		return Layer3Snapshot{}, fmt.Errorf("open snapshot: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return Layer3Snapshot{}, fmt.Errorf("read snapshot: %w", err)
	}

	var snapshot Layer3Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return Layer3Snapshot{}, fmt.Errorf("parse snapshot: %w", err)
	}
	if snapshot.Packages == nil {
		snapshot.Packages = map[string]string{}
	}
	return snapshot, nil
}

// ListSnapshots returns all snapshots for a group, sorted newest first.
func ListSnapshots(workspaceRoot, group string) ([]string, error) {
	dir := filepath.Join(workspaceRoot, SnapshotDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read snapshot dir: %w", err)
	}

	groupKey := snapshotGroup(group)
	type snapshotFile struct {
		path string
		t    time.Time
	}
	files := make([]snapshotFile, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".lock") {
			continue
		}
		if !strings.HasPrefix(entry.Name(), groupKey+".") {
			continue
		}
		timestamp, ok := snapshotTimeFromName(entry.Name(), groupKey)
		if !ok {
			continue
		}
		files = append(files, snapshotFile{
			path: filepath.Join(dir, entry.Name()),
			t:    timestamp,
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].t.After(files[j].t)
	})

	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.path)
	}
	return paths, nil
}

// LatestSnapshot returns the newest snapshot for a group.
func LatestSnapshot(workspaceRoot, group string) (Layer3Snapshot, error) {
	paths, err := ListSnapshots(workspaceRoot, group)
	if err != nil {
		return Layer3Snapshot{}, err
	}
	if len(paths) == 0 {
		return Layer3Snapshot{}, fmt.Errorf("no snapshots found for group %q", group)
	}
	return LoadSnapshot(paths[0])
}

// PreviousSnapshot returns the second-newest snapshot for a group.
func PreviousSnapshot(workspaceRoot, group string) (Layer3Snapshot, error) {
	paths, err := ListSnapshots(workspaceRoot, group)
	if err != nil {
		return Layer3Snapshot{}, err
	}
	if len(paths) < 2 {
		return Layer3Snapshot{}, fmt.Errorf("fewer than two snapshots found for group %q", group)
	}
	return LoadSnapshot(paths[1])
}

// SnapshotCount returns the total number of snapshots on disk.
func SnapshotCount(workspaceRoot string) (int, error) {
	dir := filepath.Join(workspaceRoot, SnapshotDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, fmt.Errorf("read snapshot dir: %w", err)
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".lock") {
			continue
		}
		count++
	}
	return count, nil
}

// snapshotGroup returns the filename-safe snapshot group.
func snapshotGroup(group string) string {
	group = strings.TrimSpace(group)
	if group == "" {
		return "default"
	}
	group = strings.ToLower(group)
	builder := strings.Builder{}
	builder.Grow(len(group))
	for _, r := range group {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '-' || r == '_':
			builder.WriteRune(r)
		default:
			builder.WriteByte('-')
		}
	}
	return strings.Trim(builder.String(), "-_")
}

func snapshotTimeFromName(name, groupKey string) (time.Time, bool) {
	prefix := groupKey + "."
	if !strings.HasPrefix(name, prefix) {
		return time.Time{}, false
	}
	body := strings.TrimSuffix(strings.TrimPrefix(name, prefix), ".lock")
	timestamp, err := time.Parse(time.RFC3339, body)
	if err != nil {
		return time.Time{}, false
	}
	return timestamp, true
}

package resolver

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// BaselineDir is the directory, relative to the workspace root, where
// promoted group baselines are stored.
const BaselineDir = "config/env-baselines"

// BaselineSummary is the persisted baseline metadata for one group.
type BaselineSummary struct {
	Group     string    `json:"group"`
	Timestamp time.Time `json:"timestamp"`
	Packages  int       `json:"packages"`
	Path      string    `json:"path,omitempty"`
}

// BaselinePath returns the canonical baseline path for a group.
func BaselinePath(workspaceRoot, group string) string {
	return filepath.Join(workspaceRoot, BaselineDir, fmt.Sprintf("%s.json", snapshotGroup(group)))
}

// LoadBaseline returns the promoted baseline snapshot for a group.
func LoadBaseline(workspaceRoot, group string) (Layer3Snapshot, error) {
	return LoadSnapshot(BaselinePath(workspaceRoot, group))
}

// SaveBaseline writes a promoted baseline snapshot for a group.
func SaveBaseline(workspaceRoot string, snapshot Layer3Snapshot) error {
	dir := filepath.Join(workspaceRoot, BaselineDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create baseline dir: %w", err)
	}

	path := BaselinePath(workspaceRoot, snapshot.Group)
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("encode baseline: %w", err)
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(dir, ".baseline-*.tmp")
	if err != nil {
		return fmt.Errorf("create baseline temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write baseline temp file: %w", err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return fmt.Errorf("chmod baseline temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close baseline temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace baseline: %w", err)
	}
	return nil
}

// BaselineSummaries returns all promoted baselines sorted by group.
func BaselineSummaries(workspaceRoot string) ([]BaselineSummary, error) {
	dir := filepath.Join(workspaceRoot, BaselineDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []BaselineSummary{}, nil
		}
		return nil, fmt.Errorf("read baseline dir: %w", err)
	}

	summaries := make([]BaselineSummary, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		snapshot, err := LoadSnapshot(path)
		if err != nil {
			continue
		}
		summaries = append(summaries, BaselineSummary{
			Group:     snapshot.Group,
			Timestamp: snapshot.Timestamp,
			Packages:  len(snapshot.Packages),
			Path:      path,
		})
	}
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Group < summaries[j].Group
	})
	return summaries, nil
}

// PromoteBaseline saves the selected snapshot as the active baseline.
func PromoteBaseline(workspaceRoot, group, timestamp string) (Layer3Snapshot, error) {
	group = strings.TrimSpace(group)
	if group == "" {
		group = "default"
	}

	var snapshot Layer3Snapshot
	var err error
	if strings.TrimSpace(timestamp) == "" {
		snapshot, err = LatestSnapshot(workspaceRoot, group)
	} else {
		snapshot, err = snapshotByTimestamp(workspaceRoot, group, timestamp)
	}
	if err != nil {
		return Layer3Snapshot{}, err
	}
	if err := SaveBaseline(workspaceRoot, snapshot); err != nil {
		return Layer3Snapshot{}, err
	}
	return snapshot, nil
}

func snapshotByTimestamp(workspaceRoot, group, timestamp string) (Layer3Snapshot, error) {
	paths, err := ListSnapshots(workspaceRoot, group)
	if err != nil {
		return Layer3Snapshot{}, err
	}
	for _, path := range paths {
		snapshot, err := LoadSnapshot(path)
		if err != nil {
			continue
		}
		if snapshot.Timestamp.UTC().Format(time.RFC3339) == timestamp {
			return snapshot, nil
		}
	}
	return Layer3Snapshot{}, fmt.Errorf("no snapshot found for group %q at %s", group, timestamp)
}


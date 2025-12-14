package cache

import (
	"encoding/json"
	"os"
	"path/filepath"

	"codetracker-hooks/internal/diff"
	"codetracker-hooks/internal/scanner"
)

// CachedSnapshot holds the last snapshot state
type CachedSnapshot struct {
	SnapshotID string                          `json:"snapshot_id,omitempty"`
	Files      map[string]*diff.SnapshotFileInfo `json:"files"`
}

// LoadLastSnapshot loads the last snapshot from cache file
func LoadLastSnapshot(cacheFile string) (*CachedSnapshot, error) {
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, err
	}

	var snapshot CachedSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		// Try to load as old format (map of files without wrapper)
		var files map[string]*diff.SnapshotFileInfo
		if err := json.Unmarshal(data, &files); err != nil {
			return nil, err
		}
		return &CachedSnapshot{Files: files}, nil
	}

	return &snapshot, nil
}

// SaveLastSnapshot saves the current file state to cache
func SaveLastSnapshot(cacheFile string, files map[string]*scanner.FileInfo, snapshotID string) error {
	// Ensure directory exists
	dir := filepath.Dir(cacheFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Convert to snapshot format
	snapshotFiles := make(map[string]*diff.SnapshotFileInfo)
	for path, info := range files {
		snapshotFiles[path] = &diff.SnapshotFileInfo{
			Hash: info.Hash,
			Size: info.Size,
		}
	}

	snapshot := &CachedSnapshot{
		SnapshotID: snapshotID,
		Files:      snapshotFiles,
	}

	jsonData, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFile, jsonData, 0644)
}

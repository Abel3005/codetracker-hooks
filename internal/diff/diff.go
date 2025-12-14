package diff

import (
	"codetracker-hooks/internal/scanner"
)

// ChangeType represents the type of file change
type ChangeType string

const (
	Added    ChangeType = "A"
	Modified ChangeType = "M"
	Deleted  ChangeType = "D"
)

// Change represents a file change
type Change struct {
	FilePath     string     `json:"file_path"`
	Type         ChangeType `json:"type"`
	Hash         string     `json:"hash,omitempty"`
	Content      string     `json:"content,omitempty"`
	Size         int64      `json:"size,omitempty"`
	PreviousHash string     `json:"previous_hash,omitempty"`
}

// SnapshotFileInfo holds cached file info from previous snapshot
type SnapshotFileInfo struct {
	Hash string `json:"hash"`
	Size int64  `json:"size"`
}

// CalculateChanges compares current files with previous snapshot
func CalculateChanges(currentFiles map[string]*scanner.FileInfo, previousSnapshot map[string]*SnapshotFileInfo) []*Change {
	changes := make([]*Change, 0)

	// First snapshot: all files are new
	if previousSnapshot == nil {
		for _, info := range currentFiles {
			changes = append(changes, &Change{
				FilePath: info.RelativePath,
				Type:     Added,
				Hash:     info.Hash,
				Content:  info.Content,
				Size:     info.Size,
			})
		}
		return changes
	}

	// Detect added and modified files
	for filePath, info := range currentFiles {
		prevFile, exists := previousSnapshot[filePath]

		if !exists {
			// New file
			changes = append(changes, &Change{
				FilePath: filePath,
				Type:     Added,
				Hash:     info.Hash,
				Content:  info.Content,
				Size:     info.Size,
			})
		} else if prevFile.Hash != info.Hash {
			// Modified file
			changes = append(changes, &Change{
				FilePath:     filePath,
				Type:         Modified,
				Hash:         info.Hash,
				Content:      info.Content,
				Size:         info.Size,
				PreviousHash: prevFile.Hash,
			})
		}
		// Unchanged files are skipped
	}

	// Detect deleted files
	for filePath, prevInfo := range previousSnapshot {
		if _, exists := currentFiles[filePath]; !exists {
			changes = append(changes, &Change{
				FilePath:     filePath,
				Type:         Deleted,
				PreviousHash: prevInfo.Hash,
			})
		}
	}

	return changes
}

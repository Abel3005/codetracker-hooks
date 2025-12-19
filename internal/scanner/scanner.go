package scanner

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"

	"codetracker-hooks/internal/config"
	"codetracker-hooks/internal/gitignore"
)

// FileInfo holds information about a tracked file
type FileInfo struct {
	RelativePath string
	Hash         string
	Content      string
	Size         int64
}

// Scanner scans project files
type Scanner struct {
	projectRoot     string
	ignoreMatcher   *gitignore.Matcher
	trackExtensions map[string]bool
	maxFileSize     int64
}

// NewScanner creates a new Scanner
func NewScanner(projectRoot string, cfg *config.Config) (*Scanner, error) {
	ignoreMatcher, err := gitignore.NewMatcher(cfg.IgnorePatterns)
	if err != nil {
		return nil, err
	}

	trackExtensions := make(map[string]bool)
	for _, ext := range cfg.TrackExtensions {
		trackExtensions[ext] = true
	}

	return &Scanner{
		projectRoot:     projectRoot,
		ignoreMatcher:   ignoreMatcher,
		trackExtensions: trackExtensions,
		maxFileSize:     cfg.MaxFileSize,
	}, nil
}

// shouldIgnorePath checks if a path should be ignored
func (s *Scanner) shouldIgnorePath(absPath string) bool {
	relativePath, err := filepath.Rel(s.projectRoot, absPath)
	if err != nil {
		return true
	}
	basename := filepath.Base(absPath)

	// Always ignore dotfiles and dotdirs (starting with .)
	if strings.HasPrefix(basename, ".") {
		return true
	}

	return s.ignoreMatcher.ShouldIgnore(relativePath, basename)
}

// shouldTrackFile checks if a file should be tracked
func (s *Scanner) shouldTrackFile(absPath string) bool {
	if s.shouldIgnorePath(absPath) {
		return false
	}
	ext := filepath.Ext(absPath)
	return s.trackExtensions[ext]
}

// calculateHash computes SHA256 hash of content
func calculateHash(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

// Scan scans all tracked files and returns a map
func (s *Scanner) Scan() (map[string]*FileInfo, error) {
	trackedFiles := make(map[string]*FileInfo)

	err := filepath.Walk(s.projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip files/dirs with errors
			return nil
		}

		// Skip ignored directories
		if info.IsDir() {
			if s.shouldIgnorePath(path) {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file should be tracked
		if !s.shouldTrackFile(path) {
			return nil
		}

		// Skip files exceeding max size
		if info.Size() > s.maxFileSize {
			return nil
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			// Skip files with read errors
			return nil
		}

		relativePath, err := filepath.Rel(s.projectRoot, path)
		if err != nil {
			return nil
		}

		// Normalize path separators to forward slash
		relativePath = strings.ReplaceAll(relativePath, "\\", "/")

		trackedFiles[relativePath] = &FileInfo{
			RelativePath: relativePath,
			Hash:         calculateHash(content),
			Content:      string(content),
			Size:         info.Size(),
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return trackedFiles, nil
}

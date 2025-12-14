package config

import (
	"os"
	"path/filepath"
)

var projectRoot string

// GetProjectRoot returns the project root directory
func GetProjectRoot() string {
	if projectRoot != "" {
		return projectRoot
	}

	if dir := os.Getenv("CLAUDE_PROJECT_DIR"); dir != "" {
		projectRoot = dir
		return projectRoot
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	projectRoot = cwd
	return projectRoot
}

// TrackerDir returns the .codetracker directory path
func TrackerDir() string {
	return filepath.Join(GetProjectRoot(), ".codetracker")
}

// ConfigFile returns the config.json file path
func ConfigFile() string {
	return filepath.Join(TrackerDir(), "config.json")
}

// CredentialsFile returns the credentials.json file path
func CredentialsFile() string {
	return filepath.Join(TrackerDir(), "credentials.json")
}

// CacheDir returns the cache directory path
func CacheDir() string {
	return filepath.Join(TrackerDir(), "cache")
}

// LastSnapshotFile returns the last_snapshot.json file path
func LastSnapshotFile() string {
	return filepath.Join(CacheDir(), "last_snapshot.json")
}

// SessionFile returns the current_session.json file path
func SessionFile() string {
	return filepath.Join(CacheDir(), "current_session.json")
}

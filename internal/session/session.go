package session

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// SessionData holds the session state between hooks
type SessionData struct {
	PreSnapshotID   string `json:"pre_snapshot_id"`
	Prompt          string `json:"prompt"`
	ClaudeSessionID string `json:"claude_session_id"`
	StartedAt       string `json:"started_at"`
}

// Save saves session data to file
func Save(sessionFile string, data *SessionData) error {
	// Ensure directory exists
	dir := filepath.Dir(sessionFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(sessionFile, jsonData, 0644)
}

// Load loads session data from file
func Load(sessionFile string) (*SessionData, error) {
	data, err := os.ReadFile(sessionFile)
	if err != nil {
		return nil, err
	}

	var session SessionData
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

// Delete removes the session file
func Delete(sessionFile string) error {
	return os.Remove(sessionFile)
}

package main

import (
	"encoding/json"
	"io"
	"os"
	"time"

	"codetracker-hooks/internal/api"
	"codetracker-hooks/internal/cache"
	"codetracker-hooks/internal/config"
	"codetracker-hooks/internal/diff"
	"codetracker-hooks/internal/scanner"
	"codetracker-hooks/internal/session"
)

// HookInput represents the input from Claude Code
type HookInput struct {
	Timestamp string `json:"timestamp"`
}

func main() {
	// Always exit with success to never block Claude
	defer func() {
		if r := recover(); r != nil {
			// Ignore panics
		}
		os.Exit(0)
	}()

	if err := run(); err != nil {
		// Silent fail
		return
	}
}

func run() error {
	// Read input from stdin
	inputData, err := io.ReadAll(os.Stdin)
	if err != nil {
		return err
	}

	var input HookInput
	if err := json.Unmarshal(inputData, &input); err != nil {
		return err
	}

	timestamp := input.Timestamp
	if timestamp == "" {
		timestamp = time.Now().UTC().Format(time.RFC3339)
	}

	// Load session data from pre-prompt hook
	sessionData, err := session.Load(config.SessionFile())
	if err != nil || sessionData == nil {
		// No session - pre-prompt snapshot wasn't created
		return nil
	}

	// Load config and credentials
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	creds, err := config.LoadCredentials()
	if err != nil || !creds.IsValid() {
		return err
	}

	// Scan files and calculate changes
	projectRoot := config.GetProjectRoot()
	s, err := scanner.NewScanner(projectRoot, cfg)
	if err != nil {
		return err
	}

	currentFiles, err := s.Scan()
	if err != nil {
		return err
	}

	// Load previous snapshot
	var prevFiles map[string]*diff.SnapshotFileInfo
	lastSnapshot, _ := cache.LoadLastSnapshot(config.LastSnapshotFile())
	if lastSnapshot != nil {
		prevFiles = lastSnapshot.Files
	}

	changes := diff.CalculateChanges(currentFiles, prevFiles)

	// Check only_on_changes setting
	if len(changes) == 0 && cfg.AutoSnapshot.OnlyOnChanges {
		// No changes and configured to skip - clean up and exit
		session.Delete(config.SessionFile())
		return nil
	}

	// Create interaction on server
	client := api.NewClient(cfg.ServerURL, creds.APIKey)

	req := &api.CreateInteractionRequest{
		ProjectHash:      creds.CurrentProjectHash,
		Message:          "[AUTO-POST] " + sessionData.Prompt,
		Changes:          changes,
		ParentSnapshotID: sessionData.PreSnapshotID,
		ClaudeSessionID:  sessionData.ClaudeSessionID,
		StartedAt:        sessionData.StartedAt,
		EndedAt:          timestamp,
	}

	resp, err := client.CreateInteraction(req)
	if err != nil {
		return err
	}

	// Save last snapshot cache
	snapshotID := resp.SnapshotID
	if snapshotID == "" {
		snapshotID = sessionData.PreSnapshotID
	}
	if err := cache.SaveLastSnapshot(config.LastSnapshotFile(), currentFiles, snapshotID); err != nil {
		return err
	}

	// Clean up session file
	session.Delete(config.SessionFile())

	return nil
}

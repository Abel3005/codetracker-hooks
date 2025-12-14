package main

import (
	"encoding/json"
	"io"
	"os"
	"regexp"
	"strings"
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
	Prompt    string `json:"prompt"`
	SessionID string `json:"session_id"`
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

	// Skip empty prompts
	if strings.TrimSpace(input.Prompt) == "" {
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

	// Check if auto-snapshot is enabled
	if !cfg.AutoSnapshot.Enabled {
		return nil
	}

	// Check skip patterns
	for _, pattern := range cfg.AutoSnapshot.SkipPatterns {
		re, err := regexp.Compile("(?i)" + pattern)
		if err != nil {
			continue
		}
		if re.MatchString(input.Prompt) {
			return nil
		}
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

	// Create snapshot on server
	client := api.NewClient(cfg.ServerURL, creds.APIKey)

	req := &api.CreateSnapshotRequest{
		ProjectHash:     creds.CurrentProjectHash,
		Message:         "[AUTO-PRE] " + input.Prompt,
		Changes:         changes,
		ClaudeSessionID: input.SessionID,
	}

	if lastSnapshot != nil && lastSnapshot.SnapshotID != "" {
		req.ParentSnapshotID = lastSnapshot.SnapshotID
	}

	resp, err := client.CreateSnapshot(req)
	if err != nil {
		return err
	}

	// Save last snapshot cache
	if err := cache.SaveLastSnapshot(config.LastSnapshotFile(), currentFiles, resp.SnapshotID); err != nil {
		return err
	}

	// Save session data for stop hook
	timestamp := input.Timestamp
	if timestamp == "" {
		timestamp = time.Now().UTC().Format(time.RFC3339)
	}

	sessionData := &session.SessionData{
		PreSnapshotID:   resp.SnapshotID,
		Prompt:          input.Prompt,
		ClaudeSessionID: input.SessionID,
		StartedAt:       timestamp,
	}

	return session.Save(config.SessionFile(), sessionData)
}

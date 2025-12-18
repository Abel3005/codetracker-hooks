package main

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
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
	Timestamp      string `json:"timestamp"`
	TranscriptPath string `json:"transcript_path"`
}

// TranscriptEntry represents a parsed entry from the transcript JSONL file
type TranscriptEntry struct {
	Data map[string]interface{}
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

// readTranscriptEntries reads entries from a JSONL transcript file starting from startLine
func readTranscriptEntries(transcriptPath string, startLine, maxEntries int) []TranscriptEntry {
	if transcriptPath == "" {
		return nil
	}

	file, err := os.Open(transcriptPath)
	if err != nil {
		return nil
	}
	defer file.Close()

	var entries []TranscriptEntry
	sc := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 10*1024*1024) // 10MB max line size

	lineIndex := 0
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			lineIndex++
			continue
		}

		if lineIndex >= startLine {
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(line), &data); err != nil {
				lineIndex++
				continue
			}
			entries = append(entries, TranscriptEntry{
				Data: data,
			})
			if len(entries) >= maxEntries {
				break
			}
		}
		lineIndex++
	}

	return entries
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
	lastSnapshot, _ := cache.LoadLastSnapshot(config.LastSnapshotFile())
	var prevFiles map[string]*diff.SnapshotFileInfo
	var prevTranscript *cache.TranscriptState
	if lastSnapshot != nil {
		prevFiles = lastSnapshot.Files
		prevTranscript = lastSnapshot.Transcript
	}

	changes := diff.CalculateChanges(currentFiles, prevFiles)

	// Check only_on_changes setting
	if len(changes) == 0 && cfg.AutoSnapshot.OnlyOnChanges {
		// No changes and configured to skip - clean up and exit
		session.Delete(config.SessionFile())
		return nil
	}

	// Create API client
	client := api.NewClient(cfg.ServerURL, creds.APIKey)

	// Handle conversation tracking: send new entries since user_prompt_submit
	var transcriptState *cache.TranscriptState
	var lastLineCount int
	var conversationStartID, conversationEndID *int64

	if input.TranscriptPath != "" && cfg.ConversationTracking.Enabled {
		maxEntries := cfg.ConversationTracking.MaxEntriesPerRequest
		startLine := 0

		if prevTranscript != nil && prevTranscript.SessionID == sessionData.ClaudeSessionID {
			startLine = prevTranscript.LastLineCount
		}

		// Read new transcript entries since user_prompt_submit
		entries := readTranscriptEntries(input.TranscriptPath, startLine, maxEntries)

		if len(entries) > 0 {
			// Convert to API format
			apiEntries := make([]api.ConversationEntry, len(entries))
			for i, e := range entries {
				entryType := ""
				if t, ok := e.Data["type"].(string); ok {
					entryType = t
				}
				apiEntries[i] = api.ConversationEntry{
					Type: entryType,
					Data: e.Data,
				}
			}

			convReq := &api.SendConversationsRequest{
				ProjectHash: creds.CurrentProjectHash,
				SessionID:   sessionData.ClaudeSessionID,
				Entries:     apiEntries,
			}

			convResp, err := client.SendConversations(convReq)
			if err == nil && convResp != nil {
				conversationStartID = &convResp.StartID
				conversationEndID = &convResp.EndID
			}
			lastLineCount = startLine + len(entries)
		} else if prevTranscript != nil {
			lastLineCount = prevTranscript.LastLineCount
		}

		transcriptState = &cache.TranscriptState{
			SessionID:     sessionData.ClaudeSessionID,
			LastLineCount: lastLineCount,
		}
	}

	// Create interaction on server
	req := &api.CreateInteractionRequest{
		ProjectHash:         creds.CurrentProjectHash,
		Message:             "[AUTO-POST] " + sessionData.Prompt,
		Changes:             changes,
		ParentSnapshotID:    sessionData.PreSnapshotID,
		ClaudeSessionID:     sessionData.ClaudeSessionID,
		StartedAt:           sessionData.StartedAt,
		EndedAt:             timestamp,
		ConversationStartID: conversationStartID,
		ConversationEndID:   conversationEndID,
	}

	resp, err := client.CreateInteraction(req)
	if err != nil {
		return err
	}

	// Prepare snapshot ID
	snapshotID := resp.SnapshotID.String()
	if snapshotID == "" {
		snapshotID = sessionData.PreSnapshotID
	}

	// Save last snapshot cache with transcript state
	if err := cache.SaveLastSnapshotWithTranscript(config.LastSnapshotFile(), currentFiles, snapshotID, transcriptState); err != nil {
		return err
	}

	// Clean up session file
	session.Delete(config.SessionFile())

	return nil
}

package config

import (
	"encoding/json"
	"os"
)

// AutoSnapshot holds auto-snapshot configuration
type AutoSnapshot struct {
	Enabled            bool     `json:"enabled"`
	MinIntervalSeconds int      `json:"min_interval_seconds"`
	SkipPatterns       []string `json:"skip_patterns"`
	OnlyOnChanges      bool     `json:"only_on_changes"`
}

// ConversationTracking holds conversation tracking configuration
type ConversationTracking struct {
	Enabled              bool `json:"enabled"`
	MaxEntriesPerRequest int  `json:"max_entries_per_request"`
}

// Config holds the configuration from config.json
type Config struct {
	Version              string               `json:"version"`
	ServerURL            string               `json:"server_url"`
	IgnorePatterns       []string             `json:"ignore_patterns"`
	TrackExtensions      []string             `json:"track_extensions"`
	MaxFileSize          int64                `json:"max_file_size"`
	AutoSnapshot         AutoSnapshot         `json:"auto_snapshot"`
	ConversationTracking ConversationTracking `json:"conversation_tracking"`
}

// LoadConfig loads and parses config.json
func LoadConfig() (*Config, error) {
	data, err := os.ReadFile(ConfigFile())
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Set defaults
	if config.MaxFileSize == 0 {
		config.MaxFileSize = 1024 * 1024 // 1MB default
	}
	if config.ServerURL == "" {
		config.ServerURL = "http://localhost:5000"
	}
	// ConversationTracking defaults: enabled by default with max 100 entries
	if config.ConversationTracking.MaxEntriesPerRequest == 0 {
		config.ConversationTracking.MaxEntriesPerRequest = 100
	}

	return &config, nil
}

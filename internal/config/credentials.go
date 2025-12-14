package config

import (
	"encoding/json"
	"os"
)

// Credentials holds the credentials from credentials.json
type Credentials struct {
	APIKey             string `json:"api_key"`
	Username           string `json:"username"`
	Email              string `json:"email"`
	CurrentProjectHash string `json:"current_project_hash"`
}

// LoadCredentials loads and parses credentials.json
func LoadCredentials() (*Credentials, error) {
	data, err := os.ReadFile(CredentialsFile())
	if err != nil {
		return nil, err
	}

	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}

	return &creds, nil
}

// IsValid checks if credentials have required fields
func (c *Credentials) IsValid() bool {
	return c.APIKey != "" && c.CurrentProjectHash != ""
}

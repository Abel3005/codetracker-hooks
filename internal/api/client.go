package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"codetracker-hooks/internal/diff"
)

// Client is the HTTP client for CodeTracker API
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new API client
func NewClient(serverURL, apiKey string) *Client {
	return &Client{
		baseURL: serverURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// doRequest performs an HTTP request with common headers
func (c *Client) doRequest(method, endpoint string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, c.baseURL+endpoint, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// CreateSnapshotRequest is the request body for creating a snapshot
type CreateSnapshotRequest struct {
	ProjectHash      string         `json:"project_hash"`
	Message          string         `json:"message"`
	Changes          []*diff.Change `json:"changes"`
	ClaudeSessionID  string         `json:"claude_session_id,omitempty"`
	ParentSnapshotID string         `json:"parent_snapshot_id,omitempty"`
}

// CreateSnapshotResponse is the response from creating a snapshot
type CreateSnapshotResponse struct {
	SnapshotID string `json:"snapshot_id"`
	CreatedAt  string `json:"created_at"`
}

// CreateSnapshot creates a new snapshot
func (c *Client) CreateSnapshot(req *CreateSnapshotRequest) (*CreateSnapshotResponse, error) {
	respBody, err := c.doRequest("POST", "/api/snapshots", req)
	if err != nil {
		return nil, err
	}

	var resp CreateSnapshotResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// CreateInteractionRequest is the request body for creating an interaction
type CreateInteractionRequest struct {
	ProjectHash      string         `json:"project_hash"`
	Message          string         `json:"message"`
	Changes          []*diff.Change `json:"changes"`
	ParentSnapshotID string         `json:"parent_snapshot_id"`
	ClaudeSessionID  string         `json:"claude_session_id"`
	StartedAt        string         `json:"started_at"`
	EndedAt          string         `json:"ended_at"`
}

// CreateInteractionResponse is the response from creating an interaction
type CreateInteractionResponse struct {
	SnapshotID string `json:"snapshot_id"`
}

// CreateInteraction creates a new interaction record
func (c *Client) CreateInteraction(req *CreateInteractionRequest) (*CreateInteractionResponse, error) {
	respBody, err := c.doRequest("POST", "/api/interactions", req)
	if err != nil {
		return nil, err
	}

	var resp CreateInteractionResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

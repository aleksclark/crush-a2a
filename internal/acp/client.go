package acp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

// Client talks to the Crush ACP backend.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	Logger     *slog.Logger
}

// NewClient creates a new ACP client.
func NewClient(baseURL string, logger *slog.Logger) *Client {
	return &Client{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{},
		Logger:     logger,
	}
}

// Ping checks ACP availability.
func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/ping", nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ping: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// ListAgents returns all available agents.
func (c *Client) ListAgents(ctx context.Context) ([]AgentManifest, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/agents", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list agents: unexpected status %d", resp.StatusCode)
	}
	var result AgentsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Agents, nil
}

// GetAgent retrieves a single agent manifest.
func (c *Client) GetAgent(ctx context.Context, name string) (*AgentManifest, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/agents/"+name, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get agent %q: unexpected status %d", name, resp.StatusCode)
	}
	var agent AgentManifest
	if err := json.NewDecoder(resp.Body).Decode(&agent); err != nil {
		return nil, err
	}
	return &agent, nil
}

// CreateRun creates a new ACP run (sync mode).
func (c *Client) CreateRun(ctx context.Context, r CreateRunRequest) (*Run, error) {
	body, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/runs", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("create run: status %d: %s", resp.StatusCode, string(respBody))
	}
	var run Run
	if err := json.NewDecoder(resp.Body).Decode(&run); err != nil {
		return nil, err
	}
	return &run, nil
}

// CreateRunStream creates a run in stream mode and returns the NDJSON response body.
// The caller is responsible for closing the returned io.ReadCloser.
func (c *Client) CreateRunStream(ctx context.Context, r CreateRunRequest) (io.ReadCloser, error) {
	r.Mode = "stream"
	body, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/runs", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/x-ndjson")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("create run stream: status %d: %s", resp.StatusCode, string(respBody))
	}
	return resp.Body, nil
}

// GetRun retrieves an existing run by ID.
func (c *Client) GetRun(ctx context.Context, runID string) (*Run, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/runs/"+runID, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get run %q: unexpected status %d", runID, resp.StatusCode)
	}
	var run Run
	if err := json.NewDecoder(resp.Body).Decode(&run); err != nil {
		return nil, err
	}
	return &run, nil
}

// CancelRun cancels a running ACP run.
func (c *Client) CancelRun(ctx context.Context, runID string) (*Run, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/runs/"+runID+"/cancel", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("cancel run %q: unexpected status %d", runID, resp.StatusCode)
	}
	var run Run
	if err := json.NewDecoder(resp.Body).Decode(&run); err != nil {
		return nil, err
	}
	return &run, nil
}

// ReadNDJSON reads NDJSON events from a reader, sending parsed events to the callback.
// It returns when the reader is exhausted or the context is canceled.
func ReadNDJSON(ctx context.Context, r io.Reader, fn func(Event) error) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var ev Event
		if err := json.Unmarshal(line, &ev); err != nil {
			return fmt.Errorf("parse NDJSON event: %w", err)
		}
		if err := fn(ev); err != nil {
			return err
		}
	}
	return scanner.Err()
}

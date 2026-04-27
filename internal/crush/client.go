package crush

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// Client talks to the Crush native HTTP API.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	Logger     *slog.Logger
}

// NewClient creates a Crush client from an address string.
// Supported formats: "tcp://host:port", "unix:///path/to/sock"
func NewClient(addr string, logger *slog.Logger) (*Client, error) {
	proto, rest, ok := strings.Cut(addr, "://")
	if !ok {
		return nil, fmt.Errorf("invalid address format: %s (expected tcp://host:port or unix:///path)", addr)
	}

	c := &Client{Logger: logger}

	switch proto {
	case "tcp":
		parsed, err := url.Parse("http://" + rest)
		if err != nil {
			return nil, fmt.Errorf("invalid tcp address: %w", err)
		}
		c.BaseURL = "http://" + parsed.Host + "/v1"
		c.HTTPClient = &http.Client{}
	case "unix":
		sockPath := rest
		c.BaseURL = "http://crush/v1"
		c.HTTPClient = &http.Client{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return net.Dial("unix", sockPath)
				},
			},
		}
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", proto)
	}

	return c, nil
}

// Health checks if the server is reachable.
func (c *Client) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/health", nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check: status %d", resp.StatusCode)
	}
	return nil
}

// ListWorkspaces returns all workspaces.
func (c *Client) ListWorkspaces(ctx context.Context) ([]Workspace, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/workspaces", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, c.readError(resp)
	}
	var ws []Workspace
	if err := json.NewDecoder(resp.Body).Decode(&ws); err != nil {
		return nil, err
	}
	return ws, nil
}

// CreateWorkspace creates a new workspace with the given path.
func (c *Client) CreateWorkspace(ctx context.Context, path string) (*Workspace, error) {
	body, err := json.Marshal(Workspace{Path: path})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/workspaces", bytes.NewReader(body))
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
		return nil, c.readError(resp)
	}
	var ws Workspace
	if err := json.NewDecoder(resp.Body).Decode(&ws); err != nil {
		return nil, err
	}
	return &ws, nil
}

// GetWorkspace returns a single workspace.
func (c *Client) GetWorkspace(ctx context.Context, id string) (*Workspace, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/workspaces/"+id, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, c.readError(resp)
	}
	var ws Workspace
	if err := json.NewDecoder(resp.Body).Decode(&ws); err != nil {
		return nil, err
	}
	return &ws, nil
}

// CreateSession creates a new session in a workspace.
func (c *Client) CreateSession(ctx context.Context, workspaceID, title string) (*Session, error) {
	body, err := json.Marshal(Session{Title: title})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/workspaces/"+workspaceID+"/sessions", bytes.NewReader(body))
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
		return nil, c.readError(resp)
	}
	var sess Session
	if err := json.NewDecoder(resp.Body).Decode(&sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

// InitAgent initializes the agent for a workspace.
func (c *Client) InitAgent(ctx context.Context, workspaceID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/workspaces/"+workspaceID+"/agent/init", nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return c.readError(resp)
	}
	return nil
}

// SendMessage sends a prompt to the agent.
func (c *Client) SendMessage(ctx context.Context, workspaceID string, msg AgentMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/workspaces/"+workspaceID+"/agent", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return c.readError(resp)
	}
	return nil
}

// GetAgentSession returns the agent session status.
func (c *Client) GetAgentSession(ctx context.Context, workspaceID, sessionID string) (*AgentSession, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.BaseURL+"/workspaces/"+workspaceID+"/agent/sessions/"+sessionID, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, c.readError(resp)
	}
	var as AgentSession
	if err := json.NewDecoder(resp.Body).Decode(&as); err != nil {
		return nil, err
	}
	return &as, nil
}

// CancelSession cancels a running agent session.
func (c *Client) CancelSession(ctx context.Context, workspaceID, sessionID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.BaseURL+"/workspaces/"+workspaceID+"/agent/sessions/"+sessionID+"/cancel", nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return c.readError(resp)
	}
	return nil
}

// GetMessages returns all messages for a session.
func (c *Client) GetMessages(ctx context.Context, workspaceID, sessionID string) ([]Message, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.BaseURL+"/workspaces/"+workspaceID+"/sessions/"+sessionID+"/messages", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, c.readError(resp)
	}
	var msgs []Message
	if err := json.NewDecoder(resp.Body).Decode(&msgs); err != nil {
		return nil, err
	}
	return msgs, nil
}

// SubscribeEvents opens an SSE connection to the workspace events stream.
// The caller must close the returned io.ReadCloser when done.
func (c *Client) SubscribeEvents(ctx context.Context, workspaceID string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.BaseURL+"/workspaces/"+workspaceID+"/events", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, c.readError(resp)
	}
	return resp.Body, nil
}

// ReadSSE reads SSE events from a reader, calling fn for each parsed event.
// Returns when the reader is exhausted or the context is canceled.
func ReadSSE(ctx context.Context, r io.Reader, fn func(SSEPayload) error) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	for scanner.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := line[6:]
		if len(strings.TrimSpace(data)) == 0 {
			continue
		}
		var payload SSEPayload
		if err := json.Unmarshal([]byte(data), &payload); err != nil {
			return fmt.Errorf("parse SSE payload: %w (data: %s)", err, data[:min(len(data), 200)])
		}
		if err := fn(payload); err != nil {
			return err
		}
	}
	return scanner.Err()
}

// SkipPermissions sets the workspace to auto-approve permissions.
func (c *Client) SkipPermissions(ctx context.Context, workspaceID string) error {
	body, err := json.Marshal(struct {
		Skip bool `json:"skip"`
	}{Skip: true})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.BaseURL+"/workspaces/"+workspaceID+"/permissions/skip", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return c.readError(resp)
	}
	return nil
}

func (c *Client) readError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var apiErr Error
	if json.Unmarshal(body, &apiErr) == nil && apiErr.Message != "" {
		return fmt.Errorf("crush API %d: %s", resp.StatusCode, apiErr.Message)
	}
	return fmt.Errorf("crush API %d: %s", resp.StatusCode, string(body[:min(len(body), 200)]))
}

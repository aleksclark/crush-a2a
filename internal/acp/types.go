package acp

// AgentManifest describes an ACP agent.
type AgentManifest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version,omitempty"`
}

// AgentsResponse wraps the GET /agents response.
type AgentsResponse struct {
	Agents []AgentManifest `json:"agents"`
}

// MessagePart is a single content part in an ACP message.
type MessagePart struct {
	ContentType     string `json:"content_type"`
	Content         string `json:"content"`
	ContentEncoding string `json:"content_encoding,omitempty"`
}

// Message is an ACP message with role and parts.
type Message struct {
	Role  string        `json:"role"`
	Parts []MessagePart `json:"parts"`
}

// Run represents an ACP run.
type Run struct {
	AgentName string    `json:"agent_name"`
	RunID     string    `json:"run_id"`
	SessionID string    `json:"session_id"`
	Status    string    `json:"status"`
	Output    []Message `json:"output,omitempty"`
}

// CreateRunRequest is the body for POST /runs.
type CreateRunRequest struct {
	AgentName string    `json:"agent_name"`
	Input     []Message `json:"input"`
	SessionID string    `json:"session_id,omitempty"`
	Mode      string    `json:"mode"`
}

// Event is a single NDJSON event from the ACP streaming endpoint.
type Event struct {
	Type string `json:"type"`
	Run  *Run   `json:"run,omitempty"`
	Part *struct {
		ContentType     string `json:"content_type"`
		Content         string `json:"content"`
		ContentEncoding string `json:"content_encoding,omitempty"`
	} `json:"part,omitempty"`
}

package crush

import (
	"encoding/json"
	"fmt"
)

// Workspace represents a Crush workspace.
type Workspace struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

// Session represents a Crush session.
type Session struct {
	ID              string `json:"id"`
	ParentSessionID string `json:"parent_session_id,omitempty"`
	Title           string `json:"title"`
	MessageCount    int64  `json:"message_count,omitempty"`
	CreatedAt       int64  `json:"created_at,omitempty"`
	UpdatedAt       int64  `json:"updated_at,omitempty"`
}

// AgentMessage is the body for POST /v1/workspaces/{id}/agent.
type AgentMessage struct {
	SessionID string `json:"session_id"`
	Prompt    string `json:"prompt"`
}

// AgentSession represents an agent session status.
type AgentSession struct {
	Session
	IsBusy bool `json:"is_busy"`
}

// AgentInfo represents agent information.
type AgentInfo struct {
	IsBusy  bool `json:"is_busy"`
	IsReady bool `json:"is_ready"`
}

// Error represents a Crush API error response.
type Error struct {
	Message string `json:"message"`
}

// MessageRole represents the role of a message sender.
type MessageRole string

const (
	RoleAssistant MessageRole = "assistant"
	RoleUser      MessageRole = "user"
	RoleSystem    MessageRole = "system"
	RoleTool      MessageRole = "tool"
)

// FinishReason represents why a message generation finished.
type FinishReason string

const (
	FinishEndTurn  FinishReason = "end_turn"
	FinishCanceled FinishReason = "canceled"
	FinishError    FinishReason = "error"
)

// ContentPart is a discriminated union for message content parts.
// The wire format uses {"type": "...", "data": {...}}.
type ContentPart struct {
	Type string
	Data json.RawMessage

	// Decoded fields (populated after unmarshal)
	Text       *TextContent
	ToolCall   *ToolCall
	ToolResult *ToolResult
	Finish     *Finish
	Reasoning  *ReasoningContent
}

// TextContent is a text part.
type TextContent struct {
	Text string `json:"text"`
}

// ToolCall represents a tool call.
type ToolCall struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Input    string `json:"input"`
	Finished bool   `json:"finished,omitempty"`
}

// ToolResult represents a tool result.
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Name       string `json:"name"`
	Content    string `json:"content"`
	IsError    bool   `json:"is_error,omitempty"`
}

// Finish represents the end of a message generation.
type Finish struct {
	Reason  FinishReason `json:"reason"`
	Time    int64        `json:"time,omitempty"`
	Message string       `json:"message,omitempty"`
	Details string       `json:"details,omitempty"`
}

// ReasoningContent represents reasoning/thinking content.
type ReasoningContent struct {
	Thinking string `json:"thinking"`
}

// UnmarshalJSON decodes a ContentPart from the wire format.
func (p *ContentPart) UnmarshalJSON(data []byte) error {
	var raw struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	p.Type = raw.Type
	p.Data = raw.Data
	switch raw.Type {
	case "text":
		var t TextContent
		if err := json.Unmarshal(raw.Data, &t); err != nil {
			return fmt.Errorf("decode text part: %w", err)
		}
		p.Text = &t
	case "tool_call":
		var tc ToolCall
		if err := json.Unmarshal(raw.Data, &tc); err != nil {
			return fmt.Errorf("decode tool_call part: %w", err)
		}
		p.ToolCall = &tc
	case "tool_result":
		var tr ToolResult
		if err := json.Unmarshal(raw.Data, &tr); err != nil {
			return fmt.Errorf("decode tool_result part: %w", err)
		}
		p.ToolResult = &tr
	case "finish":
		var f Finish
		if err := json.Unmarshal(raw.Data, &f); err != nil {
			return fmt.Errorf("decode finish part: %w", err)
		}
		p.Finish = &f
	case "reasoning":
		var r ReasoningContent
		if err := json.Unmarshal(raw.Data, &r); err != nil {
			return fmt.Errorf("decode reasoning part: %w", err)
		}
		p.Reasoning = &r
	}
	return nil
}

// Message represents a Crush message.
type Message struct {
	ID        string        `json:"id"`
	Role      MessageRole   `json:"role"`
	SessionID string        `json:"session_id"`
	Parts     []ContentPart `json:"parts"`
	Model     string        `json:"model,omitempty"`
	Provider  string        `json:"provider,omitempty"`
	CreatedAt int64         `json:"created_at,omitempty"`
	UpdatedAt int64         `json:"updated_at,omitempty"`
}

// IsFinished returns true if the message has a finish part.
func (m *Message) IsFinished() bool {
	for _, p := range m.Parts {
		if p.Finish != nil {
			return true
		}
	}
	return false
}

// FinishPart returns the finish part if present.
func (m *Message) FinishPart() *Finish {
	for _, p := range m.Parts {
		if p.Finish != nil {
			return p.Finish
		}
	}
	return nil
}

// TextContent returns the concatenated text of all text parts.
func (m *Message) TextContent() string {
	var result string
	for _, p := range m.Parts {
		if p.Text != nil {
			result += p.Text.Text
		}
	}
	return result
}

// PayloadType identifies the type of SSE event payload.
type PayloadType string

const (
	PayloadTypeMessage           PayloadType = "message"
	PayloadTypeSession           PayloadType = "session"
	PayloadTypeAgentEvent        PayloadType = "agent_event"
	PayloadTypePermissionRequest PayloadType = "permission_request"
	PayloadTypeFile              PayloadType = "file"
)

// SSEPayload is the outer envelope for SSE events.
type SSEPayload struct {
	Type    PayloadType     `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// EventType is the type of event within a payload.
type EventType string

const (
	EventCreated EventType = "created"
	EventUpdated EventType = "updated"
	EventDeleted EventType = "deleted"
)

// SSEEvent is the inner event within an SSE payload.
type SSEEvent[T any] struct {
	Type    EventType `json:"type"`
	Payload T         `json:"payload"`
}

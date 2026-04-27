package a2a

import "encoding/json"

// AgentCard is the A2A v1.0 agent discovery response.
type AgentCard struct {
	Name                string               `json:"name"`
	Version             string               `json:"version"`
	Description         string               `json:"description,omitempty"`
	Capabilities        *Capabilities        `json:"capabilities,omitempty"`
	Skills              []Skill              `json:"skills"`
	SupportedInterfaces []SupportedInterface `json:"supportedInterfaces"`
	DefaultInputModes   []string             `json:"defaultInputModes"`
	DefaultOutputModes  []string             `json:"defaultOutputModes"`
}

// SupportedInterface describes a protocol interface the agent supports.
type SupportedInterface struct {
	URL             string `json:"url"`
	ProtocolBinding string `json:"protocolBinding"`
	ProtocolVersion string `json:"protocolVersion"`
}

// Capabilities declares what the agent supports.
type Capabilities struct {
	Streaming              bool `json:"streaming"`
	PushNotifications      bool `json:"pushNotifications"`
	StateTransitionHistory bool `json:"stateTransitionHistory"`
}

// Skill describes one skill the agent can perform.
type Skill struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Examples    []string `json:"examples,omitempty"`
}

// Role constants for A2A v1.0 proto enum names.
const (
	RoleUser  = "ROLE_USER"
	RoleAgent = "ROLE_AGENT"
)

// Message is an A2A message.
type Message struct {
	Kind      string `json:"kind,omitempty"`
	MessageID string `json:"messageId"`
	Role      string `json:"role"`
	Parts     []Part `json:"parts"`
	ContextID string `json:"contextId,omitempty"`
	TaskID    string `json:"taskId,omitempty"`
}

// Part is a union type for message parts.
// v1.0 uses member-based polymorphism (no "kind" discriminator).
// A text part is just {"text": "Hello"}, a file part has {"file": {...}}, etc.
type Part struct {
	// Text part field
	Text string `json:"text,omitempty"`

	// File part field
	File *FileContent `json:"file,omitempty"`

	// Data part field
	Data json.RawMessage `json:"data,omitempty"`
}

// FileContent holds file data for a FilePart.
type FileContent struct {
	Name     string `json:"name,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	Bytes    string `json:"bytes,omitempty"`
	URI      string `json:"uri,omitempty"`
}

// Task is an A2A task.
type Task struct {
	ID        string     `json:"id"`
	ContextID string     `json:"contextId"`
	Status    TaskStatus `json:"status"`
	Artifacts []Artifact `json:"artifacts,omitempty"`
	History   []Message  `json:"history,omitempty"`
}

// TaskStatus represents the current state of a task.
type TaskStatus struct {
	State     string   `json:"state"`
	Timestamp string   `json:"timestamp"`
	Message   *Message `json:"message,omitempty"`
}

// Artifact is an output artifact from a task.
type Artifact struct {
	ArtifactID string `json:"artifactId"`
	Name       string `json:"name,omitempty"`
	Parts      []Part `json:"parts"`
	Append     bool   `json:"append,omitempty"`
	LastChunk  bool   `json:"lastChunk,omitempty"`
}

// TaskStatusUpdateEvent is an SSE event for status changes.
type TaskStatusUpdateEvent struct {
	Kind      string     `json:"kind,omitempty"`
	TaskID    string     `json:"taskId"`
	ContextID string     `json:"contextId"`
	Status    TaskStatus `json:"status"`
	Final     bool       `json:"final,omitempty"`
}

// TaskArtifactUpdateEvent is an SSE event for artifact updates.
type TaskArtifactUpdateEvent struct {
	Kind      string   `json:"kind,omitempty"`
	TaskID    string   `json:"taskId"`
	ContextID string   `json:"contextId"`
	Artifact  Artifact `json:"artifact"`
}

// SendMessageResult wraps the response for SendMessage as per v1.0 proto.
// The oneof is: { task: Task } or { message: Message }.
type SendMessageResult struct {
	Task    *Task    `json:"task,omitempty"`
	Message *Message `json:"message,omitempty"`
}

// SendMessageParams is the params for SendMessage / SendStreamingMessage.
type SendMessageParams struct {
	Message Message `json:"message"`
}

// GetTaskParams is the params for GetTask.
type GetTaskParams struct {
	TaskID string `json:"id"`
}

// CancelTaskParams is the params for CancelTask.
type CancelTaskParams struct {
	TaskID string `json:"id"`
}

// Task state constants.
const (
	TaskStateSubmitted     = "submitted"
	TaskStateWorking       = "working"
	TaskStateCompleted     = "completed"
	TaskStateFailed        = "failed"
	TaskStateCanceled      = "canceled"
	TaskStateInputRequired = "input-required"
)

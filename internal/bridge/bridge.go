package bridge

import (
	"encoding/base64"
	"strings"
	"time"

	"github.com/aleksclark/crush-a2a/internal/a2a"
	"github.com/aleksclark/crush-a2a/internal/acp"
	"github.com/google/uuid"
)

// A2APartsToACPParts converts A2A Parts to ACP MessageParts.
func A2APartsToACPParts(parts []a2a.Part) []acp.MessagePart {
	out := make([]acp.MessagePart, 0, len(parts))
	for _, p := range parts {
		switch p.Kind {
		case "text":
			out = append(out, acp.MessagePart{
				ContentType: "text/plain",
				Content:     p.Text,
			})
		case "file":
			if p.File != nil {
				mp := acp.MessagePart{
					ContentType: p.File.MimeType,
				}
				if p.File.Bytes != "" {
					mp.Content = p.File.Bytes
					mp.ContentEncoding = "base64"
				} else if p.File.URI != "" {
					mp.Content = p.File.URI
				}
				out = append(out, mp)
			}
		case "data":
			out = append(out, acp.MessagePart{
				ContentType: "application/json",
				Content:     string(p.Data),
			})
		}
	}
	return out
}

// ACPPartsToA2AParts converts ACP MessageParts to A2A Parts.
func ACPPartsToA2AParts(parts []acp.MessagePart) []a2a.Part {
	out := make([]a2a.Part, 0, len(parts))
	for _, p := range parts {
		if p.ContentType == "text/plain" || p.ContentType == "" {
			out = append(out, a2a.Part{
				Kind: "text",
				Text: p.Content,
			})
		} else if isFileMimeType(p.ContentType) {
			fc := &a2a.FileContent{
				MimeType: p.ContentType,
			}
			if p.ContentEncoding == "base64" {
				fc.Bytes = p.Content
			} else {
				// Try to detect if content is base64
				if _, err := base64.StdEncoding.DecodeString(p.Content); err == nil && len(p.Content) > 0 {
					fc.Bytes = p.Content
				} else {
					fc.URI = p.Content
				}
			}
			out = append(out, a2a.Part{
				Kind: "file",
				File: fc,
			})
		} else {
			out = append(out, a2a.Part{
				Kind: "text",
				Text: p.Content,
			})
		}
	}
	return out
}

// ACPMessagesToA2AArtifacts converts ACP output messages into A2A Artifacts.
func ACPMessagesToA2AArtifacts(messages []acp.Message) []a2a.Artifact {
	artifacts := make([]a2a.Artifact, 0, len(messages))
	for _, msg := range messages {
		if msg.Role != "agent" {
			continue
		}
		artifacts = append(artifacts, a2a.Artifact{
			ArtifactID: uuid.New().String(),
			Parts:      ACPPartsToA2AParts(msg.Parts),
		})
	}
	return artifacts
}

// A2AMessageToACPMessages converts an A2A Message to ACP Messages.
func A2AMessageToACPMessages(msg a2a.Message) []acp.Message {
	return []acp.Message{
		{
			Role:  msg.Role,
			Parts: A2APartsToACPParts(msg.Parts),
		},
	}
}

// ACPStatusToA2AState maps ACP run status to A2A task state.
func ACPStatusToA2AState(status string) string {
	switch status {
	case "created":
		return a2a.TaskStateSubmitted
	case "in-progress":
		return a2a.TaskStateWorking
	case "completed":
		return a2a.TaskStateCompleted
	case "failed":
		return a2a.TaskStateFailed
	case "cancelled":
		return a2a.TaskStateCanceled
	default:
		return a2a.TaskStateWorking
	}
}

// RunToTask converts an ACP Run into an A2A Task.
func RunToTask(run *acp.Run, taskID, contextID string) *a2a.Task {
	task := &a2a.Task{
		Kind:      "task",
		ID:        taskID,
		ContextID: contextID,
		Status: a2a.TaskStatus{
			State:     ACPStatusToA2AState(run.Status),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		},
		Artifacts: ACPMessagesToA2AArtifacts(run.Output),
	}
	if run.Error != nil && run.Error.Message != "" {
		task.Status.Message = &a2a.Message{
			Role: "agent",
			Parts: []a2a.Part{
				{Kind: "text", Text: run.Error.Message},
			},
		}
	}
	return task
}

// Now returns a formatted timestamp.
func Now() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func isFileMimeType(ct string) bool {
	return strings.HasPrefix(ct, "image/") ||
		strings.HasPrefix(ct, "audio/") ||
		strings.HasPrefix(ct, "video/") ||
		strings.HasPrefix(ct, "application/octet-stream") ||
		strings.HasPrefix(ct, "application/pdf")
}

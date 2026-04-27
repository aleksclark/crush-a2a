package bridge

import (
	"time"

	"github.com/aleksclark/crush-a2a/internal/a2a"
	"github.com/aleksclark/crush-a2a/internal/crush"
	"github.com/google/uuid"
)

// ExtractPromptText extracts the plain text from A2A message parts.
func ExtractPromptText(parts []a2a.Part) string {
	var text string
	for _, p := range parts {
		if p.Kind == "text" {
			if text != "" {
				text += "\n"
			}
			text += p.Text
		}
	}
	return text
}

// CrushMessagesToA2AArtifacts converts Crush assistant messages to A2A Artifacts.
func CrushMessagesToA2AArtifacts(messages []crush.Message) []a2a.Artifact {
	artifacts := make([]a2a.Artifact, 0)
	for _, msg := range messages {
		if msg.Role != crush.RoleAssistant {
			continue
		}
		parts := crushPartsToA2AParts(msg.Parts)
		if len(parts) == 0 {
			continue
		}
		artifacts = append(artifacts, a2a.Artifact{
			ArtifactID: uuid.New().String(),
			Parts:      parts,
		})
	}
	return artifacts
}

// CrushMessageToA2AArtifact converts a single Crush message to an A2A Artifact.
func CrushMessageToA2AArtifact(msg *crush.Message) *a2a.Artifact {
	parts := crushPartsToA2AParts(msg.Parts)
	if len(parts) == 0 {
		return nil
	}
	return &a2a.Artifact{
		ArtifactID: uuid.New().String(),
		Parts:      parts,
	}
}

// CrushFinishToA2AState maps a Crush finish reason to an A2A task state.
func CrushFinishToA2AState(reason crush.FinishReason) string {
	switch reason {
	case crush.FinishEndTurn:
		return a2a.TaskStateCompleted
	case crush.FinishError:
		return a2a.TaskStateFailed
	case crush.FinishCanceled:
		return a2a.TaskStateCanceled
	default:
		return a2a.TaskStateCompleted
	}
}

// Now returns a formatted timestamp.
func Now() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func crushPartsToA2AParts(parts []crush.ContentPart) []a2a.Part {
	out := make([]a2a.Part, 0, len(parts))
	for _, p := range parts {
		if p.Text != nil && p.Text.Text != "" {
			out = append(out, a2a.Part{
				Kind: "text",
				Text: p.Text.Text,
			})
		}
	}
	return out
}

package bridge

import (
	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/aleksclark/crush-a2a/internal/crush"
)

// ExtractPromptText extracts the plain text from SDK message parts.
func ExtractPromptText(parts a2a.ContentParts) string {
	var text string
	for _, p := range parts {
		if t := p.Text(); t != "" {
			if text != "" {
				text += "\n"
			}
			text += t
		}
	}
	return text
}

// CrushMessagesToA2AArtifacts converts Crush assistant messages to SDK Artifacts.
func CrushMessagesToA2AArtifacts(messages []crush.Message) []*a2a.Artifact {
	artifacts := make([]*a2a.Artifact, 0)
	for _, msg := range messages {
		if msg.Role != crush.RoleAssistant {
			continue
		}
		parts := crushPartsToA2AParts(msg.Parts)
		if len(parts) == 0 {
			continue
		}
		artifacts = append(artifacts, &a2a.Artifact{
			ID:    a2a.NewArtifactID(),
			Parts: parts,
		})
	}
	return artifacts
}

// CrushFinishToA2AState maps a Crush finish reason to an SDK TaskState.
func CrushFinishToA2AState(reason crush.FinishReason) a2a.TaskState {
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

func crushPartsToA2AParts(parts []crush.ContentPart) a2a.ContentParts {
	out := make(a2a.ContentParts, 0, len(parts))
	for _, p := range parts {
		if p.Text != nil && p.Text.Text != "" {
			out = append(out, a2a.NewTextPart(p.Text.Text))
		}
	}
	return out
}

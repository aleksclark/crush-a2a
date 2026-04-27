package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"

	"github.com/aleksclark/crush-a2a/internal/a2a"
	"github.com/aleksclark/crush-a2a/internal/acp"
	"github.com/google/uuid"
)

// SSEWriter writes A2A SSE events to an io.Writer.
type SSEWriter struct {
	W io.Writer
}

// WriteEvent writes a single SSE data event.
func (s *SSEWriter) WriteEvent(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(s.W, "data: %s\n\n", data)
	return err
}

// StreamAdapter reads ACP NDJSON events and writes A2A SSE events.
func StreamAdapter(ctx context.Context, r io.Reader, w *SSEWriter, taskID, contextID string, logger *slog.Logger) error {
	artifactCounter := 0
	return acp.ReadNDJSON(ctx, r, func(ev acp.Event) error {
		switch ev.Type {
		case "run.created":
			return w.WriteEvent(a2a.TaskStatusUpdateEvent{
				Kind:      "status-update",
				TaskID:    taskID,
				ContextID: contextID,
				Status: a2a.TaskStatus{
					State:     a2a.TaskStateSubmitted,
					Timestamp: Now(),
				},
			})

		case "run.in-progress":
			return w.WriteEvent(a2a.TaskStatusUpdateEvent{
				Kind:      "status-update",
				TaskID:    taskID,
				ContextID: contextID,
				Status: a2a.TaskStatus{
					State:     a2a.TaskStateWorking,
					Timestamp: Now(),
				},
			})

		case "message.part":
			if ev.Part == nil {
				return nil
			}
			artifactCounter++
			parts := ACPPartsToA2AParts([]acp.MessagePart{
				{
					ContentType:     ev.Part.ContentType,
					Content:         ev.Part.Content,
					ContentEncoding: ev.Part.ContentEncoding,
				},
			})
			return w.WriteEvent(a2a.TaskArtifactUpdateEvent{
				Kind:      "artifact-update",
				TaskID:    taskID,
				ContextID: contextID,
				Artifact: a2a.Artifact{
					ArtifactID: uuid.New().String(),
					Parts:      parts,
					Append:     artifactCounter > 1,
				},
			})

		case "run.completed":
			return w.WriteEvent(a2a.TaskStatusUpdateEvent{
				Kind:      "status-update",
				TaskID:    taskID,
				ContextID: contextID,
				Status: a2a.TaskStatus{
					State:     a2a.TaskStateCompleted,
					Timestamp: Now(),
				},
				Final: true,
			})

		case "run.failed":
			return w.WriteEvent(a2a.TaskStatusUpdateEvent{
				Kind:      "status-update",
				TaskID:    taskID,
				ContextID: contextID,
				Status: a2a.TaskStatus{
					State:     a2a.TaskStateFailed,
					Timestamp: Now(),
				},
				Final: true,
			})

		default:
			logger.Debug("unknown ACP event type", "type", ev.Type)
			return nil
		}
	})
}

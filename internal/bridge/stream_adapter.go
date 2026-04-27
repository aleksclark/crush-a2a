package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/aleksclark/crush-a2a/internal/a2a"
	"github.com/aleksclark/crush-a2a/internal/acp"
	"github.com/google/uuid"
)

// SSEWriter writes A2A SSE events to an http.ResponseWriter with flushing.
type SSEWriter struct {
	W       http.ResponseWriter
	Flusher http.Flusher
}

// WriteEvent writes a single SSE data event and flushes.
func (s *SSEWriter) WriteEvent(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("SSE marshal: %w", err)
	}
	n, err := fmt.Fprintf(s.W, "data: %s\n\n", data)
	if err != nil {
		return fmt.Errorf("SSE write (wrote %d bytes): %w", n, err)
	}
	if s.Flusher != nil {
		s.Flusher.Flush()
	}
	return nil
}

// StreamAdapter reads ACP NDJSON events and writes A2A SSE events.
func StreamAdapter(ctx context.Context, r io.Reader, w *SSEWriter, taskID, contextID string, logger *slog.Logger) error {
	artifactCounter := 0
	return acp.ReadNDJSON(ctx, r, func(ev acp.Event) error {
		logger.Debug("stream event received", "type", ev.Type)

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
				logger.Debug("message.part event with nil part, skipping")
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
			logger.Debug("emitting artifact-update", "artifact_num", artifactCounter, "parts_count", len(parts))
			return w.WriteEvent(a2a.TaskArtifactUpdateEvent{
				Kind:      "artifact-update",
				TaskID:    taskID,
				ContextID: contextID,
				Artifact: a2a.Artifact{
					ArtifactID: uuid.New().String(),
					Parts:      parts,
					Append:     artifactCounter > 1,
					LastChunk:  false,
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
			errMsg := ""
			if ev.Run != nil {
				if ev.Run.Error != nil {
					errMsg = ev.Run.Error.Message
				}
				logger.Error("ACP run failed", "run_id", ev.Run.RunID, "error", errMsg)
			}
			logger.Debug("run.failed raw event", "raw", string(ev.Raw[:min(len(ev.Raw), 500)]))
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

		case "run.cancelled":
			return w.WriteEvent(a2a.TaskStatusUpdateEvent{
				Kind:      "status-update",
				TaskID:    taskID,
				ContextID: contextID,
				Status: a2a.TaskStatus{
					State:     a2a.TaskStateCanceled,
					Timestamp: Now(),
				},
				Final: true,
			})

		default:
			logger.Debug("ignoring ACP event", "type", ev.Type)
			return nil
		}
	})
}

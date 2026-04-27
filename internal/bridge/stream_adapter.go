package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/aleksclark/crush-a2a/internal/a2a"
	"github.com/aleksclark/crush-a2a/internal/crush"
	"github.com/google/uuid"
)

// SSEWriter writes A2A SSE events to an http.ResponseWriter with flushing.
type SSEWriter struct {
	W       http.ResponseWriter
	Flusher http.Flusher
}

// WriteEvent writes a single SSE data event and flushes.
func (s *SSEWriter) WriteEvent(v any) error {
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

// StreamAdapter reads Crush SSE events and writes A2A SSE events.
func StreamAdapter(ctx context.Context, r io.Reader, w *SSEWriter, taskID, contextID, sessionID string, logger *slog.Logger) error {
	sentWorking := false

	return crush.ReadSSE(ctx, r, func(payload crush.SSEPayload) error {
		switch payload.Type {
		case crush.PayloadTypeMessage:
			var ev crush.SSEEvent[crush.Message]
			if err := json.Unmarshal(payload.Payload, &ev); err != nil {
				logger.Debug("failed to decode message event", "error", err)
				return nil
			}

			msg := ev.Payload
			if msg.SessionID != sessionID {
				return nil
			}

			if msg.Role != crush.RoleAssistant {
				return nil
			}

			if !sentWorking {
				sentWorking = true
				if err := w.WriteEvent(a2a.TaskStatusUpdateEvent{
					Kind:      "status-update",
					TaskID:    taskID,
					ContextID: contextID,
					Status: a2a.TaskStatus{
						State:     a2a.TaskStateWorking,
						Timestamp: Now(),
					},
				}); err != nil {
					return err
				}
			}

			artifact := CrushMessageToA2AArtifact(&msg)
			if artifact != nil {
				if err := w.WriteEvent(a2a.TaskArtifactUpdateEvent{
					Kind:      "artifact-update",
					TaskID:    taskID,
					ContextID: contextID,
					Artifact:  *artifact,
				}); err != nil {
					return err
				}
			}

			if finish := msg.FinishPart(); finish != nil {
				state := CrushFinishToA2AState(finish.Reason)
				var statusMsg *a2a.Message
				if finish.Message != "" {
					statusMsg = &a2a.Message{
						Kind:      "message",
						MessageID: uuid.New().String(),
						Role:      "agent",
						Parts:     []a2a.Part{{Kind: "text", Text: finish.Message}},
					}
				}
				return w.WriteEvent(a2a.TaskStatusUpdateEvent{
					Kind:      "status-update",
					TaskID:    taskID,
					ContextID: contextID,
					Status: a2a.TaskStatus{
						State:     state,
						Timestamp: Now(),
						Message:   statusMsg,
					},
					Final: true,
				})
			}

		case crush.PayloadTypeAgentEvent:
			logger.Debug("agent_event received")

		default:
			logger.Debug("ignoring SSE event", "type", payload.Type)
		}

		return nil
	})
}

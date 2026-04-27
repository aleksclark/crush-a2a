package server

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/aleksclark/crush-a2a/internal/a2a"
	"github.com/aleksclark/crush-a2a/internal/bridge"
	"github.com/aleksclark/crush-a2a/internal/crush"
	"github.com/google/uuid"
)

func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request, req a2a.JSONRPCRequest) {
	var params a2a.SendMessageParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.Logger.Error("failed to parse SendMessage params", "error", err)
		writeJSONRPCError(w, a2a.ErrInvalidParams(req.ID, err.Error()))
		return
	}

	msg := params.Message
	prompt := bridge.ExtractPromptText(msg.Parts)
	if prompt == "" {
		writeJSONRPCError(w, a2a.ErrInvalidParams(req.ID, "empty prompt"))
		return
	}

	contextID := msg.ContextID
	if contextID == "" {
		contextID = uuid.New().String()
	}

	taskID := msg.TaskID
	if taskID == "" {
		taskID = uuid.New().String()
	}

	wsID, err := s.ensureWorkspace(r)
	if err != nil {
		s.Logger.Error("failed to ensure workspace", "error", err)
		writeJSONRPCError(w, a2a.ErrInternalError(req.ID, err.Error()))
		return
	}

	sess, err := s.Crush.CreateSession(r.Context(), wsID, "A2A task "+taskID)
	if err != nil {
		s.Logger.Error("failed to create session", "error", err)
		writeJSONRPCError(w, a2a.ErrInternalError(req.ID, err.Error()))
		return
	}

	s.Logger.Info("SendMessage: sending prompt",
		"task_id", taskID,
		"context_id", contextID,
		"workspace_id", wsID,
		"session_id", sess.ID,
	)

	sseStream, err := s.Crush.SubscribeEvents(r.Context(), wsID)
	if err != nil {
		s.Logger.Error("failed to subscribe to events", "error", err)
		writeJSONRPCError(w, a2a.ErrInternalError(req.ID, err.Error()))
		return
	}
	defer sseStream.Close()

	err = s.Crush.SendMessage(r.Context(), wsID, crush.AgentMessage{
		SessionID: sess.ID,
		Prompt:    prompt,
	})
	if err != nil {
		s.Logger.Error("failed to send message", "error", err)
		writeJSONRPCError(w, a2a.ErrInternalError(req.ID, err.Error()))
		return
	}

	finished := make(chan struct{})
	var finalState string
	var finishMsg string

	go func() {
		defer close(finished)
		crush.ReadSSE(r.Context(), sseStream, func(payload crush.SSEPayload) error {
			if payload.Type != crush.PayloadTypeMessage {
				return nil
			}
			var ev crush.SSEEvent[crush.Message]
			if err := json.Unmarshal(payload.Payload, &ev); err != nil {
				return nil
			}
			m := ev.Payload
			if m.SessionID != sess.ID || m.Role != crush.RoleAssistant {
				return nil
			}
			if finish := m.FinishPart(); finish != nil {
				finalState = bridge.CrushFinishToA2AState(finish.Reason)
				finishMsg = finish.Message
				return io.EOF
			}
			return nil
		})
	}()

	select {
	case <-finished:
	case <-time.After(5 * time.Minute):
		finalState = a2a.TaskStateFailed
		finishMsg = "timeout waiting for agent response"
	case <-r.Context().Done():
		finalState = a2a.TaskStateCanceled
	}

	if finalState == "" {
		finalState = a2a.TaskStateCompleted
	}

	messages, err := s.Crush.GetMessages(r.Context(), wsID, sess.ID)
	if err != nil {
		s.Logger.Error("failed to get messages", "error", err)
		writeJSONRPCError(w, a2a.ErrInternalError(req.ID, err.Error()))
		return
	}

	artifacts := bridge.CrushMessagesToA2AArtifacts(messages)

	task := &a2a.Task{
		Kind:      "task",
		ID:        taskID,
		ContextID: contextID,
		Status: a2a.TaskStatus{
			State:     finalState,
			Timestamp: bridge.Now(),
		},
		Artifacts: artifacts,
	}

	if finishMsg != "" {
		task.Status.Message = &a2a.Message{
			Kind:      "message",
			MessageID: uuid.New().String(),
			Role:      "agent",
			Parts:     []a2a.Part{{Kind: "text", Text: finishMsg}},
		}
	}

	s.Store.Put(&bridge.TaskEntry{
		TaskID:      taskID,
		ContextID:   contextID,
		WorkspaceID: wsID,
		SessionID:   sess.ID,
		Task:        task,
	})

	s.Logger.Info("SendMessage: returning task",
		"task_id", taskID,
		"state", task.Status.State,
		"artifacts", len(task.Artifacts),
	)

	writeJSONRPCResponse(w, a2a.NewJSONRPCResponse(req.ID, task))
}

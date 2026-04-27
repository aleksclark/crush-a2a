package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/aleksclark/crush-a2a/internal/a2a"
	"github.com/aleksclark/crush-a2a/internal/bridge"
	"github.com/aleksclark/crush-a2a/internal/crush"
	"github.com/google/uuid"
)

func (s *Server) handleSendStreamingMessage(w http.ResponseWriter, r *http.Request, req a2a.JSONRPCRequest) {
	var params a2a.SendMessageParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.Logger.Error("failed to parse SendStreamingMessage params", "error", err)
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

	sess, err := s.Crush.CreateSession(r.Context(), wsID, "A2A stream "+taskID)
	if err != nil {
		s.Logger.Error("failed to create session", "error", err)
		writeJSONRPCError(w, a2a.ErrInternalError(req.ID, err.Error()))
		return
	}

	s.Logger.Info("SendStreamingMessage",
		"task_id", taskID,
		"context_id", contextID,
		"workspace_id", wsID,
		"session_id", sess.ID,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		<-r.Context().Done()
		cancel()
	}()

	sseStream, err := s.Crush.SubscribeEvents(ctx, wsID)
	if err != nil {
		s.Logger.Error("failed to subscribe to events", "error", err)
		writeJSONRPCError(w, a2a.ErrInternalError(req.ID, err.Error()))
		return
	}
	defer sseStream.Close()

	err = s.Crush.SendMessage(ctx, wsID, crush.AgentMessage{
		SessionID: sess.ID,
		Prompt:    prompt,
	})
	if err != nil {
		s.Logger.Error("failed to send message", "error", err)
		writeJSONRPCError(w, a2a.ErrInternalError(req.ID, err.Error()))
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		s.Logger.Error("response writer does not support flushing")
		writeJSONRPCError(w, a2a.ErrInternalError(req.ID, "streaming not supported"))
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	sse := &bridge.SSEWriter{W: w, Flusher: flusher}

	err = bridge.StreamAdapter(ctx, sseStream, sse, taskID, contextID, sess.ID, s.Logger)
	if err != nil {
		s.Logger.Error("stream adapter error", "error", err)
	}

	s.Store.Put(&bridge.TaskEntry{
		TaskID:      taskID,
		ContextID:   contextID,
		WorkspaceID: wsID,
		SessionID:   sess.ID,
		Task: &a2a.Task{
			ID:        taskID,
			ContextID: contextID,
			Status: a2a.TaskStatus{
				State:     a2a.TaskStateCompleted,
				Timestamp: bridge.Now(),
			},
		},
	})
}

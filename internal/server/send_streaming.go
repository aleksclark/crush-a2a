package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/aleksclark/crush-a2a/internal/a2a"
	"github.com/aleksclark/crush-a2a/internal/acp"
	"github.com/aleksclark/crush-a2a/internal/bridge"
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
	acpMessages := bridge.A2AMessageToACPMessages(msg)

	contextID := msg.ContextID
	if contextID == "" {
		contextID = uuid.New().String()
	}

	taskID := msg.TaskID
	if taskID == "" {
		taskID = uuid.New().String()
	}

	s.Logger.Info("SendStreamingMessage", "task_id", taskID, "context_id", contextID)

	createReq := acp.CreateRunRequest{
		AgentName: s.AgentName,
		Input:     acpMessages,
		SessionID: contextID,
		Mode:      "stream",
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		<-r.Context().Done()
		cancel()
	}()

	stream, err := s.ACPClient.CreateRunStream(ctx, createReq)
	if err != nil {
		s.Logger.Error("ACP CreateRunStream failed", "error", err)
		writeJSONRPCError(w, a2a.ErrInternalError(req.ID, err.Error()))
		return
	}
	defer stream.Close()

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

	err = bridge.StreamAdapter(ctx, stream, sse, taskID, contextID, s.Logger)
	if err != nil {
		s.Logger.Error("stream adapter error", "error", err)
	}

	s.Store.Put(&bridge.TaskEntry{
		TaskID:    taskID,
		ContextID: contextID,
		Task: &a2a.Task{
			Kind:      "task",
			ID:        taskID,
			ContextID: contextID,
			Status: a2a.TaskStatus{
				State:     a2a.TaskStateCompleted,
				Timestamp: bridge.Now(),
			},
		},
	})
}

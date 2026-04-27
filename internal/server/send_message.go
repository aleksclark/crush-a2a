package server

import (
	"encoding/json"
	"net/http"

	"github.com/aleksclark/crush-a2a/internal/a2a"
	"github.com/aleksclark/crush-a2a/internal/acp"
	"github.com/aleksclark/crush-a2a/internal/bridge"
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
	acpMessages := bridge.A2AMessageToACPMessages(msg)

	contextID := msg.ContextID
	if contextID == "" {
		contextID = uuid.New().String()
	}

	taskID := msg.TaskID
	if taskID == "" {
		taskID = uuid.New().String()
	}

	createReq := acp.CreateRunRequest{
		AgentName: s.AgentName,
		Input:     acpMessages,
		SessionID: contextID,
		Mode:      "sync",
	}

	s.Logger.Info("SendMessage: creating ACP run", "task_id", taskID, "context_id", contextID, "agent", s.AgentName)

	run, err := s.ACPClient.CreateRun(r.Context(), createReq)
	if err != nil {
		s.Logger.Error("ACP CreateRun failed", "error", err)
		writeJSONRPCError(w, a2a.ErrInternalError(req.ID, err.Error()))
		return
	}

	s.Logger.Info("ACP run completed",
		"run_id", run.RunID,
		"status", run.Status,
		"output_count", len(run.Output),
	)

	task := bridge.RunToTask(run, taskID, contextID)

	s.Store.Put(&bridge.TaskEntry{
		TaskID:    taskID,
		ContextID: contextID,
		RunID:     run.RunID,
		SessionID: run.SessionID,
		Task:      task,
	})

	s.Logger.Info("SendMessage: returning task", "task_id", taskID, "state", task.Status.State, "artifacts", len(task.Artifacts))

	writeJSONRPCResponse(w, a2a.NewJSONRPCResponse(req.ID, task))
}

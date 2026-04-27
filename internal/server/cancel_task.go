package server

import (
	"encoding/json"
	"net/http"

	"github.com/aleksclark/crush-a2a/internal/a2a"
	"github.com/aleksclark/crush-a2a/internal/bridge"
)

func (s *Server) handleCancelTask(w http.ResponseWriter, r *http.Request, req a2a.JSONRPCRequest) {
	var params a2a.CancelTaskParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		writeJSONRPCError(w, a2a.ErrInvalidParams(req.ID, err.Error()))
		return
	}

	entry, ok := s.Store.Get(params.TaskID)
	if !ok {
		writeJSONRPCError(w, a2a.ErrTaskNotFound(req.ID, params.TaskID))
		return
	}

	state := entry.Task.Status.State
	if state == a2a.TaskStateCompleted || state == a2a.TaskStateFailed || state == a2a.TaskStateCanceled {
		writeJSONRPCError(w, a2a.ErrTaskNotCancelable(req.ID, params.TaskID))
		return
	}

	if entry.RunID != "" {
		_, err := s.ACPClient.CancelRun(r.Context(), entry.RunID)
		if err != nil {
			s.Logger.Error("ACP CancelRun failed", "error", err, "run_id", entry.RunID)
		}
	}

	entry.Task.Status = a2a.TaskStatus{
		State:     a2a.TaskStateCanceled,
		Timestamp: bridge.Now(),
	}

	writeJSONRPCResponse(w, a2a.NewJSONRPCResponse(req.ID, entry.Task))
}

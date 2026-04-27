package server

import (
	"encoding/json"
	"net/http"

	"github.com/aleksclark/crush-a2a/internal/a2a"
)

func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request, req a2a.JSONRPCRequest) {
	var params a2a.GetTaskParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		writeJSONRPCError(w, a2a.ErrInvalidParams(req.ID, err.Error()))
		return
	}

	entry, ok := s.Store.Get(params.TaskID)
	if !ok {
		writeJSONRPCError(w, a2a.ErrTaskNotFound(req.ID, params.TaskID))
		return
	}

	writeJSONRPCResponse(w, a2a.NewJSONRPCResponse(req.ID, entry.Task))
}

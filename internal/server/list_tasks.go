package server

import (
	"net/http"

	"github.com/aleksclark/crush-a2a/internal/a2a"
)

func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request, req a2a.JSONRPCRequest) {
	entries := s.Store.List()
	tasks := make([]*a2a.Task, 0, len(entries))
	for _, e := range entries {
		tasks = append(tasks, e.Task)
	}

	writeJSONRPCResponse(w, a2a.NewJSONRPCResponse(req.ID, tasks))
}

package server

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/aleksclark/crush-a2a/internal/a2a"
	"github.com/aleksclark/crush-a2a/internal/bridge"
	"github.com/aleksclark/crush-a2a/internal/crush"
)

// Server is the A2A v1.0 HTTP server backed by the Crush native API.
type Server struct {
	Crush         *crush.Client
	WorkspacePath string
	BaseURL       string
	Store         *bridge.TaskStore
	Logger        *slog.Logger
	Mux           *http.ServeMux

	workspaceID string
}

// Config holds server configuration.
type Config struct {
	Port          int
	CrushAddr     string
	WorkspacePath string
	BaseURL       string
	Logger        *slog.Logger
}

// New creates a new A2A server.
func New(cfg Config) (*Server, error) {
	client, err := crush.NewClient(cfg.CrushAddr, cfg.Logger)
	if err != nil {
		return nil, err
	}

	s := &Server{
		Crush:         client,
		WorkspacePath: cfg.WorkspacePath,
		BaseURL:       cfg.BaseURL,
		Store:         bridge.NewTaskStore(),
		Logger:        cfg.Logger,
		Mux:           http.NewServeMux(),
	}
	s.Mux.HandleFunc("GET /.well-known/agent-card.json", s.handleAgentCard)
	s.Mux.HandleFunc("POST /", s.handleJSONRPC)
	return s, nil
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.Mux.ServeHTTP(w, r)
}

// handleJSONRPC dispatches JSON-RPC 2.0 requests to the appropriate handler.
func (s *Server) handleJSONRPC(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONRPCError(w, a2a.ErrParseError(nil, "failed to read request body"))
		return
	}

	var req a2a.JSONRPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONRPCError(w, a2a.ErrParseError(nil, err.Error()))
		return
	}

	if req.JSONRPC != "2.0" {
		writeJSONRPCError(w, a2a.ErrInvalidRequest(req.ID, "jsonrpc must be \"2.0\""))
		return
	}

	s.Logger.Info("JSON-RPC request", "method", req.Method, "id", string(req.ID))

	switch req.Method {
	case "SendMessage":
		s.handleSendMessage(w, r, req)
	case "SendStreamingMessage":
		s.handleSendStreamingMessage(w, r, req)
	case "GetTask":
		s.handleGetTask(w, r, req)
	case "CancelTask":
		s.handleCancelTask(w, r, req)
	case "ListTasks":
		s.handleListTasks(w, r, req)
	default:
		writeJSONRPCError(w, a2a.ErrMethodNotFound(req.ID, req.Method))
	}
}

// ensureWorkspace creates or reuses the workspace for this bridge.
func (s *Server) ensureWorkspace(r *http.Request) (string, error) {
	if s.workspaceID != "" {
		return s.workspaceID, nil
	}

	ws, err := s.Crush.CreateWorkspace(r.Context(), s.WorkspacePath)
	if err != nil {
		return "", err
	}

	s.Logger.Info("workspace created", "id", ws.ID, "path", ws.Path)

	if err := s.Crush.SkipPermissions(r.Context(), ws.ID); err != nil {
		s.Logger.Warn("failed to set skip permissions", "error", err)
	}

	if err := s.Crush.InitAgent(r.Context(), ws.ID); err != nil {
		s.Logger.Warn("failed to init agent", "error", err)
	}

	s.workspaceID = ws.ID
	return ws.ID, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeJSONRPCResponse(w http.ResponseWriter, resp *a2a.JSONRPCResponse) {
	writeJSON(w, http.StatusOK, resp)
}

func writeJSONRPCError(w http.ResponseWriter, resp *a2a.JSONRPCError) {
	writeJSON(w, http.StatusOK, resp)
}

package server

import (
	"encoding/json"
	"net/http"

	"github.com/aleksclark/crush-a2a/internal/a2a"
)

func (s *Server) handleAgentCard(w http.ResponseWriter, r *http.Request) {
	card := a2a.AgentCard{
		Name:        "crush",
		Version:     "1.0.0",
		URL:         s.BaseURL,
		Description: "Crush AI assistant exposed via A2A v1.0 protocol",
		Capabilities: &a2a.Capabilities{
			Streaming:              true,
			PushNotifications:      false,
			StateTransitionHistory: false,
		},
		Skills: []a2a.Skill{
			{
				ID:          "general",
				Name:        "General Assistant",
				Description: "General-purpose AI assistant powered by Crush",
				Tags:        []string{"general", "coding", "assistant"},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(card)
}

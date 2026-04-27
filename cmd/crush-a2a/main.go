package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/aleksclark/crush-a2a/internal/server"
)

func main() {
	port := flag.Int("port", 8200, "HTTP listen port")
	acpURL := flag.String("acp-url", "http://localhost:8199", "Crush ACP backend URL")
	agentName := flag.String("agent-name", "crush", "ACP agent name to proxy")
	verbose := flag.Bool("v", false, "Enable debug logging")
	flag.Parse()

	level := slog.LevelInfo
	if *verbose {
		level = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	}))

	baseURL := fmt.Sprintf("http://localhost:%d", *port)

	srv := server.New(server.Config{
		Port:      *port,
		ACPURL:    *acpURL,
		AgentName: *agentName,
		BaseURL:   baseURL,
		Logger:    logger,
	})

	addr := fmt.Sprintf(":%d", *port)
	logger.Info("starting crush-a2a server",
		"addr", addr,
		"acp_url", *acpURL,
		"agent_name", *agentName,
	)

	if err := http.ListenAndServe(addr, srv); err != nil {
		logger.Error("server failed", "error", err)
		os.Exit(1)
	}
}

package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2asrv"
	"github.com/aleksclark/crush-a2a/internal/crush"
	"github.com/aleksclark/crush-a2a/internal/executor"
)

func main() {
	port := flag.Int("port", 8200, "HTTP listen port")
	crushAddr := flag.String("crush-addr", "tcp://localhost:19200", "Crush server address (tcp://host:port or unix:///path)")
	workspacePath := flag.String("workspace-path", "/tmp/crush-a2a-workspace", "Workspace directory path")
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

	client, err := crush.NewClient(*crushAddr, logger)
	if err != nil {
		logger.Error("failed to create crush client", "error", err)
		os.Exit(1)
	}

	exec := &executor.CrushExecutor{
		Crush:         client,
		WorkspacePath: *workspacePath,
		Logger:        logger,
	}

	capabilities := a2a.AgentCapabilities{Streaming: true}

	agentCard := &a2a.AgentCard{
		Name:        "crush",
		Version:     "1.0.0",
		Description: "Crush AI assistant exposed via A2A v1.0 protocol",
		Capabilities: capabilities,
		SupportedInterfaces: []*a2a.AgentInterface{
			a2a.NewAgentInterface(baseURL, a2a.TransportProtocolJSONRPC),
		},
		DefaultInputModes:  []string{"text"},
		DefaultOutputModes: []string{"text"},
		Skills: []a2a.AgentSkill{
			{
				ID:          "general",
				Name:        "General Assistant",
				Description: "General-purpose AI assistant powered by Crush",
				Tags:        []string{"general", "coding", "assistant"},
			},
		},
	}

	requestHandler := a2asrv.NewHandler(exec,
		a2asrv.WithLogger(logger),
		a2asrv.WithCapabilityChecks(&capabilities),
	)

	mux := http.NewServeMux()
	mux.Handle("/", a2asrv.NewJSONRPCHandler(requestHandler))
	mux.Handle(a2asrv.WellKnownAgentCardPath, a2asrv.NewStaticAgentCardHandler(agentCard))

	addr := fmt.Sprintf(":%d", *port)
	logger.Info("starting crush-a2a server",
		"addr", addr,
		"crush_addr", *crushAddr,
		"workspace_path", *workspacePath,
	)

	if err := http.ListenAndServe(addr, mux); err != nil {
		logger.Error("server failed", "error", err)
		os.Exit(1)
	}
}

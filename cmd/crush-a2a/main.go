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

	srv, err := server.New(server.Config{
		Port:          *port,
		CrushAddr:     *crushAddr,
		WorkspacePath: *workspacePath,
		BaseURL:       baseURL,
		Logger:        logger,
	})
	if err != nil {
		logger.Error("failed to create server", "error", err)
		os.Exit(1)
	}

	addr := fmt.Sprintf(":%d", *port)
	logger.Info("starting crush-a2a server",
		"addr", addr,
		"crush_addr", *crushAddr,
		"workspace_path", *workspacePath,
	)

	if err := http.ListenAndServe(addr, srv); err != nil {
		logger.Error("server failed", "error", err)
		os.Exit(1)
	}
}

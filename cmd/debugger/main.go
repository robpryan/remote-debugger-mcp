package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/robpryan/go-debugger-mcp/pkg/tools/delve"
	"github.com/rs/zerolog"
)

const (
	ServerName  = "go-debugger-mcp"
	ServiceName = "Go Debugger MCP Server"
)

func main() {
	var (
		debug    bool
		bindAddr string
	)
	flag.BoolVar(&debug, "debug", false, "debug mode")
	flag.StringVar(&bindAddr, "bind", "localhost:8899", "bind address (host:port)")

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		logger.Debug().Msg("debug mode enabled")
	}

	impl := &mcp.Implementation{
		Name:    ServerName,
		Version: "1.0",
	}

	srv := mcp.NewServer(impl, nil)

	// Register Delve tool
	delveTool := delve.New(logger)
	delveTool.Register(srv)

	// Create HTTP handler for MCP server
	// Use StreamableHTTPHandler for HTTP transport with SSE streaming (Claude Code compatible)
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return srv
	}, nil)

	http.Handle("/mcp", handler)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"service": ServiceName,
			"version": "1.0",
			"endpoints": map[string]string{
				"mcp": "/mcp",
			},
		})
	})

	logger.Info().Msgf("%s starting on address %s", ServiceName, bindAddr)
	logger.Info().Msgf("MCP endpoint available at: http://%s/mcp", bindAddr)

	go func() {
		if err := http.ListenAndServe(bindAddr, nil); !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal().Msgf("%s failed to start: %v", ServerName, err)
		}
	}()
	<-signalCtx.Done()
	logger.Info().Msgf("%s shutdown complete", ServiceName)
}

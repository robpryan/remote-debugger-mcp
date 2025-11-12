package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/robpryan/remote-debugger-mcp/pkg/server"
	"github.com/robpryan/remote-debugger-mcp/pkg/tools"
	"github.com/robpryan/remote-debugger-mcp/pkg/tools/delve"
	"github.com/rs/zerolog"
)

const (
	ServerName      = "remote-debugger-mcp"
	ServiceName     = "Remote Debugger MCP Connector"
	ShutdownTimeout = 10 * time.Second
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

	srv := server.NewServer(impl)
	toolList := []tools.Tool{
		delve.New(logger),
	}
	// Register all tools
	for _, tool := range toolList {
		tool.Register(srv)
	}
	// Create HTTP handler for MCP server
	// Use StreamableHTTPHandler for HTTP transport with SSE streaming (Claude Code compatible)
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return &srv.Server
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
	ctx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
	defer cancel()
	// Shutdown MCP server
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error().Msgf("%s shutdown error: %v", ServiceName, err)
	} else {
		logger.Info().Msgf("%s shutdown complete", ServiceName)
	}
}

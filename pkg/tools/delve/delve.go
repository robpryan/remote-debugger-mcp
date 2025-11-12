package delve

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/robpryan/go-debugger-mcp/pkg/tools/delve/client"
	"github.com/robpryan/go-debugger-mcp/pkg/types"
	"github.com/rs/zerolog"
)

const (
	cleanupInterval    = 5 * time.Minute
	sessionMaxIdleTime = 30 * time.Minute
)

type Input struct {
	Host      string `json:"host,omitempty" validate:"omitempty,hostname|ip"`
	Port      int    `json:"port,omitempty" validate:"min=0,max=65535"`
	Command   string `json:"command,omitempty" validate:"max=4096"`
	SessionID string `json:"session_id,omitempty" validate:"omitempty,max=64"`                       // Session ID for persistent connections
	Action    string `json:"action,omitempty" validate:"omitempty,oneof=connect disconnect command"` // Action: connect, disconnect, or command (default: command)
	MaxLines  int    `json:"max_lines,omitempty" validate:"min=0,max=100000"`                        // Maximum lines to return (default: 1000)
	Offset    int    `json:"offset,omitempty" validate:"min=0"`                                      // Line offset for pagination
}

type Output struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Command    string `json:"command,omitempty"`
	SessionID  string `json:"session_id,omitempty"`
	Action     string `json:"action,omitempty"`
	Output     string `json:"output"`
	TotalLines int    `json:"total_lines"`
	Offset     int    `json:"offset"`
	MaxLines   int    `json:"max_lines"`
	Truncated  bool   `json:"truncated"`
	Status     string `json:"status"` // Session status: connected, disconnected, command_executed
}

// DelveSession represents a persistent Delve debugger API connection.
type DelveSession struct {
	client   *client.Client
	host     string
	port     int
	mu       sync.Mutex
	lastUsed time.Time
}

type Tool struct {
	logger    zerolog.Logger
	validator *validator.Validate
	sessions  map[string]*DelveSession
	sessionMu sync.RWMutex
}

func (d *Tool) DelveHandler(ctx context.Context, req *mcp.CallToolRequest, params *Input) (*mcp.CallToolResult, any, error) {
	// Validate input using validator
	if err := d.validator.Struct(params); err != nil {
		return nil, nil, fmt.Errorf("validation error: %w", err)
	}

	host := "localhost"
	if params.Host != "" {
		host = params.Host
	}

	port := 2345
	if params.Port != 0 {
		port = params.Port
	}

	action := "command"
	if params.Action != "" {
		action = params.Action
	}

	// Fallback to generated session ID if no session_id provided
	if params.SessionID == "" {
		params.SessionID = fmt.Sprintf("session-%d", time.Now().UnixNano())
	}

	// Non-session mode (backward compatibility)
	result, data, err := d.handleSessionOperation(ctx, *params, host, port, action)
	if err != nil {
		d.logger.Error().Err(err).Msgf("DelveHandler returning error to MCP for session %s", params.SessionID)
	} else {
		d.logger.Debug().Msgf("DelveHandler returning success to MCP for session %s", params.SessionID)
	}
	return result, data, err
}

// Register registers the Delve tool with the MCP server.
func (d *Tool) Register(srv *mcp.Server) {
	delveTool := &mcp.Tool{
		Name: "delve",
		Description: `Interactive Go debugger using Delve API with persistent session support.

WORKFLOW:
1. Connect: Use action="connect" with session_id to establish a persistent debugging session
2. Debug: Execute commands using action="command" with the same session_id
3. Disconnect: Use action="disconnect" when done (optional - sessions auto-timeout after 30min)

AVAILABLE COMMANDS:
Execution Control:
  continue, c      - Resume execution until next breakpoint
  next, n          - Step over to next source line
  step, s          - Step into function calls
  stepout, so      - Step out of current function
  halt             - Pause execution (required before inspection when program is running)
  restart, r       - Restart the program

Breakpoints:
  break, b <location>  - Set breakpoint (e.g., "main.main" or "file.go:42")
  breakpoints, bp      - List all breakpoints
  clear <id>           - Remove breakpoint by ID
  clearall             - Remove all breakpoints

Variable Inspection:
  print, p <expr>  - Evaluate and print expression
  locals           - Show local variables
  args             - Show function arguments
  vars [regex]     - Show package variables (optionally filtered)
  whatis <expr>    - Show type of expression
  set <var>=<val>  - Modify variable value

Stack & Context:
  stack, bt [depth]    - Show stack trace (default: 10 frames)
  frame <n>            - Switch to stack frame N
  list, ls, l [loc]    - Show source code

Goroutines:
  goroutines, grs [filter]  - List goroutines (filter: all, user, runtime)
  goroutine, gr <id>        - Switch to goroutine ID
  goroutine, gr <id> <cmd>  - Run command in goroutine context

Information:
  funcs [regex]    - List functions (optionally filtered)
  types [regex]    - List types (optionally filtered)
  sources [regex]  - List source files (optionally filtered)

PARAMETERS:
  session_id  - Required. Unique ID for persistent session (string)
  action      - "connect", "disconnect", or "command" (default: "command")
  host        - Delve server host (default: "localhost")
  port        - Delve server port (default: 2345)
  command     - Command to execute (required for action="command")
  max_lines   - Maximum output lines to return (default: 1000)
  offset      - Line offset for paginated output (default: 0)

IMPORTANT NOTES:
- Sessions persist across multiple calls - reuse the same session_id
- The debugger uses AutoHalt mode: program is automatically halted before inspection
- When program is running, "halt" before using inspection commands
- Use unique session_id per debugging target
- Sessions automatically cleanup after 30 minutes of inactivity

EXAMPLES:
1. Connect: {"session_id": "debug-1", "action": "connect", "port": 2345}
2. Set breakpoint: {"session_id": "debug-1", "action": "command", "command": "break main.main"}
3. Continue: {"session_id": "debug-1", "action": "command", "command": "continue"}
4. Inspect locals: {"session_id": "debug-1", "action": "command", "command": "locals"}
5. Disconnect: {"session_id": "debug-1", "action": "disconnect"}`,
	}

	mcp.AddTool(srv, delveTool, d.DelveHandler)
	d.logger.Debug().Msg("delve tool registered")

	// Start cleanup goroutine for stale sessions
	go d.cleanupStaleSessions()
}

// New creates a new Delve tool instance.
func New(logger zerolog.Logger) *Tool {
	validate := validator.New()

	return &Tool{
		logger:    logger.With().Str("tool", "delve").Logger(),
		validator: validate,
		sessions:  make(map[string]*DelveSession),
	}
}

// cleanupStaleSessions removes sessions that haven't been used for 30 minutes.
func (d *Tool) cleanupStaleSessions() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		d.sessionMu.Lock()
		now := time.Now()
		for sessionID, session := range d.sessions {
			if now.Sub(session.lastUsed) > sessionMaxIdleTime {
				d.logger.Info().Msgf("Cleaning up stale session %s", sessionID)
				// Properly cleanup the session
				d.cleanupSession(session)
				delete(d.sessions, sessionID)
			}
		}
		d.sessionMu.Unlock()
	}
}

// connectSession creates a new Delve API session.
func (d *Tool) connectSession(ctx context.Context, sessionID, host string, port int) (*DelveSession, error) {
	addr := fmt.Sprintf("%s:%d", host, port)
	d.logger.Info().Msgf("Creating new Delve API session %s at %s", sessionID, addr)

	// Create client with AutoHalt enabled (recommended for --continue mode)
	apiClient, err := client.NewClient(client.Config{
		Host:     host,
		Port:     port,
		AutoHalt: true, // Automatically halt before inspection operations
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Delve client: %w", err)
	}

	// Test the connection with a timeout
	// Use a channel to make the blocking GetState() call timeout-aware
	type stateResult struct {
		err error
	}
	resultCh := make(chan stateResult, 1)

	d.logger.Debug().Msgf("Starting connection test goroutine for %s", addr)
	go func() {
		d.logger.Debug().Msgf("Calling GetStateNonBlocking() for %s", addr)
		_, err := apiClient.GetStateNonBlocking()
		d.logger.Debug().Msgf("GetStateNonBlocking() returned for %s: err=%v", addr, err)
		resultCh <- stateResult{err}
	}()

	// Wait for connection test with 5 second timeout
	d.logger.Debug().Msgf("Waiting for connection test with 5s timeout for %s", addr)
	select {
	case result := <-resultCh:
		d.logger.Debug().Msgf("Received result from GetState() for %s", addr)
		if result.err != nil {
			d.logger.Error().Err(result.err).Msgf("GetState() failed for %s", addr)
			// Close in background to avoid blocking
			go func() {
				if err := apiClient.Close(); err != nil {
					d.logger.Debug().Err(err).Msg("Error closing failed client")
				}
			}()
			return nil, fmt.Errorf("failed to connect to Delve server at %s: %w", addr, result.err)
		}
		d.logger.Debug().Msgf("GetState() succeeded for %s", addr)
	case <-time.After(5 * time.Second):
		d.logger.Warn().Msgf("Connection to %s timed out after 5 seconds", addr)
		// Close in background to avoid blocking - the connection might be hanging
		go func() {
			if err := apiClient.Close(); err != nil {
				d.logger.Debug().Err(err).Msg("Error closing timed-out client (this is expected)")
			}
		}()
		timeoutErr := fmt.Errorf("connection to Delve server at %s timed out after 5 seconds", addr)
		d.logger.Error().Err(timeoutErr).Msg("Returning timeout error from connectSession")
		return nil, timeoutErr
	case <-ctx.Done():
		d.logger.Warn().Msgf("Connection to %s cancelled", addr)
		// Close in background to avoid blocking
		go func() {
			if err := apiClient.Close(); err != nil {
				d.logger.Debug().Err(err).Msg("Error closing cancelled client")
			}
		}()
		return nil, fmt.Errorf("connection cancelled: %w", ctx.Err())
	}

	session := &DelveSession{
		client:   apiClient,
		host:     host,
		port:     port,
		lastUsed: time.Now(),
	}

	d.logger.Info().Msgf("Successfully connected to Delve API at %s", addr)
	return session, nil
}

// executeCommand executes a command via the API client.
func (d *Tool) executeCommand(session *DelveSession, command string) (string, error) {
	session.mu.Lock()
	defer session.mu.Unlock()

	session.lastUsed = time.Now()

	d.logger.Debug().Str("command", command).Msg("Executing command via API")

	// Parse command and arguments
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "", fmt.Errorf("empty command")
	}

	cmdName := parts[0]
	args := parts[1:]

	// Execute command via API client
	output, err := session.client.ExecuteCommand(cmdName, args)
	if err != nil {
		d.logger.Error().Err(err).Str("command", cmdName).Msg("Command execution failed")
		return "", fmt.Errorf("command failed: %w", err)
	}

	d.logger.Debug().Str("command", cmdName).Msg("Command executed successfully")
	return output, nil
}

// disconnectSession closes a Delve session.
func (d *Tool) disconnectSession(sessionID string) error {
	d.sessionMu.Lock()
	defer d.sessionMu.Unlock()

	session, exists := d.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	// Properly cleanup the session
	d.cleanupSession(session)

	delete(d.sessions, sessionID)
	d.logger.Info().Msgf("Disconnected Delve session %s", sessionID)
	return nil
}

// cleanupSession properly terminates a Delve API session.
func (d *Tool) cleanupSession(session *DelveSession) {
	if session == nil {
		return
	}

	// Close the API client connection
	if session.client != nil {
		if err := session.client.Close(); err != nil {
			d.logger.Error().Err(err).Msg("Failed to close Delve API client")
		}
	}
}

// handleSessionOperation handles session-based operations.
func (d *Tool) handleSessionOperation(ctx context.Context, input Input, host string, port int, action string) (*mcp.CallToolResult, any, error) {
	switch action {
	case "connect":
		return d.handleConnect(ctx, input, host, port)
	case "disconnect":
		return d.handleDisconnect(input)
	case "command":
		return d.handleCommand(input)
	default:
		return nil, nil, fmt.Errorf("unsupported action: %s. Use 'connect', 'disconnect', or 'command'", action)
	}
}

// handleConnect creates a new Delve session.
func (d *Tool) handleConnect(ctx context.Context, input Input, host string, port int) (*mcp.CallToolResult, any, error) {
	d.sessionMu.Lock()
	if _, exists := d.sessions[input.SessionID]; exists {
		d.sessionMu.Unlock()
		return nil, nil, fmt.Errorf("session %s already exists", input.SessionID)
	}

	session, err := d.connectSession(ctx, input.SessionID, host, port)
	if err != nil {
		d.sessionMu.Unlock()
		d.logger.Error().Err(err).Msgf("Failed to connect session %s, returning error to client", input.SessionID)
		return nil, nil, err
	}

	d.sessions[input.SessionID] = session
	d.sessionMu.Unlock()

	resultText := fmt.Sprintf("Connected to Delve debugger at %s:%d\nSession ID: %s\nSession established. Use 'command' action to send debugging commands.", host, port, input.SessionID)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
	}, nil, nil
}

// handleDisconnect disconnects a Delve session.
func (d *Tool) handleDisconnect(input Input) (*mcp.CallToolResult, any, error) {
	if err := d.disconnectSession(input.SessionID); err != nil {
		return nil, nil, err
	}

	resultText := "Disconnected Delve session: " + input.SessionID

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
	}, nil, nil
}

// handleCommand executes a command in an existing session.
func (d *Tool) handleCommand(input Input) (*mcp.CallToolResult, any, error) {
	d.sessionMu.RLock()
	session, exists := d.sessions[input.SessionID]
	d.sessionMu.RUnlock()

	if !exists {
		return nil, nil, fmt.Errorf("session %s not found. Use 'connect' action first", input.SessionID)
	}

	command := "help"
	if input.Command != "" {
		command = input.Command
	}

	output, err := d.executeCommand(session, command)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute command: %w", err)
	}

	// Apply pagination
	maxLines := types.MaxDefaultLines
	if input.MaxLines > 0 {
		maxLines = input.MaxLines
	}

	offset := 0
	if input.Offset > 0 {
		offset = input.Offset
	}

	lines := strings.Split(output, "\n")
	totalLines := len(lines)

	// Apply offset and limit
	truncated := false
	if offset < totalLines {
		end := offset + maxLines
		if end > totalLines {
			end = totalLines
		} else {
			truncated = true
		}
		lines = lines[offset:end]
	} else {
		lines = []string{}
	}

	paginatedOutput := strings.Join(lines, "\n")

	resultText := fmt.Sprintf("Session %s - Command: %s\n", input.SessionID, command)
	if truncated {
		resultText += fmt.Sprintf("[Showing lines %d-%d of %d total lines. Use offset parameter to view more.]\n", offset+1, offset+len(lines), totalLines)
	}
	resultText += "\n" + strings.TrimSpace(paginatedOutput)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
	}, nil, nil
}

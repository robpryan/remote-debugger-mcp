package delve

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog"
	"github.com/tb0hdan/remote-debugger-mcp/pkg/server"
	"github.com/tb0hdan/remote-debugger-mcp/pkg/tools"
	"github.com/tb0hdan/remote-debugger-mcp/pkg/types"
)

const (
	sessionStartupDelay = 500 * time.Millisecond
	commandTimeout      = 5 * time.Second
	disconnectDelay     = 100 * time.Millisecond
	cleanupInterval     = 5 * time.Minute
	sessionMaxIdleTime  = 30 * time.Minute
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

// DelveSession represents a persistent Delve debugger connection.
type DelveSession struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	stderr    io.ReadCloser
	reader    *bufio.Reader
	outputCh  chan string // Broadcast channel for output chunks
	host      string
	port      int
	mu        sync.Mutex
	lastUsed  time.Time
	ctxCancel context.CancelFunc // To cancel the background context
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
	return d.handleSessionOperation(ctx, *params, host, port, action)
}

func (d *Tool) Register(srv *server.Server) {
	delveTool := &mcp.Tool{
		Name:        "delve",
		Description: "Connects to a remote Delve debugger with session support for interactive debugging",
	}

	mcp.AddTool(&srv.Server, delveTool, d.DelveHandler)
	d.logger.Debug().Msg("delve tool registered")

	// Start cleanup goroutine for stale sessions
	go d.cleanupStaleSessions()
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

// connectSession creates a new Delve session.
func (d *Tool) connectSession(_ context.Context, sessionID, host string, port int) (*DelveSession, error) {
	addr := fmt.Sprintf("%s:%d", host, port)
	d.logger.Info().Msgf("Creating new Delve session %s at %s", sessionID, addr)

	// Create a background context that won't be cancelled when the calling context ends
	// Using context.WithoutCancel would be better but requires Go 1.21+
	backgroundCtx, cancel := context.WithCancel(context.Background())

	// Create command with the background context
	cmd := exec.CommandContext(backgroundCtx, "dlv", "connect", addr) //nolint:contextcheck // intentionally using a detached context for background process

	// Set process group ID so we can kill the entire group
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start dlv command: %w", err)
	}

	session := &DelveSession{
		cmd:       cmd,
		stdin:     stdin,
		stdout:    stdout,
		stderr:    stderr,
		reader:    bufio.NewReader(stdout),
		outputCh:  make(chan string, 100), // Buffered channel for output
		host:      host,
		port:      port,
		lastUsed:  time.Now(),
		ctxCancel: cancel,
	}

	// Start a goroutine to reap the process when it exits (prevents zombies)
	go func() {
		_ = cmd.Wait()
	}()

	// Start persistent reader goroutine that broadcasts output
	go func() {
		defer close(session.outputCh)
		buf := make([]byte, 1024)
		for {
			n, err := session.reader.Read(buf)
			if n > 0 {
				chunk := string(buf[:n])
				d.logger.Debug().Str("chunk", chunk).Msg("Reader goroutine read chunk")
				session.outputCh <- chunk
			}
			if err != nil {
				if err != io.EOF {
					d.logger.Error().Err(err).Msg("Reader goroutine error")
				}
				d.logger.Debug().Msg("Reader goroutine exiting")
				return
			}
		}
	}()

	// Read initial prompt from output channel
	d.logger.Debug().Msg("Reading initial prompt from output channel")
	var initialOutput strings.Builder
	timeout := time.After(sessionStartupDelay * 2)

	for {
		select {
		case chunk, ok := <-session.outputCh:
			if !ok {
				return nil, errors.New("output channel closed before prompt received")
			}
			initialOutput.WriteString(chunk)
			d.logger.Debug().Str("output", chunk).Msg("Read initial output chunk")
			if strings.Contains(initialOutput.String(), "(dlv)") {
				d.logger.Debug().Msg("Found initial prompt")
				return session, nil
			}
		case <-timeout:
			d.logger.Warn().Msg("Timeout waiting for initial prompt, proceeding anyway")
			return session, nil
		}
	}
}

// executeCommand sends a command to an existing session and reads the response.
func (d *Tool) executeCommand(session *DelveSession, command string) (string, error) {
	session.mu.Lock()
	defer session.mu.Unlock()

	session.lastUsed = time.Now()

	d.logger.Debug().Str("command", command).Msg("Executing command")

	// Determine timeout based on command type
	cmdTimeout := commandTimeout // Default 5 seconds
	cmdWords := strings.Fields(command)
	if len(cmdWords) > 0 {
		switch cmdWords[0] {
		case "continue", "c", "next", "n", "step", "s", "stepout", "so":
			// Execution commands can take a long time or wait indefinitely
			cmdTimeout = 30 * time.Second
			d.logger.Debug().Msg("Using extended timeout for execution command")
		case "break", "b", "clear", "clearall":
			// Breakpoint commands sometimes take longer to respond
			cmdTimeout = 10 * time.Second
			d.logger.Debug().Msg("Using extended timeout for breakpoint command")
		}
	}

	// Send command
	if _, err := fmt.Fprintf(session.stdin, "%s\n", command); err != nil {
		d.logger.Error().Err(err).Msg("Failed to send command")
		return "", fmt.Errorf("failed to send command: %w", err)
	}
	d.logger.Debug().Msg("Command sent successfully")

	// Read response from output channel with grace period after prompt
	d.logger.Debug().Msg("Reading response from output channel")
	var output strings.Builder
	chunkCount := 0
	promptFound := false
	skippedInitialPrompt := false         // Skip first prompt echo from previous command
	gracePeriod := 500 * time.Millisecond // Grace period for buffered output
	var graceTimer *time.Timer
	commandTimer := time.NewTimer(cmdTimeout)
	defer commandTimer.Stop()

	hasNonPromptOutput := false

	for {
		var timeout <-chan time.Time
		if promptFound && graceTimer != nil {
			timeout = graceTimer.C
		}

		select {
		case chunk, ok := <-session.outputCh:
			if !ok {
				d.logger.Error().Msg("Output channel closed unexpectedly")
				return output.String(), errors.New("output channel closed")
			}

			chunkCount++
			d.logger.Debug().Int("chunk_num", chunkCount).Str("chunk", chunk).Str("chunk_bytes", fmt.Sprintf("%q", chunk)).Msg("Read chunk")

			trimmed := strings.TrimSpace(chunk)

			// Skip the first prompt we see (it's usually the echo from the previous command)
			if !skippedInitialPrompt && chunkCount == 1 && (trimmed == "(dlv)" || trimmed == ">") {
				d.logger.Debug().Msg("Skipping initial prompt echo from previous command")
				skippedInitialPrompt = true
				continue // Don't add this to output
			}

			output.WriteString(chunk)

			// Track if we have non-prompt output
			if trimmed != "" && trimmed != "(dlv)" && trimmed != ">" {
				hasNonPromptOutput = true
				d.logger.Debug().Str("trimmed", trimmed).Msg("Received non-prompt output")
			} else if trimmed != "" {
				d.logger.Debug().Str("trimmed", trimmed).Msg("Received prompt")
			}

			// Check for prompt indicating command completion
			// Only start grace period if we've seen actual output, not just the initial prompt
			if !promptFound && hasNonPromptOutput && (strings.Contains(output.String(), "(dlv)") || strings.Contains(output.String(), ">")) {
				d.logger.Debug().Msg("Found prompt after output, starting grace period")
				promptFound = true
				graceTimer = time.NewTimer(gracePeriod)
			} else if promptFound && graceTimer != nil {
				// Reset grace timer if we got more data
				d.logger.Debug().Msg("Got more data after prompt, resetting grace timer")
				if !graceTimer.Stop() {
					<-graceTimer.C
				}
				graceTimer.Reset(gracePeriod)
			}

		case <-timeout:
			d.logger.Debug().Msg("Grace period expired after prompt, finishing")
			return output.String(), nil

		case <-commandTimer.C:
			d.logger.Warn().Dur("timeout", cmdTimeout).Msg("Command timed out")
			return output.String(), errors.New("command timed out")
		}
	}
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

// cleanupSession properly terminates a Delve session and prevents zombie processes.
func (d *Tool) cleanupSession(session *DelveSession) {
	if session == nil {
		return
	}

	// Try to send exit command first for graceful shutdown
	if session.stdin != nil {
		_, _ = fmt.Fprintf(session.stdin, "exit\n")
		time.Sleep(disconnectDelay)
		_ = session.stdin.Close()
	}

	// Close stdout and stderr pipes
	if session.stdout != nil {
		_ = session.stdout.Close()
	}
	if session.stderr != nil {
		_ = session.stderr.Close()
	}

	// Kill the process group to ensure all child processes are terminated
	if session.cmd != nil && session.cmd.Process != nil {
		pgid := session.cmd.Process.Pid
		// Try SIGTERM first for graceful shutdown
		if err := syscall.Kill(-pgid, syscall.SIGTERM); err != nil {
			// If SIGTERM fails, try SIGKILL
			if killErr := syscall.Kill(-pgid, syscall.SIGKILL); killErr != nil {
				// Log the error but don't fail if the process is already gone
				if !strings.Contains(killErr.Error(), "no such process") {
					d.logger.Error().Err(killErr).Msg("Failed to kill Delve process group")
				}
			}
		}
	}

	// Cancel the background context
	if session.ctxCancel != nil {
		session.ctxCancel()
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

func New(logger zerolog.Logger) tools.Tool {
	validate := validator.New()

	return &Tool{
		logger:    logger.With().Str("tool", "delve").Logger(),
		validator: validate,
		sessions:  make(map[string]*DelveSession),
	}
}

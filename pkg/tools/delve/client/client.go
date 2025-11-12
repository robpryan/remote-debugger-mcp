package client

import (
	"fmt"

	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
)

// Client wraps the Delve rpc2.RPCClient with session-aware functionality.
// It tracks the current goroutine/frame context and halted state.
type Client struct {
	rpcClient        *rpc2.RPCClient
	currentGoroutine int64
	currentFrame     int
	halted           bool
	autoHalt         bool // Auto-halt before inspection operations
	host             string
	port             int
}

// Config holds configuration for creating a new Delve client.
type Config struct {
	Host     string
	Port     int
	AutoHalt bool // Automatically halt before inspection operations (recommended for --continue mode)
}

// NewClient creates a new Delve API client.
// It connects to the Delve server at host:port using the rpc2 protocol.
func NewClient(config Config) (*Client, error) {
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)

	// Use rpc2.NewClient for direct connection
	rpcClient := rpc2.NewClient(addr)

	client := &Client{
		rpcClient:        rpcClient,
		currentGoroutine: -1, // -1 means current goroutine
		currentFrame:     0,
		halted:           false,
		autoHalt:         config.AutoHalt,
		host:             config.Host,
		port:             config.Port,
	}

	return client, nil
}

// Halt stops the program execution.
// This is required before calling inspection methods when the program is running.
func (c *Client) Halt() (*api.DebuggerState, error) {
	state, err := c.rpcClient.Halt()
	if err != nil {
		return nil, fmt.Errorf("halt failed: %w", err)
	}
	c.halted = true
	return state, nil
}

// Continue resumes program execution.
// This is non-blocking and returns immediately.
func (c *Client) Continue() (*api.DebuggerState, error) {
	stateCh := c.rpcClient.Continue()
	c.halted = false
	// The channel will eventually receive the state when program stops,
	// but we return nil here since Continue is non-blocking
	_ = stateCh
	return nil, nil
}

// GetState returns the current debugger state.
// IMPORTANT: This blocks when the program is running until it stops.
// Only call this when you expect the program to be stopped (halted or at breakpoint).
func (c *Client) GetState() (*api.DebuggerState, error) {
	state, err := c.rpcClient.GetState()
	if err != nil {
		return nil, fmt.Errorf("get state failed: %w", err)
	}
	return state, nil
}

// GetStateNonBlocking returns the current debugger state without blocking.
// Unlike GetState(), this returns immediately even if the program is running.
func (c *Client) GetStateNonBlocking() (*api.DebuggerState, error) {
	state, err := c.rpcClient.GetStateNonBlocking()
	if err != nil {
		return nil, fmt.Errorf("get state non-blocking failed: %w", err)
	}
	return state, nil
}

// ensureHalted ensures the program is halted before inspection operations.
// This is called automatically if AutoHalt is enabled.
func (c *Client) ensureHalted() error {
	if c.halted {
		return nil
	}

	if c.autoHalt {
		_, err := c.Halt()
		return err
	}

	return fmt.Errorf("program is not halted; call Halt() first or enable AutoHalt")
}

// CreateBreakpoint creates a new breakpoint at the specified location.
func (c *Client) CreateBreakpoint(bp *api.Breakpoint) (*api.Breakpoint, error) {
	if err := c.ensureHalted(); err != nil {
		return nil, err
	}

	createdBp, err := c.rpcClient.CreateBreakpoint(bp)
	if err != nil {
		return nil, fmt.Errorf("create breakpoint failed: %w", err)
	}
	return createdBp, nil
}

// ListBreakpoints returns all breakpoints.
func (c *Client) ListBreakpoints() ([]*api.Breakpoint, error) {
	if err := c.ensureHalted(); err != nil {
		return nil, err
	}

	bps, err := c.rpcClient.ListBreakpoints(false)
	if err != nil {
		return nil, fmt.Errorf("list breakpoints failed: %w", err)
	}
	return bps, nil
}

// ClearBreakpoint removes a breakpoint by ID.
func (c *Client) ClearBreakpoint(id int) (*api.Breakpoint, error) {
	if err := c.ensureHalted(); err != nil {
		return nil, err
	}

	bp, err := c.rpcClient.ClearBreakpoint(id)
	if err != nil {
		return nil, fmt.Errorf("clear breakpoint failed: %w", err)
	}
	return bp, nil
}

// ListGoroutines returns a list of goroutines.
func (c *Client) ListGoroutines(start, count int) ([]*api.Goroutine, int, error) {
	if err := c.ensureHalted(); err != nil {
		return nil, 0, err
	}

	grs, nextStart, err := c.rpcClient.ListGoroutines(start, count)
	if err != nil {
		return nil, 0, fmt.Errorf("list goroutines failed: %w", err)
	}
	return grs, nextStart, nil
}

// Stacktrace returns the stack trace for a goroutine.
// Use goroutineID = -1 for the current goroutine.
func (c *Client) Stacktrace(goroutineID int64, depth int, opts api.StacktraceOptions) ([]api.Stackframe, error) {
	if err := c.ensureHalted(); err != nil {
		return nil, err
	}

	frames, err := c.rpcClient.Stacktrace(goroutineID, depth, opts, nil)
	if err != nil {
		return nil, fmt.Errorf("stacktrace failed: %w", err)
	}
	return frames, nil
}

// ListLocalVariables returns local variables in the current scope.
func (c *Client) ListLocalVariables(scope api.EvalScope, cfg api.LoadConfig) ([]api.Variable, error) {
	if err := c.ensureHalted(); err != nil {
		return nil, err
	}

	vars, err := c.rpcClient.ListLocalVariables(scope, cfg)
	if err != nil {
		return nil, fmt.Errorf("list local variables failed: %w", err)
	}
	return vars, nil
}

// ListFunctionArgs returns function arguments in the current scope.
func (c *Client) ListFunctionArgs(scope api.EvalScope, cfg api.LoadConfig) ([]api.Variable, error) {
	if err := c.ensureHalted(); err != nil {
		return nil, err
	}

	vars, err := c.rpcClient.ListFunctionArgs(scope, cfg)
	if err != nil {
		return nil, fmt.Errorf("list function args failed: %w", err)
	}
	return vars, nil
}

// Eval evaluates an expression in the current scope.
func (c *Client) Eval(scope api.EvalScope, expr string, cfg *api.LoadConfig) (*api.Variable, error) {
	if err := c.ensureHalted(); err != nil {
		return nil, err
	}

	variable, err := c.rpcClient.EvalVariable(scope, expr, *cfg)
	if err != nil {
		return nil, fmt.Errorf("eval failed: %w", err)
	}
	return variable, nil
}

// SetVariable sets a variable to a new value.
func (c *Client) SetVariable(scope api.EvalScope, symbol, value string) error {
	if err := c.ensureHalted(); err != nil {
		return err
	}

	err := c.rpcClient.SetVariable(scope, symbol, value)
	if err != nil {
		return fmt.Errorf("set variable failed: %w", err)
	}
	return nil
}

// ListFunctions returns functions matching a filter.
func (c *Client) ListFunctions(filter string, followCalls int) ([]string, error) {
	if err := c.ensureHalted(); err != nil {
		return nil, err
	}

	funcs, err := c.rpcClient.ListFunctions(filter, followCalls)
	if err != nil {
		return nil, fmt.Errorf("list functions failed: %w", err)
	}
	return funcs, nil
}

// ListTypes returns types matching a filter.
func (c *Client) ListTypes(filter string) ([]string, error) {
	if err := c.ensureHalted(); err != nil {
		return nil, err
	}

	types, err := c.rpcClient.ListTypes(filter)
	if err != nil {
		return nil, fmt.Errorf("list types failed: %w", err)
	}
	return types, nil
}

// ListSources returns source files matching a filter.
func (c *Client) ListSources(filter string) ([]string, error) {
	if err := c.ensureHalted(); err != nil {
		return nil, err
	}

	sources, err := c.rpcClient.ListSources(filter)
	if err != nil {
		return nil, fmt.Errorf("list sources failed: %w", err)
	}
	return sources, nil
}

// Next steps to the next source line.
func (c *Client) Next() (*api.DebuggerState, error) {
	state, err := c.rpcClient.Next()
	if err != nil {
		return nil, fmt.Errorf("next failed: %w", err)
	}
	c.halted = true
	return state, nil
}

// Step steps into a function call.
func (c *Client) Step() (*api.DebuggerState, error) {
	state, err := c.rpcClient.Step()
	if err != nil {
		return nil, fmt.Errorf("step failed: %w", err)
	}
	c.halted = true
	return state, nil
}

// StepOut steps out of the current function.
func (c *Client) StepOut() (*api.DebuggerState, error) {
	state, err := c.rpcClient.StepOut()
	if err != nil {
		return nil, fmt.Errorf("step out failed: %w", err)
	}
	c.halted = true
	return state, nil
}

// Restart restarts the debugging session.
func (c *Client) Restart() (*api.DebuggerState, error) {
	// Restart returns discarded breakpoints, then new state via channel
	_, err := c.rpcClient.Restart(false)
	if err != nil {
		return nil, fmt.Errorf("restart failed: %w", err)
	}
	c.halted = true

	// Get the new state after restart
	state, err := c.GetState()
	if err != nil {
		return nil, fmt.Errorf("failed to get state after restart: %w", err)
	}
	return state, nil
}

// SetCurrentGoroutine sets the current goroutine context for subsequent operations.
func (c *Client) SetCurrentGoroutine(goroutineID int64) {
	c.currentGoroutine = goroutineID
	c.currentFrame = 0 // Reset frame when switching goroutines
}

// SetCurrentFrame sets the current frame context for subsequent operations.
func (c *Client) SetCurrentFrame(frame int) {
	c.currentFrame = frame
}

// GetScope returns the current evaluation scope based on session context.
func (c *Client) GetScope() api.EvalScope {
	return api.EvalScope{
		GoroutineID: c.currentGoroutine,
		Frame:       c.currentFrame,
	}
}

// Detach detaches from the debugging session.
// If kill is true, the debugged process will be killed.
func (c *Client) Detach(kill bool) error {
	err := c.rpcClient.Detach(kill)
	if err != nil {
		return fmt.Errorf("detach failed: %w", err)
	}
	return nil
}

// Close closes the connection to the Delve server.
func (c *Client) Close() error {
	// Detach first (without killing)
	_ = c.Detach(false)

	// The rpc2 client doesn't have an explicit Close method,
	// but Detach should handle cleanup
	return nil
}

// IsHalted returns whether the program is currently halted.
func (c *Client) IsHalted() bool {
	return c.halted
}

// GetConnectionInfo returns the connection details.
func (c *Client) GetConnectionInfo() (string, int) {
	return c.host, c.port
}
